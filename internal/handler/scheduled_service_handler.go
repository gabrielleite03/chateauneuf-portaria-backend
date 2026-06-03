package handler

import (
	"encoding/json"
	"net/http"

	"chateauneuf-portaria-backend/internal/usecase"
)

type ScheduledServiceHandler struct {
	service *usecase.ScheduledServiceService
}

func NewScheduledServiceHandler(service *usecase.ScheduledServiceService) *ScheduledServiceHandler {
	return &ScheduledServiceHandler{service: service}
}

func (h *ScheduledServiceHandler) List(w http.ResponseWriter, r *http.Request) {
	services, err := h.service.List(r.Context())
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, services)
}

func (h *ScheduledServiceHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input usecase.CreateScheduledServiceInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "JSON invalido", "VALIDATION_ERROR")
		return
	}

	service, err := h.service.Create(r.Context(), input)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, service)
}

func (h *ScheduledServiceHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	var input usecase.UpdateScheduledServiceStatusInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "JSON invalido", "VALIDATION_ERROR")
		return
	}

	service, err := h.service.UpdateStatus(r.Context(), input)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, service)
}

func (h *ScheduledServiceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	var input usecase.DeleteScheduledServiceInput
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
