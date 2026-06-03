package handler

import (
	"encoding/json"
	"net/http"

	"chateauneuf-portaria-backend/internal/usecase"
)

type ResidentHandler struct {
	service *usecase.ResidentService
}

func NewResidentHandler(service *usecase.ResidentService) *ResidentHandler {
	return &ResidentHandler{service: service}
}

func (h *ResidentHandler) List(w http.ResponseWriter, r *http.Request) {
	residents, err := h.service.List(r.Context())
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, residents)
}

func (h *ResidentHandler) Upsert(w http.ResponseWriter, r *http.Request) {
	var input usecase.UpsertResidentInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "JSON invalido", "VALIDATION_ERROR")
		return
	}

	resident, err := h.service.Upsert(r.Context(), input)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resident)
}
