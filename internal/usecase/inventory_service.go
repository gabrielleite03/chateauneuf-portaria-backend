package usecase

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

type InventoryError string

func (e InventoryError) Error() string { return string(e) }

const ErrInventoryStock InventoryError = "Estoque insuficiente para esta retirada. Atualize os saldos."
const ErrInventoryRequest InventoryError = "Esta operação já foi enviada com outros dados. Atualize a página."

type InventoryProduct struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Unit         string `json:"unit"`
	MinimumMilli int64  `json:"minimumMilli"`
	StockMilli   int64  `json:"stockMilli"`
	InitialMilli int64  `json:"initialMilli"`
	Deleted      bool   `json:"deleted"`
}
type InventoryPurchase struct {
	ID         string `json:"id"`
	Date       string `json:"date"`
	Supplier   string `json:"supplier"`
	Document   string `json:"document"`
	Notes      string `json:"notes"`
	TotalCents int64  `json:"totalCents"`
}
type InventoryMovement struct {
	ID            int64  `json:"id"`
	ProductID     string `json:"productId"`
	PurchaseID    string `json:"purchaseId"`
	Kind          string `json:"kind"`
	Date          string `json:"date"`
	QuantityMilli int64  `json:"quantityMilli"`
	UnitCostCents int64  `json:"unitCostCents"`
	TotalCents    int64  `json:"totalCents"`
	Responsible   string `json:"responsible"`
	Notes         string `json:"notes"`
}
type InventorySnapshot struct {
	Products  []InventoryProduct  `json:"products"`
	Purchases []InventoryPurchase `json:"purchases"`
	Movements []InventoryMovement `json:"movements"`
}
type InventoryProductInput struct {
	Password     string `json:"password,omitempty"`
	RequestID    string `json:"requestId"`
	Name         string `json:"name"`
	Unit         string `json:"unit"`
	MinimumMilli int64  `json:"minimumMilli"`
	InitialMilli int64  `json:"initialMilli"`
	Date         string `json:"date"`
	Responsible  string `json:"responsible"`
}
type InventoryPurchaseItem struct {
	ProductID     string `json:"productId"`
	QuantityMilli int64  `json:"quantityMilli"`
	UnitCostCents int64  `json:"unitCostCents"`
}
type InventoryPurchaseInput struct {
	RequestID string                  `json:"requestId"`
	Date      string                  `json:"date"`
	Supplier  string                  `json:"supplier"`
	Document  string                  `json:"document"`
	Notes     string                  `json:"notes"`
	Items     []InventoryPurchaseItem `json:"items"`
}
type InventoryWithdrawalInput struct {
	RequestID     string `json:"requestId"`
	ProductID     string `json:"productId"`
	Date          string `json:"date"`
	QuantityMilli int64  `json:"quantityMilli"`
	Responsible   string `json:"responsible"`
	Notes         string `json:"notes"`
}
type InventoryService struct {
	db               *sql.DB
	passwordVerifier string
}

func NewInventoryService(db *sql.DB) *InventoryService {
	return &InventoryService{db: db, passwordVerifier: inventoryPasswordVerifier}
}

var inventoryID = regexp.MustCompile(`^[a-zA-Z0-9-]{8,80}$`)

func inventoryText(s string, max int) bool { return len([]rune(strings.TrimSpace(s))) <= max }
func inventoryDate(s string) bool {
	d, err := time.Parse("2006-01-02", s)
	// The portaria uses Sao Paulo dates; a fixed UTC-3 offset avoids container tzdata dependencies.
	today := time.Now().In(time.FixedZone("Sao Paulo", -3*60*60)).Format("2006-01-02")
	return err == nil && d.Year() >= 2000 && s <= today
}

