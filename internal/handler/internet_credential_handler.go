package handler

import (
	"chateauneuf-portaria-backend/internal/usecase"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

type InternetCredentialHandler struct {
	service *usecase.InternetCredentialService
	token   string
}
type internetCredentialInput struct {
	Apartment string    `json:"apartment"`
	Username  string    `json:"username"`
	Password  string    `json:"password"`
	ExpiresAt time.Time `json:"expires_at"`
}

func NewInternetCredentialHandler(service *usecase.InternetCredentialService, token string) *InternetCredentialHandler {
	return &InternetCredentialHandler{service: service, token: token}
}
func (h *InternetCredentialHandler) Send(w http.ResponseWriter, r *http.Request) {
	got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if h.token == "" || len(got) != len(h.token) || subtle.ConstantTimeCompare([]byte(got), []byte(h.token)) != 1 {
		writeError(w, http.StatusUnauthorized, "nao autorizado", "UNAUTHORIZED")
		return
	}
	var input internetCredentialInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "JSON invalido", "VALIDATION_ERROR")
		return
	}
	_, err := h.service.Send(r.Context(), input.Apartment, input.Username, input.Password, input.ExpiresAt)
	if errors.Is(err, usecase.ErrInternetCredentialEmailRequired) {
		writeError(w, http.StatusUnprocessableEntity, err.Error(), "RESIDENT_EMAIL_REQUIRED")
		return
	}
	if err != nil {
		writeError(w, http.StatusBadGateway, "nao foi possivel enviar o e-mail: "+err.Error(), "EMAIL_SEND_FAILED")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
