package handler

import (
	"encoding/json"
	"net/http"

	"chateauneuf-portaria-backend/internal/photos"
	"chateauneuf-portaria-backend/internal/usecase"
)

type DiaristaHandler struct {
	service    *usecase.DiaristaService
	photoStore *photos.Store
}

func NewDiaristaHandler(service *usecase.DiaristaService, photoStore *photos.Store) *DiaristaHandler {
	return &DiaristaHandler{service: service, photoStore: photoStore}
}

func (h *DiaristaHandler) List(w http.ResponseWriter, r *http.Request) {
	entries, err := h.service.List(r.Context())
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

func (h *DiaristaHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input usecase.CreateDiaristaInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "JSON invalido", "VALIDATION_ERROR")
		return
	}
	if h.photoStore != nil {
		photo, err := h.photoStore.SaveDataURL(r.Context(), "diaristas", input.Photo)
		if err != nil {
			writeError(w, http.StatusBadRequest, "foto invalida", "VALIDATION_ERROR")
			return
		}
		input.Photo = photo
	}

	entry, err := h.service.Create(r.Context(), input)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, entry)
}

func (h *DiaristaHandler) Checkout(w http.ResponseWriter, r *http.Request) {
	var input usecase.CheckoutDiaristaInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "JSON invalido", "VALIDATION_ERROR")
		return
	}

	entry, err := h.service.Checkout(r.Context(), input)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, entry)
}
