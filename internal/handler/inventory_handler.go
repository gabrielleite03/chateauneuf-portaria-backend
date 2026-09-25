package handler

import (
	"chateauneuf-portaria-backend/internal/usecase"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

type InventoryHandler struct{ service *usecase.InventoryService }

func inventoryDecode(w http.ResponseWriter, r *http.Request, input any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 128*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(input); err != nil {
		writeError(w, 400, "Dados inválidos. Confira os campos enviados.", "INVALID_INVENTORY")
		return false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeError(w, 400, "Envie apenas uma operação.", "INVALID_INVENTORY")
		return false
	}
	return true
}
func inventoryResult(w http.ResponseWriter, err error) {
	if err != nil {
		if errors.Is(err, usecase.ErrInventoryUnauthorized) {
			writeError(w, http.StatusForbidden, err.Error(), "INVENTORY_UNAUTHORIZED")
			return
		}
		if usecase.IsInventoryValidation(err) {
			writeError(w, 400, err.Error(), "INVALID_INVENTORY")
		} else {
			writeError(w, 500, "Não foi possível concluir a operação de estoque. Tente novamente.", "INVENTORY_ERROR")
		}
		return
	}
	writeJSON(w, 200, map[string]bool{"saved": true})
}
func (h *InventoryHandler) Snapshot(w http.ResponseWriter, r *http.Request) {
	snapshot, err := h.service.Snapshot(r.Context())
	if err != nil {
		inventoryResult(w, err)
		return
	}
	writeJSON(w, 200, snapshot)
}
func (h *InventoryHandler) Product(w http.ResponseWriter, r *http.Request) {
	var input usecase.InventoryProductInput
	if !inventoryDecode(w, r, &input) {
		return
	}
	inventoryResult(w, h.service.SaveProduct(r.Context(), r.PathValue("productID"), input))
}
func (h *InventoryHandler) Purchase(w http.ResponseWriter, r *http.Request) {
	var input usecase.InventoryPurchaseInput
	if !inventoryDecode(w, r, &input) {
		return
	}
	inventoryResult(w, h.service.Purchase(r.Context(), input))
}
func (h *InventoryHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	var input usecase.InventoryWithdrawalInput
	if !inventoryDecode(w, r, &input) {
		return
	}
	inventoryResult(w, h.service.Withdraw(r.Context(), input))
}

func (h *InventoryHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	var input usecase.InventoryDeleteInput
	if !inventoryDecode(w, r, &input) {
		return
	}
	inventoryResult(w, h.service.DeleteProduct(r.Context(), r.PathValue("productID"), input))
}
