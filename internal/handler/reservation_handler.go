package handler

import (
	"encoding/json"
	"net/http"

	"chateauneuf-portaria-backend/internal/usecase"
)

type ReservationHandler struct {
	service *usecase.ReservationService
}

func NewReservationHandler(service *usecase.ReservationService) *ReservationHandler {
	return &ReservationHandler{service: service}
}

func (h *ReservationHandler) List(w http.ResponseWriter, r *http.Request) {
	reservations, err := h.service.List(r.Context())
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, reservations)
}

func (h *ReservationHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input usecase.CreateReservationInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "JSON invalido", "VALIDATION_ERROR")
		return
	}
	reservation, err := h.service.Create(r.Context(), input)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, reservation)
}

func (h *ReservationHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	var input usecase.UpdateReservationStatusInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "JSON invalido", "VALIDATION_ERROR")
		return
	}
	reservation, err := h.service.UpdateStatus(r.Context(), input)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, reservation)
}

func (h *ReservationHandler) Delete(w http.ResponseWriter, r *http.Request) {
	var input usecase.DeleteReservationInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "JSON invalido", "VALIDATION_ERROR")
		return
	}
	if err := h.service.Delete(r.Context(), input); err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
