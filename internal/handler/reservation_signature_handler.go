package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"chateauneuf-portaria-backend/internal/domain"
	"chateauneuf-portaria-backend/internal/usecase"
)

type ReservationSignatureHandler struct {
	service *usecase.ReservationSignatureService
}

func NewReservationSignatureHandler(service *usecase.ReservationSignatureService) *ReservationSignatureHandler {
	return &ReservationSignatureHandler{service: service}
}
func (h *ReservationSignatureHandler) CreateCode(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.CreateCode(r.Context(), r.PathValue("id"))
	if err != nil {
		if errors.Is(err, usecase.ErrInternetCredentialEmailRequired) {
			writeError(w, http.StatusUnprocessableEntity, err.Error(), "RESIDENT_EMAIL_REQUIRED")
			return
		}
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}
func (h *ReservationSignatureHandler) Lookup(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Code string `json:"code"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		writeError(w, 400, "Código inválido", "VALIDATION_ERROR")
		return
	}
	result, err := h.service.Lookup(r.Context(), strings.TrimSpace(in.Code))
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, 200, result)
}
func (h *ReservationSignatureHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Code      string `json:"code"`
		Signature string `json:"signature"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		writeError(w, 400, "Dados inválidos", "VALIDATION_ERROR")
		return
	}
	if err := h.service.Confirm(r.Context(), strings.TrimSpace(in.Code), in.Signature); err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, 200, map[string]string{"status": "signed"})
}
func (h *ReservationSignatureHandler) Document(w http.ResponseWriter, r *http.Request) {
	pdf, err := h.service.Document(r.Context(), r.PathValue("id"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "inline; filename=termo-reserva.pdf")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdf)
}
func (h *ReservationSignatureHandler) writeError(w http.ResponseWriter, err error) {
	if errors.Is(err, usecase.ErrSignatureCodeInvalid) {
		writeError(w, http.StatusGone, "Código inválido, expirado ou já utilizado", "SIGNATURE_CODE_INVALID")
		return
	}
	if errors.Is(err, domain.ErrInvalidInput) {
		writeDomainError(w, err)
		return
	}
	writeError(w, 500, "Não foi possível concluir a assinatura", "INTERNAL_ERROR")
}
