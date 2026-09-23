package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"chateauneuf-portaria-backend/internal/usecase"
)

type DeliveryPhotoHandler struct{ service *usecase.DeliveryPhotoService }

func NewDeliveryPhotoHandler(service *usecase.DeliveryPhotoService) *DeliveryPhotoHandler {
	return &DeliveryPhotoHandler{service: service}
}

func (h *DeliveryPhotoHandler) Create(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.Create(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "nao foi possivel iniciar a coleta da foto", "INTERNAL_ERROR")
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (h *DeliveryPhotoHandler) Upload(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Code  string `json:"code"`
		Photo string `json:"photo"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 3<<20)).Decode(&input) != nil {
		writeError(w, http.StatusBadRequest, "foto invalida", "VALIDATION_ERROR")
		return
	}
	if err := h.service.Upload(r.Context(), input.Code, input.Photo); err != nil {
		if errors.Is(err, usecase.ErrDeliveryPhotoCodeInvalid) {
			writeError(w, http.StatusGone, "codigo invalido, expirado ou ja utilizado", "DELIVERY_PHOTO_CODE_INVALID")
			return
		}
		writeError(w, http.StatusInternalServerError, "nao foi possivel receber a foto", "INTERNAL_ERROR")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "uploaded"})
}

func (h *DeliveryPhotoHandler) Status(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.Status(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusGone, "sessao de foto invalida", "DELIVERY_PHOTO_SESSION_INVALID")
		return
	}
	writeJSON(w, http.StatusOK, result)
}
