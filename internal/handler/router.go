package handler

import (
	"net/http"
	"strings"

	"chateauneuf-portaria-backend/internal/photos"
	"chateauneuf-portaria-backend/internal/usecase"
	"chateauneuf-portaria-backend/internal/version"
)

type RouterDeps struct {
	InventoryService            *usecase.InventoryService
	ResidentVehicleService      *usecase.ResidentVehicleService
	ReservationGuestService     *usecase.ReservationGuestService
	AccessLogService            *usecase.AccessLogService
	ResidentService             *usecase.ResidentService
	KeyService                  *usecase.KeyService
	DiaristaService             *usecase.DiaristaService
	ScheduledService            *usecase.ScheduledServiceService
	ShoppingService             *usecase.ShoppingService
	ReservationService          *usecase.ReservationService
	ReservationSignatureService *usecase.ReservationSignatureService
	DeliveryPhotoService        *usecase.DeliveryPhotoService
	DeliveryWithdrawalService   *usecase.DeliveryWithdrawalSignatureService
	SyncService                 usecase.SyncService
	PhotoStore                  *photos.Store
	AllowedOrigin               string
	InternetCredentialService   *usecase.InternetCredentialService
	InternalAPIToken            string
}

func NewRouter(deps RouterDeps) http.Handler {
	accessLogHandler := NewAccessLogHandler(deps.AccessLogService, deps.PhotoStore)
	residentHandler := NewResidentHandler(deps.ResidentService)
	internetCredentialHandler := NewInternetCredentialHandler(deps.InternetCredentialService, deps.InternalAPIToken)
	keyHandler := NewKeyHandler(deps.KeyService)
	diaristaHandler := NewDiaristaHandler(deps.DiaristaService, deps.PhotoStore)
	scheduledServiceHandler := NewScheduledServiceHandler(deps.ScheduledService, deps.PhotoStore)
	shoppingHandler := NewShoppingHandler(deps.ShoppingService, deps.PhotoStore)
	reservationHandler := NewReservationHandler(deps.ReservationService)
	signatureHandler := NewReservationSignatureHandler(deps.ReservationSignatureService)
	deliveryPhotoHandler := NewDeliveryPhotoHandler(deps.DeliveryPhotoService)
	deliveryWithdrawalHandler := NewDeliveryWithdrawalSignatureHandler(deps.DeliveryWithdrawalService)
	syncHandler := NewSyncHandler(deps.SyncService)

	mux := http.NewServeMux()
	if deps.InventoryService != nil {
		inventory := &InventoryHandler{service: deps.InventoryService}
		mux.HandleFunc("GET /api/inventory", inventory.Snapshot)
		mux.HandleFunc("POST /api/inventory/products", inventory.Product)
		mux.HandleFunc("POST /api/inventory/products/{productID}", inventory.Product)
		mux.HandleFunc("POST /api/inventory/purchases", inventory.Purchase)
		mux.HandleFunc("POST /api/inventory/withdrawals", inventory.Withdraw)
	}
	if deps.ResidentVehicleService != nil {
		vehicles := &ResidentVehicleHandler{service: deps.ResidentVehicleService}
		mux.HandleFunc("GET /api/residents/{unit}/vehicles", vehicles.List)
		mux.HandleFunc("POST /api/residents/{unit}/vehicles", vehicles.Save)
		mux.HandleFunc("PUT /api/residents/{unit}/vehicles/{vehicleID}", vehicles.Save)
		mux.HandleFunc("DELETE /api/residents/{unit}/vehicles/{vehicleID}", vehicles.Delete)
	}
	if deps.ReservationGuestService != nil {
		guests := &ReservationGuestHandler{service: deps.ReservationGuestService}
		mux.HandleFunc("GET /api/reservations/{id}/guests", guests.List)
		mux.HandleFunc("POST /api/reservations/{id}/guests", guests.Add)
		mux.HandleFunc("PATCH /api/reservations/{id}/guests/{guestID}", guests.Confirm)
	}
	mux.HandleFunc("POST /api/access-logs", accessLogHandler.Create)
	mux.HandleFunc("GET /api/access-logs", accessLogHandler.List)
	mux.HandleFunc("GET /api/access-logs/open", accessLogHandler.ListOpen)
	mux.HandleFunc("PATCH /api/access-logs/{id}/checkout", accessLogHandler.Checkout)
	mux.HandleFunc("GET /api/residents", residentHandler.List)
	mux.HandleFunc("POST /api/residents", residentHandler.Upsert)
	mux.HandleFunc("POST /internal/residents/internet-credentials", internetCredentialHandler.Send)
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
	mux.HandleFunc("GET /api/shopping", shoppingHandler.List)
	mux.HandleFunc("POST /api/shopping", shoppingHandler.Create)
	mux.HandleFunc("POST /api/shopping/withdraw", shoppingHandler.Withdraw)
	mux.HandleFunc("GET /api/reservations", reservationHandler.List)
	mux.HandleFunc("POST /api/reservations", reservationHandler.Create)
	mux.HandleFunc("POST /api/reservations/status", reservationHandler.UpdateStatus)
	mux.HandleFunc("POST /api/reservations/delete", reservationHandler.Delete)
	mux.HandleFunc("POST /api/reservations/{id}/signature-code", signatureHandler.CreateCode)
	mux.HandleFunc("POST /api/signatures/lookup", signatureHandler.Lookup)
	mux.HandleFunc("POST /api/signatures/confirm", signatureHandler.Confirm)
	mux.HandleFunc("GET /api/reservations/{id}/signed-document", signatureHandler.Document)
	mux.HandleFunc("POST /api/delivery-photo-sessions", deliveryPhotoHandler.Create)
	mux.HandleFunc("GET /api/delivery-photo-sessions/{id}", deliveryPhotoHandler.Status)
	mux.HandleFunc("POST /api/delivery-photo-sessions/upload", deliveryPhotoHandler.Upload)
	mux.HandleFunc("POST /api/shopping/{id}/withdrawal-signature-code", deliveryWithdrawalHandler.CreateCode)
	mux.HandleFunc("GET /api/shopping/{id}/withdrawal-signature-status", deliveryWithdrawalHandler.Status)
	mux.HandleFunc("POST /api/delivery-withdrawal-signatures/lookup", deliveryWithdrawalHandler.Lookup)
	mux.HandleFunc("POST /api/delivery-withdrawal-signatures/confirm", deliveryWithdrawalHandler.Confirm)
	mux.HandleFunc("GET /api/sync/status", syncHandler.Status)
	mux.HandleFunc("POST /api/sync/run", syncHandler.Run)
	mux.HandleFunc("POST /api/sync/import", syncHandler.ImportAccessLogs)
	mux.HandleFunc("GET /api/version", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, version.Current())
	})
	if deps.PhotoStore != nil && deps.PhotoStore.Dir() != "" {
		mux.Handle("GET /api/photos/", http.StripPrefix("/api/photos/", http.FileServer(http.Dir(deps.PhotoStore.Dir()))))
	}

	return corsMiddleware(deps.AllowedOrigin, jsonMiddleware(mux))
}

func jsonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/photos/") {
			next.ServeHTTP(w, r)
			return
		}
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
