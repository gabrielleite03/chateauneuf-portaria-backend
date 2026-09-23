package handler

import (
	"encoding/json"
	"net/http"

	"chateauneuf-portaria-backend/internal/usecase"
)

type ReservationGuestHandler struct {
	service *usecase.ReservationGuestService
}

func (h *ReservationGuestHandler) List(w http.ResponseWriter, r *http.Request) {
	guests, err := h.service.List(r.Context(), r.PathValue("id"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, guests)
}

func (h *ReservationGuestHandler) Add(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name     string `json:"name"`
		Document string `json:"document"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "JSON invalido", "VALIDATION_ERROR")
		return
	}
	guest, err := h.service.Add(r.Context(), r.PathValue("id"), input.Name, input.Document)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, guest)
}

func (h *ReservationGuestHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Confirmed *bool `json:"confirmed"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.Confirmed == nil {
		writeError(w, http.StatusBadRequest, "Informe a confirmacao de entrada", "VALIDATION_ERROR")
		return
	}
	if err := h.service.Confirm(r.Context(), r.PathValue("id"), r.PathValue("guestID"), *input.Confirmed); err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"confirmed": *input.Confirmed})
}
