package handler

import (
	"net/http"

	"chateauneuf-portaria-backend/internal/usecase"
)

type SyncHandler struct {
	service usecase.SyncService
}

func NewSyncHandler(service usecase.SyncService) *SyncHandler {
	return &SyncHandler{service: service}
}

func (h *SyncHandler) Status(w http.ResponseWriter, r *http.Request) {
	status, err := h.service.Status(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao consultar sincronizacao", "INTERNAL_ERROR")
		return
	}

	writeJSON(w, http.StatusOK, status)
}

func (h *SyncHandler) Run(w http.ResponseWriter, r *http.Request) {
	if err := h.service.RunOnce(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao executar sincronizacao", "INTERNAL_ERROR")
		return
	}

	status, err := h.service.Status(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao consultar sincronizacao", "INTERNAL_ERROR")
		return
	}

	writeJSON(w, http.StatusOK, status)
}
