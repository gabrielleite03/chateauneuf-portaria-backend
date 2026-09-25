package usecase

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"chateauneuf-portaria-backend/internal/database"
)

func inventoryTestService(t *testing.T) (*InventoryService, *sql.DB) {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "inventory.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	db.SetMaxOpenConns(1)
	if err := database.Migrate(db, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	service := NewInventoryService(db)
	service.passwordVerifier = "bab5121981ecfb09398732ffdf7a488d:8f90271e50d9a4f6f6cfbafd02625c9c0db0e5317c6c68d9c9001647bae35e02"
	return service, db
}
func inventoryTestProduct(t *testing.T, s *InventoryService, id string, initial int64) {
	t.Helper()
	if err := s.SaveProduct(context.Background(), "", InventoryProductInput{RequestID: id, Name: id, Unit: "L", InitialMilli: initial, MinimumMilli: 2000, Date: "2026-01-01", Responsible: "Maria"}); err != nil {
		t.Fatal(err)
	}
}
func inventoryTestSnapshot(t *testing.T, s *InventoryService) *InventorySnapshot {
	t.Helper()
	out, err := s.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestInventoryPurchaseWithdrawalAndRestart(t *testing.T) {
	s, db := inventoryTestService(t)
	ctx := context.Background()
	inventoryTestProduct(t, s, "product-0001", 2000)
	inventoryTestProduct(t, s, "product-0002", 0)
	purchase := InventoryPurchaseInput{RequestID: "purchase-0001", Date: "2026-01-02", Supplier: "Fornecedor A", Document: "NF-123", Notes: "Entrega completa", Items: []InventoryPurchaseItem{{ProductID: "product-0001", QuantityMilli: 1500, UnitCostCents: 1999}, {ProductID: "product-0002", QuantityMilli: 2000, UnitCostCents: 100}}}
	if err := s.Purchase(ctx, purchase); err != nil {
		t.Fatal(err)
	}
	if err := s.Purchase(ctx, purchase); err != nil {
		t.Fatalf("retry: %v", err)
	}
	out := inventoryTestSnapshot(t, s)
	if len(out.Purchases) != 1 || out.Purchases[0].TotalCents != 3199 || out.Purchases[0].Supplier != "Fornecedor A" || out.Purchases[0].Document != "NF-123" {
		t.Fatalf("purchase: %+v", out.Purchases)
	}
	if out.Products[0].StockMilli != 3500 || out.Products[1].StockMilli != 2000 || len(out.Movements) != 3 {
		t.Fatalf("unexpected stock: %+v", out)
	}
	withdrawal := InventoryWithdrawalInput{RequestID: "withdraw-0001", ProductID: "product-0001", Date: "2026-01-03", QuantityMilli: 500, Responsible: "João", Notes: "Garagem"}
	if err := s.Withdraw(ctx, withdrawal); err != nil {
		t.Fatal(err)
	}
	if err := s.Withdraw(ctx, withdrawal); err != nil {
		t.Fatal(err)
	}
	// Editing a label/minimum must not change the stock or historical price.
	if err := s.SaveProduct(ctx, "product-0001", InventoryProductInput{RequestID: "edit-0000001", Name: "Detergente", Unit: "L", MinimumMilli: 4000, InitialMilli: 2000, Password: "inventory-test-secret"}); err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(db, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatalf("restart migrations: %v", err)
	}
	out = inventoryTestSnapshot(t, NewInventoryService(db))
	if out.Products[0].StockMilli != 3000 || out.Products[0].MinimumMilli != 4000 || len(out.Movements) != 4 {
		t.Fatalf("restart changed data: %+v", out)
	}
	if out.Movements[0].Responsible != "João" || out.Movements[0].Notes != "Garagem" || out.Movements[0].QuantityMilli != -500 {
		t.Fatalf("withdrawal history: %+v", out.Movements[0])
	}
	purchase.Supplier = "Other supplier"
	if err := s.Purchase(ctx, purchase); !errors.Is(err, ErrInventoryRequest) {
		t.Fatalf("changed retry accepted: %v", err)
	}
}

func TestInventoryPurchaseIsAtomic(t *testing.T) {
	s, _ := inventoryTestService(t)
	ctx := context.Background()
	inventoryTestProduct(t, s, "product-0001", 1000)
	purchase := InventoryPurchaseInput{RequestID: "purchase-0001", Date: "2026-01-02", Supplier: "Supplier", Items: []InventoryPurchaseItem{{ProductID: "product-0001", QuantityMilli: 1000, UnitCostCents: 100}, {ProductID: "missing-product", QuantityMilli: 1000, UnitCostCents: 100}}}
	if err := s.Purchase(ctx, purchase); !IsInventoryValidation(err) {
		t.Fatalf("expected invalid product: %v", err)
	}
	out := inventoryTestSnapshot(t, s)
	if out.Products[0].StockMilli != 1000 || len(out.Purchases) != 0 || len(out.Movements) != 1 {
		t.Fatalf("partial purchase persisted: %+v", out)
	}
	// Failed operations leave no reservation of the request ID, so the corrected form can be saved.
	purchase.Items = purchase.Items[:1]
	if err := s.Purchase(ctx, purchase); err != nil {
		t.Fatal(err)
	}
}

func TestInventoryValidation(t *testing.T) {
	s, _ := inventoryTestService(t)
	ctx := context.Background()
	inventoryTestProduct(t, s, "product-0001", 1000)
	for _, quantity := range []int64{-1000, 0, 1001, 100000001} {
		in := InventoryWithdrawalInput{RequestID: fmt.Sprintf("withdraw-%d", quantity), ProductID: "product-0001", Date: "2026-01-02", Responsible: "Maria", QuantityMilli: quantity}
		if err := s.Withdraw(ctx, in); !IsInventoryValidation(err) {
			t.Fatalf("quantity %d: %v", quantity, err)
		}
	}
	for _, in := range []InventoryWithdrawalInput{
		{RequestID: "withdraw-0001", ProductID: "product-0001", Date: "2026-01-02", QuantityMilli: 1000},
		{RequestID: "withdraw-0002", ProductID: "product-0001", Date: "2099-01-01", QuantityMilli: 1000, Responsible: "Maria"},
		{RequestID: "withdraw-0003", ProductID: "product-0001", Date: "2026-02-30", QuantityMilli: 1000, Responsible: "Maria"},
	} {
		if err := s.Withdraw(ctx, in); !IsInventoryValidation(err) {
			t.Fatalf("invalid withdrawal accepted: %v", err)
		}
	}
	duplicate := InventoryProductInput{RequestID: "product-0002", Name: "  PRODUCT-0001  ", Unit: "L"}
	if err := s.SaveProduct(ctx, "", duplicate); !IsInventoryValidation(err) {
		t.Fatalf("duplicate name accepted: %v", err)
	}
	duplicate.Name = "New label"
	duplicate.Password = "inventory-test-secret"
	duplicate.Unit = "kg"
	if err := s.SaveProduct(ctx, "product-0001", duplicate); !IsInventoryValidation(err) {
		t.Fatalf("unit change accepted: %v", err)
	}
	items := []InventoryPurchaseItem{{ProductID: "product-0001", QuantityMilli: 1000, UnitCostCents: 100}, {ProductID: "product-0001", QuantityMilli: 1000, UnitCostCents: 200}}
	if err := s.Purchase(ctx, InventoryPurchaseInput{RequestID: "purchase-0001", Date: "2026-01-02", Supplier: "Supplier", Items: items}); !IsInventoryValidation(err) {
		t.Fatalf("duplicate item accepted: %v", err)
	}
	if got := inventoryTestSnapshot(t, s).Products[0].StockMilli; got != 1000 {
		t.Fatalf("invalid operation changed stock: %d", got)
	}
}

func TestInventoryConcurrentWithdrawalsCannotOverdraw(t *testing.T) {
	s, _ := inventoryTestService(t)
	inventoryTestProduct(t, s, "product-0001", 1000)
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results <- s.Withdraw(context.Background(), InventoryWithdrawalInput{RequestID: fmt.Sprintf("withdraw-%04d", i), ProductID: "product-0001", Date: "2026-01-02", QuantityMilli: 1000, Responsible: "Maria"})
		}(i)
	}
	wg.Wait()
	close(results)
	saved, rejected := 0, 0
	for err := range results {
		if err == nil {
			saved++
		} else if errors.Is(err, ErrInventoryStock) {
			rejected++
		} else {
			t.Fatal(err)
		}
	}
	if saved != 1 || rejected != 1 || inventoryTestSnapshot(t, s).Products[0].StockMilli != 0 {
		t.Fatal("concurrent withdrawals did not preserve stock")
	}
}
