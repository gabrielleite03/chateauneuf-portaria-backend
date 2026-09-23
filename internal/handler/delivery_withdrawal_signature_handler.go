package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"chateauneuf-portaria-backend/internal/domain"
	"chateauneuf-portaria-backend/internal/usecase"
)

type DeliveryWithdrawalSignatureHandler struct {
	service *usecase.DeliveryWithdrawalSignatureService
}

func NewDeliveryWithdrawalSignatureHandler(service *usecase.DeliveryWithdrawalSignatureService) *DeliveryWithdrawalSignatureHandler {
	return &DeliveryWithdrawalSignatureHandler{service: service}
}

func (h *DeliveryWithdrawalSignatureHandler) CreateCode(w http.ResponseWriter, r *http.Request) {
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

func (h *DeliveryWithdrawalSignatureHandler) Lookup(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Code string `json:"code"`
	}
	if json.NewDecoder(r.Body).Decode(&input) != nil {
		writeError(w, http.StatusBadRequest, "Código inválido", "VALIDATION_ERROR")
		return
	}
	result, err := h.service.Lookup(r.Context(), strings.TrimSpace(input.Code))
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *DeliveryWithdrawalSignatureHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Code      string `json:"code"`
		Signature string `json:"signature"`
	}
	if json.NewDecoder(r.Body).Decode(&input) != nil {
		writeError(w, http.StatusBadRequest, "Dados inválidos", "VALIDATION_ERROR")
		return
	}
	if err := h.service.Confirm(r.Context(), strings.TrimSpace(input.Code), input.Signature); err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "signed"})
}

func (h *DeliveryWithdrawalSignatureHandler) Status(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.Status(r.Context(), r.PathValue("id"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *DeliveryWithdrawalSignatureHandler) writeError(w http.ResponseWriter, err error) {
	if errors.Is(err, usecase.ErrSignatureCodeInvalid) {
		writeError(w, http.StatusGone, "Código inválido, expirado ou já utilizado", "SIGNATURE_CODE_INVALID")
		return
	}
	if errors.Is(err, domain.ErrInvalidInput) {
		writeDomainError(w, err)
		return
	}
	writeError(w, http.StatusInternalServerError, "Não foi possível concluir a retirada", "INTERNAL_ERROR")
}
