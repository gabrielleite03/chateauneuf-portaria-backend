package handler

import (
	"bytes"
	"chateauneuf-portaria-backend/internal/database"
	"chateauneuf-portaria-backend/internal/usecase"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestInventoryHTTPFlow(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "inventory.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(db, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	router := NewRouter(RouterDeps{InventoryService: usecase.NewInventoryService(db)})
	request := func(method, path, body string, status int) *httptest.ResponseRecorder {
		t.Helper()
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(method, path, bytes.NewBufferString(body)))
		if w.Code != status {
			t.Fatalf("%s %s: %d %s", method, path, w.Code, w.Body.String())
		}
		return w
	}
	request(http.MethodPost, "/api/inventory/products", `{"requestId":"product-0001","name":"Sabão","unit":"kg","minimumMilli":1000,"initialMilli":0}`, 200)
	purchase := `{"requestId":"purchase-0001","date":"2026-01-01","supplier":"Loja de limpeza","document":"123","notes":"Compra mensal","items":[{"productId":"product-0001","quantityMilli":2000,"unitCostCents":1250}]}`
	request(http.MethodPost, "/api/inventory/purchases", purchase, 200)
	request(http.MethodPost, "/api/inventory/purchases", purchase, 200)
	request(http.MethodPost, "/api/inventory/withdrawals", `{"requestId":"withdraw-0001","productId":"product-0001","quantityMilli":500,"date":"2026-01-02","responsible":"Maria","notes":"Hall"}`, 200)
	request(http.MethodPost, "/api/inventory/withdrawals", `{"requestId":"withdraw-0002","productId":"product-0001","quantityMilli":2000,"date":"2026-01-02","responsible":"Maria"}`, 400)
	request(http.MethodPost, "/api/inventory/products", `{"unexpected":true}`, 400)
	request(http.MethodPost, "/api/inventory/products/product-0001", `{"requestId":"edit-0000001","name":"Sabão","unit":"kg","initialMilli":5000}`, 403)
	request(http.MethodPost, "/api/inventory/products/product-0001", `{"requestId":"edit-0000001","name":"Sabão","unit":"kg","initialMilli":5000,"password":"wrong"}`, 403)
	request(http.MethodPost, "/api/inventory/products/product-0001/delete", `{"requestId":"delete-000001"}`, 403)
	request(http.MethodPost, "/api/inventory/products/product-0001/delete", `{"requestId":"delete-000001","password":"wrong"}`, 403)
	request(http.MethodPost, "/api/inventory/products", `{} {}`, 400)
	response := request(http.MethodGet, "/api/inventory", "", 200)
	var data usecase.InventorySnapshot
	if err := json.Unmarshal(response.Body.Bytes(), &data); err != nil {
		t.Fatal(err)
	}
	if len(data.Products) != 1 || data.Products[0].StockMilli != 1500 || len(data.Purchases) != 1 || data.Purchases[0].TotalCents != 2500 || len(data.Movements) != 2 {
		t.Fatalf("unexpected persisted data: %+v", data)
	}
}
