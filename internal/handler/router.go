package handler

import (
	"net/http"

	"chateauneuf-portaria-backend/internal/usecase"
)

type RouterDeps struct {
	AccessLogService *usecase.AccessLogService
	ResidentService  *usecase.ResidentService
	KeyService       *usecase.KeyService
	DiaristaService  *usecase.DiaristaService
	ScheduledService *usecase.ScheduledServiceService
	SyncService      usecase.SyncService
	AllowedOrigin    string
}

func NewRouter(deps RouterDeps) http.Handler {
	accessLogHandler := NewAccessLogHandler(deps.AccessLogService)
	residentHandler := NewResidentHandler(deps.ResidentService)
	keyHandler := NewKeyHandler(deps.KeyService)
	diaristaHandler := NewDiaristaHandler(deps.DiaristaService)
	scheduledServiceHandler := NewScheduledServiceHandler(deps.ScheduledService)
	syncHandler := NewSyncHandler(deps.SyncService)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/access-logs", accessLogHandler.Create)
	mux.HandleFunc("GET /api/access-logs", accessLogHandler.List)
	mux.HandleFunc("GET /api/access-logs/open", accessLogHandler.ListOpen)
	mux.HandleFunc("PATCH /api/access-logs/{id}/checkout", accessLogHandler.Checkout)
	mux.HandleFunc("GET /api/residents", residentHandler.List)
	mux.HandleFunc("POST /api/residents", residentHandler.Upsert)
	mux.HandleFunc("GET /api/keys", keyHandler.List)
	mux.HandleFunc("POST /api/keys", keyHandler.Create)
	mux.HandleFunc("POST /api/keys/return", keyHandler.Return)
	mux.HandleFunc("POST /api/keys/delete", keyHandler.Delete)
	mux.HandleFunc("GET /api/diaristas", diaristaHandler.List)
	mux.HandleFunc("POST /api/diaristas", diaristaHandler.Create)
	mux.HandleFunc("POST /api/diaristas/exit", diaristaHandler.Checkout)
	mux.HandleFunc("GET /api/scheduled-services", scheduledServiceHandler.List)
	mux.HandleFunc("POST /api/scheduled-services", scheduledServiceHandler.Create)
	mux.HandleFunc("POST /api/scheduled-services/status", scheduledServiceHandler.UpdateStatus)
	mux.HandleFunc("POST /api/scheduled-services/delete", scheduledServiceHandler.Delete)
	mux.HandleFunc("GET /api/sync/status", syncHandler.Status)
	mux.HandleFunc("POST /api/sync/run", syncHandler.Run)

	return corsMiddleware(deps.AllowedOrigin, jsonMiddleware(mux))
}

func jsonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

func corsMiddleware(allowedOrigin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if allowedOrigin != "" {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PATCH,OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
