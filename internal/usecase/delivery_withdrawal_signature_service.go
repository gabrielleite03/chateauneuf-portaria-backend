package usecase

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"html"
	"strconv"
	"strings"
	"time"

	"chateauneuf-portaria-backend/internal/domain"
)

type DeliveryWithdrawalSignatureService struct {
	db        *sql.DB
	residents *ResidentService
	mailer    PDFEmailSender
}

type DeliveryWithdrawalForm struct {
	DeliveryID string    `json:"deliveryId"`
	Unit       string    `json:"unit"`
	Recipient  string    `json:"recipient"`
	Store      string    `json:"store"`
	Product    string    `json:"product"`
	ReceivedAt time.Time `json:"receivedAt"`
	ExpiresAt  time.Time `json:"expiresAt"`
}

type DeliveryWithdrawalStatus struct {
	Status      string `json:"status"`
	EmailStatus string `json:"emailStatus,omitempty"`
}

func NewDeliveryWithdrawalSignatureService(db *sql.DB, residents *ResidentService, mailer PDFEmailSender) *DeliveryWithdrawalSignatureService {
	return &DeliveryWithdrawalSignatureService{db: db, residents: residents, mailer: mailer}
}

func (s *DeliveryWithdrawalSignatureService) CreateCode(ctx context.Context, deliveryID string) (SignatureCode, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(deliveryID), 10, 64)
	if err != nil {
		return SignatureCode{}, domain.ErrInvalidInput
	}
	var unit, status string
	if err = s.db.QueryRowContext(ctx, `SELECT unit,status FROM shopping_deliveries WHERE id=?`, id).Scan(&unit, &status); err != nil {
		return SignatureCode{}, domain.ErrNotFound
	}
	if status != string(domain.ShoppingStatusWaiting) {
		return SignatureCode{}, domain.ErrInvalidInput
	}
	email, err := (&ReservationSignatureService{residents: s.residents}).residentEmail(ctx, unit)
	if err != nil {
		return SignatureCode{}, err
	}
	var random [4]byte
	if _, err = rand.Read(random[:]); err != nil {
		return SignatureCode{}, err
	}
	code := fmt.Sprintf("%06d", (uint32(random[0])<<24|uint32(random[1])<<16|uint32(random[2])<<8|uint32(random[3]))%1000000)
	now, expires := time.Now().Round(0), time.Now().Add(SignatureCodeTTL).Round(0)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return SignatureCode{}, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `UPDATE delivery_withdrawal_signatures SET expires_at=? WHERE delivery_id=? AND consumed_at IS NULL`, now, id); err != nil {
		return SignatureCode{}, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO delivery_withdrawal_signatures(delivery_id,code_hash,expires_at,recipient_email,created_at) VALUES(?,?,?,?,?)`, id, codeHash(code), expires, email, now); err != nil {
		return SignatureCode{}, err
	}
	if err = tx.Commit(); err != nil {
		return SignatureCode{}, err
	}
	return SignatureCode{Code: code, ExpiresAt: expires}, nil
}

func (s *DeliveryWithdrawalSignatureService) Lookup(ctx context.Context, code string) (DeliveryWithdrawalForm, error) {
	var form DeliveryWithdrawalForm
	err := s.db.QueryRowContext(ctx, `
		SELECT CAST(d.id AS TEXT),d.unit,d.recipient,d.store,d.product,d.received_at,s.expires_at
		FROM delivery_withdrawal_signatures s
		JOIN shopping_deliveries d ON d.id=s.delivery_id
		WHERE s.code_hash=? AND s.consumed_at IS NULL AND s.expires_at>? AND d.status=?
	`, codeHash(strings.TrimSpace(code)), time.Now(), domain.ShoppingStatusWaiting).Scan(&form.DeliveryID, &form.Unit, &form.Recipient, &form.Store, &form.Product, &form.ReceivedAt, &form.ExpiresAt)
	if err != nil {
		return DeliveryWithdrawalForm{}, ErrSignatureCodeInvalid
	}
	return form, nil
}

func (s *DeliveryWithdrawalSignatureService) Confirm(ctx context.Context, code, dataURL string) error {
	comma := strings.IndexByte(dataURL, ',')
	if comma < 0 || !strings.HasPrefix(dataURL, "data:image/png") {
		return domain.ErrInvalidInput
	}
	signature, err := base64.StdEncoding.DecodeString(dataURL[comma+1:])
	if err != nil || len(signature) < 100 || len(signature) > 2_000_000 {
		return domain.ErrInvalidInput
	}
	now := time.Now().Round(0)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var deliveryID int64
	var unit, recipient, store, product, email string
	err = tx.QueryRowContext(ctx, `
		SELECT d.id,d.unit,d.recipient,d.store,d.product,s.recipient_email
		FROM delivery_withdrawal_signatures s JOIN shopping_deliveries d ON d.id=s.delivery_id
		WHERE s.code_hash=? AND s.consumed_at IS NULL AND s.expires_at>? AND d.status=?
	`, codeHash(strings.TrimSpace(code)), now, domain.ShoppingStatusWaiting).Scan(&deliveryID, &unit, &recipient, &store, &product, &email)
	if err != nil {
		return ErrSignatureCodeInvalid
	}
	result, err := tx.ExecContext(ctx, `UPDATE delivery_withdrawal_signatures SET consumed_at=?,signature_png=?,email_status='pending' WHERE code_hash=? AND consumed_at IS NULL AND expires_at>?`, now, signature, codeHash(strings.TrimSpace(code)), now)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected != 1 {
		return ErrSignatureCodeInvalid
	}
	result, err = tx.ExecContext(ctx, `UPDATE shopping_deliveries SET withdrawn_at=?,status=?,sync_status=?,sync_error='',updated_at=? WHERE id=? AND status=?`, now, domain.ShoppingStatusWithdrawn, domain.SyncStatusPending, now, deliveryID, domain.ShoppingStatusWaiting)
	if err != nil {
		return err
	}
	affected, _ = result.RowsAffected()
	if affected != 1 {
		return ErrSignatureCodeInvalid
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	body := fmt.Sprintf(`<!doctype html><html><body style="margin:0;background:#102d3a;font-family:Arial,sans-serif;color:#172b35"><table width="100%%" role="presentation"><tr><td align="center" style="padding:28px 12px"><table width="100%%" style="max-width:520px;background:#fff;border-radius:22px;overflow:hidden"><tr><td align="center" style="padding:28px;background:#17495b"><h1 style="font:24px Georgia;margin:0;color:#c9a45d">Condomínio Edifício Chateauneuf</h1><p style="letter-spacing:2px;font-size:12px;color:#fff">RETIRADA CONFIRMADA</p></td></tr><tr><td style="padding:30px"><p>Olá, <strong>%s</strong>.</p><p>A retirada da mercadoria do apartamento <strong>%s</strong> foi confirmada em %s.</p><div style="padding:18px;background:#f4f7f8;border:1px solid #dbe4e7;border-radius:12px"><strong>%s</strong><br>Origem: %s</div><p style="margin:20px 0 8px;font-weight:bold">Assinatura de retirada:</p><img src="cid:delivery-photo" alt="Assinatura" style="display:block;max-width:100%%;height:auto;border:1px solid #dbe4e7;border-radius:8px"></td></tr></table></td></tr></table></body></html>`, html.EscapeString(recipient), html.EscapeString(unit), now.In(time.Local).Format("02/01/2006 15:04"), html.EscapeString(product), html.EscapeString(store))
	err = s.mailer.SendInlineImage(ctx, email, "Retirada confirmada - Apartamento "+unit, body, "image/png", signature)
	emailStatus, emailError := "sent", ""
	if err != nil {
		emailStatus, emailError = "failed", err.Error()
	}
	_, _ = s.db.ExecContext(context.Background(), `UPDATE delivery_withdrawal_signatures SET email_status=?,email_error=? WHERE code_hash=?`, emailStatus, emailError, codeHash(strings.TrimSpace(code)))
	return nil
}

func (s *DeliveryWithdrawalSignatureService) Status(ctx context.Context, deliveryID string) (DeliveryWithdrawalStatus, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(deliveryID), 10, 64)
	if err != nil {
		return DeliveryWithdrawalStatus{}, domain.ErrInvalidInput
	}
	var expires time.Time
	var consumed sql.NullTime
	var emailStatus string
	err = s.db.QueryRowContext(ctx, `SELECT expires_at,consumed_at,email_status FROM delivery_withdrawal_signatures WHERE delivery_id=? ORDER BY id DESC LIMIT 1`, id).Scan(&expires, &consumed, &emailStatus)
	if err != nil {
		return DeliveryWithdrawalStatus{}, domain.ErrNotFound
	}
	if consumed.Valid {
		return DeliveryWithdrawalStatus{Status: "signed", EmailStatus: emailStatus}, nil
	}
	if !expires.After(time.Now()) {
		return DeliveryWithdrawalStatus{Status: "expired"}, nil
	}
	return DeliveryWithdrawalStatus{Status: "waiting"}, nil
}