// Reserve the operation before changing stock: retries after a lost response cannot duplicate a purchase.
func (s *InventoryService) operation(ctx context.Context, id, action string, input any, apply func(*sql.Tx) error) error {
	if !inventoryID.MatchString(id) {
		return InventoryError("Identificação da operação inválida.")
	}
	data, err := json.Marshal(input)
	if err != nil {
		return err
	}
	fingerprint := fmt.Sprintf("%x", sha256.Sum256(append([]byte(action), data...)))
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `INSERT INTO inventory_requests(id,fingerprint) VALUES (?,?) ON CONFLICT(id) DO NOTHING`, id, fingerprint)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		var previous string
		if err := tx.QueryRowContext(ctx, `SELECT fingerprint FROM inventory_requests WHERE id=?`, id).Scan(&previous); err != nil {
			return err
		}
		if previous != fingerprint {
			return ErrInventoryRequest
		}
		return nil
	}
	if err := apply(tx); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *InventoryService) SaveProduct(ctx context.Context, id string, in InventoryProductInput) error {
	if id != "" {
		if err := s.authorize(in.Password); err != nil {
			return err
		}
	}
	in.Password = "" // Never retain credentials in idempotency fingerprints or audit data.
	in.Name = strings.Join(strings.Fields(in.Name), " ")
	in.Responsible = strings.TrimSpace(in.Responsible)
	validUnit := map[string]bool{"un": true, "frasco": true, "galão": true, "pacote": true, "caixa": true, "L": true, "kg": true}
	if in.Name == "" || !inventoryText(in.Name, 120) || !validUnit[in.Unit] || in.MinimumMilli < 0 || in.MinimumMilli > 100000000 || in.InitialMilli < 0 || in.InitialMilli > 100000000 || !inventoryText(in.Responsible, 120) {
		return InventoryError("Confira nome, unidade, estoque mínimo e saldo inicial do produto.")
	}
	if id == "" && in.InitialMilli > 0 && (!inventoryDate(in.Date) || in.Responsible == "") {
		return InventoryError("Informe data válida e responsável pela contagem inicial.")
	}
	return s.operation(ctx, in.RequestID, "product:"+id, in, func(tx *sql.Tx) error {
		var err error
		if id == "" {
			_, err = tx.ExecContext(ctx, `INSERT INTO inventory_products(id,name,name_key,unit,minimum_milli,stock_milli) VALUES (?,?,?,?,?,?)`, in.RequestID, in.Name, strings.ToLower(in.Name), in.Unit, in.MinimumMilli, in.InitialMilli)
		} else {
			previous, e := inventoryProductInTx(ctx, tx, id)
			if e != nil {
				return e
			}
			if previous.Deleted || previous.Unit != in.Unit {
				return InventoryError("Produto excluído ou unidade alterada.")
			}
			nextStock := previous.StockMilli + in.InitialMilli - previous.InitialMilli
			if nextStock < 0 || nextStock > 9000000000000 {
				return InventoryError("A quantidade inicial informada é incompatível com as compras e retiradas já registradas.")
			}
			_, err = tx.ExecContext(ctx, `UPDATE inventory_products SET name=?,name_key=?,minimum_milli=?,stock_milli=? WHERE id=? AND deleted_at=''`, in.Name, strings.ToLower(in.Name), in.MinimumMilli, nextStock, id)
			if err == nil {
				if in.InitialMilli != previous.InitialMilli {
					if in.InitialMilli == 0 {
						_, err = tx.ExecContext(ctx, `DELETE FROM inventory_movements WHERE product_id=? AND kind='initial'`, id)
					} else if previous.InitialMilli > 0 {
						_, err = tx.ExecContext(ctx, `UPDATE inventory_movements SET quantity_milli=? WHERE product_id=? AND kind='initial'`, in.InitialMilli, id)
					} else {
						_, err = tx.ExecContext(ctx, `INSERT INTO inventory_movements(product_id,kind,date,quantity_milli,notes) VALUES (?,'initial',?,?,?)`, id, time.Now().In(time.FixedZone("Sao Paulo", -3*60*60)).Format("2006-01-02"), in.InitialMilli, "Saldo inicial incluído por correção autorizada")
					}
					if err != nil {
						return err
					}
				}
				next := *previous
				next.Name = in.Name
				next.MinimumMilli = in.MinimumMilli
				next.InitialMilli = in.InitialMilli
				next.StockMilli = nextStock
				if err := inventoryAudit(ctx, tx, "edit", previous, &next); err != nil {
					return err
				}
			}
		}
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE constraint failed: inventory_products.name_key") {
				return InventoryError("Já existe um produto com esse nome.")
			}
			return err
		}
		if id == "" && in.InitialMilli > 0 {
			_, err = tx.ExecContext(ctx, `INSERT INTO inventory_movements(product_id,kind,date,quantity_milli,responsible) VALUES (?,'initial',?,?,?)`, in.RequestID, in.Date, in.InitialMilli, in.Responsible)
		}
		return err
	})
}
func (s *InventoryService) Purchase(ctx context.Context, in InventoryPurchaseInput) error {
	in.Supplier = strings.TrimSpace(in.Supplier)
	in.Document = strings.TrimSpace(in.Document)
	in.Notes = strings.TrimSpace(in.Notes)
	if !inventoryDate(in.Date) || in.Supplier == "" || !inventoryText(in.Supplier, 160) || !inventoryText(in.Document, 100) || !inventoryText(in.Notes, 1000) || len(in.Items) == 0 || len(in.Items) > 100 {
		return InventoryError("Informe data válida, fornecedor e pelo menos um item da compra.")
	}
	var total int64
	seen := map[string]bool{}
	for _, item := range in.Items {
		if item.ProductID == "" || seen[item.ProductID] || item.QuantityMilli <= 0 || item.QuantityMilli > 100000000 || item.UnitCostCents < 0 || item.UnitCostCents > 10000000 {
			return InventoryError("Confira os itens: não repita produtos e informe quantidades e valores válidos.")
		}
		seen[item.ProductID] = true
		total += (item.QuantityMilli*item.UnitCostCents + 500) / 1000
	}
	return s.operation(ctx, in.RequestID, "purchase", in, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO inventory_purchases(id,date,supplier,document,notes,total_cents) VALUES (?,?,?,?,?,?)`, in.RequestID, in.Date, in.Supplier, in.Document, in.Notes, total)
		if err != nil {
			return err
		}
		for _, item := range in.Items {
			result, err := tx.ExecContext(ctx, `UPDATE inventory_products SET stock_milli=stock_milli+? WHERE id=? AND deleted_at='' AND stock_milli<=9000000000000-?`, item.QuantityMilli, item.ProductID, item.QuantityMilli)
			if err != nil {
				return err
			}
			n, err := result.RowsAffected()
			if err != nil {
				return err
			}
			if n != 1 {
				return InventoryError("Produto inexistente ou saldo acima do limite.")
			}
			_, err = tx.ExecContext(ctx, `INSERT INTO inventory_movements(product_id,purchase_id,kind,date,quantity_milli,unit_cost_cents,total_cents) VALUES (?,?,'purchase',?,?,?,?)`, item.ProductID, in.RequestID, in.Date, item.QuantityMilli, item.UnitCostCents, (item.QuantityMilli*item.UnitCostCents+500)/1000)
			if err != nil {
				return err
			}
		}
		return nil
	})
}
func (s *InventoryService) Withdraw(ctx context.Context, in InventoryWithdrawalInput) error {
	in.Responsible = strings.TrimSpace(in.Responsible)
	in.Notes = strings.TrimSpace(in.Notes)
	if !inventoryDate(in.Date) || in.ProductID == "" || in.QuantityMilli <= 0 || in.QuantityMilli > 100000000 || in.Responsible == "" || !inventoryText(in.Responsible, 120) || !inventoryText(in.Notes, 1000) {
		return InventoryError("Informe produto, quantidade positiva, data válida e responsável pela retirada.")
	}
	return s.operation(ctx, in.RequestID, "withdrawal", in, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `UPDATE inventory_products SET stock_milli=stock_milli-? WHERE id=? AND deleted_at='' AND stock_milli>=?`, in.QuantityMilli, in.ProductID, in.QuantityMilli)
		if err != nil {
			return err
		}
		n, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if n != 1 {
			return ErrInventoryStock
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO inventory_movements(product_id,kind,date,quantity_milli,responsible,notes) VALUES (?,'withdrawal',?,?,?,?)`, in.ProductID, in.Date, -in.QuantityMilli, in.Responsible, in.Notes)
		return err
	})
}
func (s *InventoryService) Snapshot(ctx context.Context) (*InventorySnapshot, error) {
	// Read all three collections in one SQLite snapshot so totals and movements agree.
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	out := &InventorySnapshot{Products: []InventoryProduct{}, Purchases: []InventoryPurchase{}, Movements: []InventoryMovement{}}
	rows, err := tx.QueryContext(ctx, `SELECT id,name,unit,minimum_milli,stock_milli,COALESCE((SELECT SUM(quantity_milli) FROM inventory_movements WHERE product_id=inventory_products.id AND kind='initial'),0),deleted_at<>'' FROM inventory_products ORDER BY name_key`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var p InventoryProduct
		if err := rows.Scan(&p.ID, &p.Name, &p.Unit, &p.MinimumMilli, &p.StockMilli, &p.InitialMilli, &p.Deleted); err != nil {
			rows.Close()
			return nil, err
		}
		out.Products = append(out.Products, p)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	rows, err = tx.QueryContext(ctx, `SELECT id,date,supplier,document,notes,total_cents FROM inventory_purchases ORDER BY date DESC,created_at DESC,id`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var p InventoryPurchase
		if err := rows.Scan(&p.ID, &p.Date, &p.Supplier, &p.Document, &p.Notes, &p.TotalCents); err != nil {
			rows.Close()
			return nil, err
		}
		out.Purchases = append(out.Purchases, p)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	rows, err = tx.QueryContext(ctx, `SELECT id,product_id,COALESCE(purchase_id,''),kind,date,quantity_milli,unit_cost_cents,total_cents,responsible,notes FROM inventory_movements ORDER BY date DESC,id DESC`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var m InventoryMovement
		if err := rows.Scan(&m.ID, &m.ProductID, &m.PurchaseID, &m.Kind, &m.Date, &m.QuantityMilli, &m.UnitCostCents, &m.TotalCents, &m.Responsible, &m.Notes); err != nil {
			rows.Close()
			return nil, err
		}
		out.Movements = append(out.Movements, m)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	return out, tx.Commit()
}

// Keep typed validation errors distinct from database failures at the HTTP boundary.
func IsInventoryValidation(err error) bool {
	var validation InventoryError
	return errors.As(err, &validation)
}
