package repository

import (
	"chateauneuf-portaria-backend/internal/database"
	"chateauneuf-portaria-backend/internal/domain"
	"chateauneuf-portaria-backend/internal/usecase"
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func TestShoppingDeleteAuditAndWithdrawal(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = database.Migrate(db, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	repo := NewSQLiteShoppingRepository(db)
	service := usecase.NewShoppingService(repo)
	delivery, err := service.Create(ctx, usecase.CreateShoppingInput{Unit: "Apto 13", Recipient: "Morador", Product: "Caixa"})
	if err != nil {
		t.Fatal(err)
	}
	for _, password := range []string{"", "wrong"} {
		if err = service.Delete(ctx, delivery.ID, password); !errors.Is(err, usecase.ErrInventoryUnauthorized) {
			t.Fatalf("unauthorized: %v", err)
		}
	}
	rows, err := repo.List(ctx)
	if err != nil || len(rows) != 1 {
		t.Fatalf("record lost before deletion: %v", err)
	}
	for i := 0; i < 2; i++ {
		if err = repo.Delete(ctx, delivery.ID); err != nil {
			t.Fatal(err)
		}
	}
	rows, err = repo.List(ctx)
	if err != nil || len(rows) != 0 {
		t.Fatalf("deleted record visible: %v", err)
	}
	var count int
	if err = db.QueryRow(`SELECT COUNT(*) FROM shopping_deliveries d JOIN shopping_delivery_deletions x ON x.delivery_id=d.id WHERE x.deleted_at IS NOT NULL AND d.product='Caixa'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("audit missing: %v", err)
	}
	pending, err := repo.ListPendingSync(ctx, 50)
	if err != nil || len(pending) != 0 {
		t.Fatalf("deleted record pending sync: %v", err)
	}
	if count, err = repo.SyncStats(ctx); err != nil || count != 0 {
		t.Fatalf("sync count: %d %v", count, err)
	}
	if _, err = repo.Withdraw(ctx, delivery.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("deleted record withdrawn: %v", err)
	}
	signatures := usecase.NewDeliveryWithdrawalSignatureService(db, nil, nil)
	if _, err = signatures.CreateCode(ctx, delivery.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("deleted record code: %v", err)
	}
	other, err := service.Create(ctx, usecase.CreateShoppingInput{Unit: "Apto 13", Recipient: "Morador", Product: "Outra"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.Withdraw(ctx, other.ID); err != nil {
		t.Fatal(err)
	}
	if err = repo.Delete(ctx, other.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("withdrawn record deleted: %v", err)
	}
}
