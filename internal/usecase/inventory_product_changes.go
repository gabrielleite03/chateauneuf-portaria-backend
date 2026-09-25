package usecase

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
)

func inventoryProductInTx(ctx context.Context, tx *sql.Tx, id string) (*InventoryProduct, error) {
	var p InventoryProduct
	err := tx.QueryRowContext(ctx, `SELECT id,name,unit,minimum_milli,stock_milli,COALESCE((SELECT SUM(quantity_milli) FROM inventory_movements WHERE product_id=inventory_products.id AND kind='initial'),0),deleted_at<>'' FROM inventory_products WHERE id=?`, id).Scan(&p.ID, &p.Name, &p.Unit, &p.MinimumMilli, &p.StockMilli, &p.InitialMilli, &p.Deleted)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, InventoryError("Produto não encontrado.")
	}
	return &p, err
}

func inventoryAudit(ctx context.Context, tx *sql.Tx, action string, before, after *InventoryProduct) error {
	previous, err := json.Marshal(before)
	if err != nil {
		return err
	}
	next, err := json.Marshal(after)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO inventory_product_changes(product_id,action,before_json,after_json) VALUES (?,?,?,?)`, before.ID, action, string(previous), string(next))
	return err
}

func (s *InventoryService) DeleteProduct(ctx context.Context, id string, in InventoryDeleteInput) error {
	if err := s.authorize(in.Password); err != nil {
		return err
	}
	in.Password = ""
	return s.operation(ctx, in.RequestID, "delete-product:"+id, in, func(tx *sql.Tx) error {
		previous, err := inventoryProductInTx(ctx, tx, id)
		if err != nil {
			return err
		}
		if previous.Deleted {
			return InventoryError("Este produto já foi excluído.")
		}
		// Archive so purchases, supplier costs and stock history remain available.
		_, err = tx.ExecContext(ctx, `UPDATE inventory_products SET deleted_at=CURRENT_TIMESTAMP,name_key=? WHERE id=?`, "archived:"+id, id)
		if err != nil {
			return err
		}
		next := *previous
		next.Deleted = true
		return inventoryAudit(ctx, tx, "delete", previous, &next)
	})
}
