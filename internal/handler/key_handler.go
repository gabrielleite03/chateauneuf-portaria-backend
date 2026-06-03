package handler

import (
	"encoding/json"
	"net/http"

	"chateauneuf-portaria-backend/internal/usecase"
)

type KeyHandler struct {
	service *usecase.KeyService
}

func NewKeyHandler(service *usecase.KeyService) *KeyHandler {
	return &KeyHandler{service: service}
}

func (h *KeyHandler) List(w http.ResponseWriter, r *http.Request) {
	keys, err := h.service.List(r.Context())
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, keys)
}

func (h *KeyHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input usecase.CreateKeyInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "JSON invalido", "VALIDATION_ERROR")
		return
	}

	key, err := h.service.Create(r.Context(), input)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, key)
}

func (h *KeyHandler) Return(w http.ResponseWriter, r *http.Request) {
	var input usecase.ReturnKeyInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "JSON invalido", "VALIDATION_ERROR")
		return
	}

	key, err := h.service.Return(r.Context(), input)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, key)
}

func (h *KeyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	var input usecase.DeleteKeyInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "JSON invalido", "VALIDATION_ERROR")
		return
	}

	if err := h.service.Delete(r.Context(), input); err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}
