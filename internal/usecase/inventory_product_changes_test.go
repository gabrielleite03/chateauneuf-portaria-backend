package usecase

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"chateauneuf-portaria-backend/internal/database"
)

func TestInventoryProtectedCorrections(t *testing.T) {
	s, db := inventoryTestService(t)
	ctx := context.Background()
	inventoryTestProduct(t, s, "product-0001", 10000)
	purchase := InventoryPurchaseInput{RequestID: "purchase-0001", Date: "2026-01-02", Supplier: "Fornecedor", Items: []InventoryPurchaseItem{{ProductID: "product-0001", QuantityMilli: 5000, UnitCostCents: 299}}}
	if err := s.Purchase(ctx, purchase); err != nil {
		t.Fatal(err)
	}
	if err := s.Withdraw(ctx, InventoryWithdrawalInput{RequestID: "withdraw-0001", ProductID: "product-0001", QuantityMilli: 8000, Date: "2026-01-03", Responsible: "Maria"}); err != nil {
		t.Fatal(err)
	}
	edit := InventoryProductInput{RequestID: "edit-0000001", Name: "Detergente", Unit: "L", InitialMilli: 5000, MinimumMilli: 1000}
	for _, password := range []string{"", "wrong-password"} {
		edit.Password = password
		if err := s.SaveProduct(ctx, "product-0001", edit); !errors.Is(err, ErrInventoryUnauthorized) {
			t.Fatalf("unauthorized edit: %v", err)
		}
		if inventoryTestSnapshot(t, s).Products[0].StockMilli != 7000 {
			t.Fatal("unauthorized edit changed stock")
		}
	}
	edit.Password = "inventory-test-secret"
	if err := s.SaveProduct(ctx, "product-0001", edit); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveProduct(ctx, "product-0001", edit); err != nil {
		t.Fatalf("authorized retry: %v", err)
	}
	data := inventoryTestSnapshot(t, s)
	if data.Products[0].StockMilli != 2000 || data.Products[0].InitialMilli != 5000 || data.Purchases[0].TotalCents != 1495 || len(data.Movements) != 3 {
		t.Fatalf("bad corrected stock: %+v", data)
	}
	edit.RequestID = "edit-0000002"
	edit.InitialMilli = 2000
	if err := s.SaveProduct(ctx, "product-0001", edit); !IsInventoryValidation(err) {
		t.Fatalf("negative balance accepted: %v", err)
	}
	if inventoryTestSnapshot(t, s).Products[0].StockMilli != 2000 {
		t.Fatal("failed correction changed stock")
	}
	var auditCount int
	var before, after string
	if err := db.QueryRow(`SELECT COUNT(*),before_json,after_json FROM inventory_product_changes`).Scan(&auditCount, &before, &after); err != nil {
		t.Fatal(err)
	}
	if auditCount != 1 || !strings.Contains(before, `"initialMilli":10000`) || !strings.Contains(after, `"initialMilli":5000`) || strings.Contains(after, "password") || strings.Contains(before, edit.Password) {
		t.Fatalf("invalid audit: %d", auditCount)
	}
	if err := database.Migrate(db, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	if inventoryTestSnapshot(t, NewInventoryService(db)).Products[0].InitialMilli != 5000 {
		t.Fatal("correction lost on restart")
	}
}

func TestInventoryInitialZeroAndProtectedDeletion(t *testing.T) {
	s, _ := inventoryTestService(t)
	ctx := context.Background()
	inventoryTestProduct(t, s, "product-0001", 0)
	edit := InventoryProductInput{RequestID: "edit-0000001", Name: "product-0001", Unit: "L", InitialMilli: 1000, Password: "inventory-test-secret"}
	if err := s.SaveProduct(ctx, "product-0001", edit); err != nil {
		t.Fatal(err)
	}
	edit.RequestID = "edit-0000002"
	edit.InitialMilli = 0
	if err := s.SaveProduct(ctx, "product-0001", edit); err != nil {
		t.Fatal(err)
	}
	data := inventoryTestSnapshot(t, s)
	if data.Products[0].InitialMilli != 0 || data.Products[0].StockMilli != 0 || len(data.Movements) != 0 {
		t.Fatalf("zero correction: %+v", data)
	}
	purchase := InventoryPurchaseInput{RequestID: "purchase-0001", Date: "2026-01-02", Supplier: "Fornecedor", Items: []InventoryPurchaseItem{{ProductID: "product-0001", QuantityMilli: 5000, UnitCostCents: 299}}}
	if err := s.Purchase(ctx, purchase); err != nil {
		t.Fatal(err)
	}
	deletion := InventoryDeleteInput{RequestID: "delete-000001"}
	for _, password := range []string{"", "wrong-password"} {
		deletion.Password = password
		if err := s.DeleteProduct(ctx, "product-0001", deletion); !errors.Is(err, ErrInventoryUnauthorized) {
			t.Fatalf("unauthorized delete: %v", err)
		}
	}
	deletion.Password = "inventory-test-secret"
	if err := s.DeleteProduct(ctx, "product-0001", deletion); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteProduct(ctx, "product-0001", deletion); err != nil {
		t.Fatalf("retry delete: %v", err)
	}
	data = inventoryTestSnapshot(t, s)
	if !data.Products[0].Deleted || data.Products[0].StockMilli != 5000 || len(data.Purchases) != 1 || len(data.Movements) != 1 {
		t.Fatalf("deleted history lost: %+v", data)
	}
	purchase.RequestID = "purchase-0002"
	if err := s.Purchase(ctx, purchase); !IsInventoryValidation(err) {
		t.Fatalf("purchase for deleted product accepted: %v", err)
	}
	if err := s.Withdraw(ctx, InventoryWithdrawalInput{RequestID: "withdraw-0001", ProductID: "product-0001", QuantityMilli: 1000, Date: "2026-01-03", Responsible: "Maria"}); !IsInventoryValidation(err) {
		t.Fatalf("withdrawal for deleted product accepted: %v", err)
	}
	edit.RequestID = "edit-0000003"
	if err := s.SaveProduct(ctx, "product-0001", edit); !IsInventoryValidation(err) {
		t.Fatalf("editing deleted product accepted: %v", err)
	}
	if err := s.SaveProduct(ctx, "", InventoryProductInput{RequestID: "product-0002", Name: "product-0001", Unit: "L"}); err != nil {
		t.Fatalf("recreate name: %v", err)
	}
}
