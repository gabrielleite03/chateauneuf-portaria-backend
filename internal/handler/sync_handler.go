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

func (h *SyncHandler) ImportAccessLogs(w http.ResponseWriter, r *http.Request) {
	importedScheduledServicesCount, err := h.service.ImportScheduledServices(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao importar servicos agendados da planilha", "INTERNAL_ERROR")
		return
	}

	importedDiaristasCount, err := h.service.ImportDiaristas(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao importar diaristas da planilha", "INTERNAL_ERROR")
		return
	}

	importedResidentsCount, err := h.service.ImportResidents(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao importar moradores da planilha", "INTERNAL_ERROR")
		return
	}

	importedCount, err := h.service.ImportAccessLogs(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao importar historico da planilha", "INTERNAL_ERROR")
		return
	}

	status, err := h.service.Status(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao consultar sincronizacao", "INTERNAL_ERROR")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"imported_count":                    importedCount,
		"imported_residents_count":          importedResidentsCount,
		"imported_diaristas_count":          importedDiaristasCount,
		"imported_scheduled_services_count": importedScheduledServicesCount,
		"status":                            status,
	})
}
