package handler

import (
	"encoding/json"
	"net/http"

	"chateauneuf-portaria-backend/internal/usecase"
)

type ShoppingHandler struct {
	service *usecase.ShoppingService
}

func NewShoppingHandler(service *usecase.ShoppingService) *ShoppingHandler {
	return &ShoppingHandler{service: service}
}

func (h *ShoppingHandler) List(w http.ResponseWriter, r *http.Request) {
	deliveries, err := h.service.List(r.Context())
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, deliveries)
}

func (h *ShoppingHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input usecase.CreateShoppingInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "JSON invalido", "VALIDATION_ERROR")
		return
	}

	delivery, err := h.service.Create(r.Context(), input)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, delivery)
}

func (h *ShoppingHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	var input usecase.WithdrawShoppingInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "JSON invalido", "VALIDATION_ERROR")
		return
	}

	delivery, err := h.service.Withdraw(r.Context(), input)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, delivery)
}
