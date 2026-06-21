package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"chateauneuf-portaria-backend/internal/domain"
	"chateauneuf-portaria-backend/internal/photos"
	"chateauneuf-portaria-backend/internal/usecase"
)

type AccessLogHandler struct {
	service    *usecase.AccessLogService
	photoStore *photos.Store
}

func NewAccessLogHandler(service *usecase.AccessLogService, photoStore *photos.Store) *AccessLogHandler {
	return &AccessLogHandler{service: service, photoStore: photoStore}
}

func (h *AccessLogHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input usecase.CreateAccessLogInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "JSON invalido", "VALIDATION_ERROR")
		return
	}
	if h.photoStore != nil {
		photo, err := h.photoStore.SaveDataURL(r.Context(), "entradas", input.Photo)
		if err != nil {
			writeError(w, http.StatusBadRequest, "foto invalida", "VALIDATION_ERROR")
			return
		}
		input.Photo = photo
	}

	accessLog, err := h.service.Create(r.Context(), input)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, accessLog)
}

func (h *AccessLogHandler) List(w http.ResponseWriter, r *http.Request) {
	filters := domain.AccessLogFilters{
		Date:        r.URL.Query().Get("date"),
		Unit:        r.URL.Query().Get("unit"),
		Status:      domain.VisitStatus(r.URL.Query().Get("status")),
		VisitorName: r.URL.Query().Get("visitor_name"),
		Document:    r.URL.Query().Get("document"),
	}

	accessLogs, err := h.service.List(r.Context(), filters)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, accessLogs)
}

func (h *AccessLogHandler) ListOpen(w http.ResponseWriter, r *http.Request) {
	accessLogs, err := h.service.ListOpen(r.Context())
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, accessLogs)
}

func (h *AccessLogHandler) Checkout(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalido", "VALIDATION_ERROR")
		return
	}

	accessLog, err := h.service.Checkout(r.Context(), id)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, accessLog)
}

func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "dados invalidos", "VALIDATION_ERROR")
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, "registro nao encontrado", "NOT_FOUND")
	default:
		writeError(w, http.StatusInternalServerError, "erro interno", "INTERNAL_ERROR")
	}
}
