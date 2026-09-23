package usecase

import (
	"context"
	"database/sql"
	"encoding/base64"
	"strings"
	"testing"

	"chateauneuf-portaria-backend/internal/domain"
	_ "modernc.org/sqlite"
)

func TestNormalizeApartmentUnit(t *testing.T) {
	tests := map[string]string{
		"13":             "13",
		"Apto 13":        "13",
		"apto. 013":      "13",
		"Apartamento 13": "13",
	}
	for input, expected := range tests {
		if actual := normalizeApartmentUnit(input); actual != expected {
			t.Errorf("normalizeApartmentUnit(%q) = %q; want %q", input, actual, expected)
		}
	}
}

type withdrawalResidentRepository struct{}

func (withdrawalResidentRepository) List(context.Context) ([]domain.Resident, error) {
	return []domain.Resident{{Unit: "13", Email: "morador@example.com"}}, nil
}
func (withdrawalResidentRepository) Upsert(context.Context, domain.Resident) (*domain.Resident, error) {
	return nil, nil
}

type withdrawalMailer struct{ sent bool }

func (*withdrawalMailer) Send(context.Context, string, string, string) error { return nil }
func (*withdrawalMailer) SendAttachment(context.Context, string, string, string, string, []byte) error {
	return nil
}
func (m *withdrawalMailer) SendInlineImage(context.Context, string, string, string, string, []byte) error {
	m.sent = true
	return nil
}

func TestDeliveryWithdrawalSignatureFlow(t *testing.T) {
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`
		CREATE TABLE shopping_deliveries(id INTEGER PRIMARY KEY,unit TEXT,recipient TEXT,store TEXT,product TEXT,received_at DATETIME,withdrawn_at DATETIME,status TEXT,sync_status TEXT,sync_error TEXT,updated_at DATETIME);
		CREATE TABLE delivery_withdrawal_signatures(id INTEGER PRIMARY KEY,delivery_id INTEGER,code_hash TEXT UNIQUE,expires_at DATETIME,consumed_at DATETIME,signature_png BLOB,recipient_email TEXT,email_status TEXT DEFAULT 'pending',email_error TEXT DEFAULT '',created_at DATETIME);
		INSERT INTO shopping_deliveries(id,unit,recipient,store,product,received_at,status,sync_status,sync_error,updated_at) VALUES(1,'Apto 13','Morador','Loja','Caixa',CURRENT_TIMESTAMP,'aguardando_retirada','SINCRONIZADO','',CURRENT_TIMESTAMP);
	`)
	if err != nil {
		t.Fatal(err)
	}
	mailer := &withdrawalMailer{}
	service := NewDeliveryWithdrawalSignatureService(db, NewResidentService(withdrawalResidentRepository{}), mailer)
	code, err := service.CreateCode(context.Background(), "1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.Lookup(context.Background(), code.Code); err != nil {
		t.Fatal(err)
	}
	signature := "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte(strings.Repeat("x", 150)))
	if err = service.Confirm(context.Background(), code.Code, signature); err != nil {
		t.Fatal(err)
	}
	status, err := service.Status(context.Background(), "1")
	if err != nil || status.Status != "signed" || !mailer.sent {
		t.Fatalf("status=%+v sent=%v err=%v", status, mailer.sent, err)
	}
	if err = service.Confirm(context.Background(), code.Code, signature); err != ErrSignatureCodeInvalid {
		t.Fatalf("expected one-use code error, got %v", err)
	}
}
