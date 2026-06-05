package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"chateauneuf-portaria-backend/internal/config"
	"chateauneuf-portaria-backend/internal/database"
	"chateauneuf-portaria-backend/internal/google"
	"chateauneuf-portaria-backend/internal/handler"
	"chateauneuf-portaria-backend/internal/repository"
	syncworker "chateauneuf-portaria-backend/internal/sync"
	"chateauneuf-portaria-backend/internal/usecase"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg := config.Load()

	db, err := database.Open(cfg.DatabasePath)
	if err != nil {
		logger.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := database.Migrate(db, cfg.MigrationsPath); err != nil {
		logger.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	accessLogRepo := repository.NewSQLiteAccessLogRepository(db)
	residentRepo := repository.NewSQLiteResidentRepository(db)
	keyRepo := repository.NewSQLiteKeyRepository(db)
	diaristaRepo := repository.NewSQLiteDiaristaRepository(db)
	scheduledServiceRepo := repository.NewSQLiteScheduledServiceRepository(db)
	shoppingRepo := repository.NewSQLiteShoppingRepository(db)
	sheetsClient, err := google.NewSheetsClient(context.Background(), cfg.GoogleSpreadsheetID, cfg.GoogleSheetName, cfg.GoogleCredentialsFile)
	if err != nil {
		logger.Error("failed to create google sheets client", "error", err)
		os.Exit(1)
	}
	syncService := syncworker.NewService(accessLogRepo, diaristaRepo, keyRepo, scheduledServiceRepo, shoppingRepo, sheetsClient, logger)
	accessLogService := usecase.NewAccessLogService(accessLogRepo, syncService)
	residentService := usecase.NewResidentService(residentRepo)
	keyService := usecase.NewKeyService(keyRepo)
	diaristaService := usecase.NewDiaristaService(diaristaRepo)
	scheduledService := usecase.NewScheduledServiceService(scheduledServiceRepo)
	shoppingService := usecase.NewShoppingService(shoppingRepo)

	router := handler.NewRouter(handler.RouterDeps{
		AccessLogService: accessLogService,
		ResidentService:  residentService,
		KeyService:       keyService,
		DiaristaService:  diaristaService,
		ScheduledService: scheduledService,
		ShoppingService:  shoppingService,
		SyncService:      syncService,
		AllowedOrigin:    cfg.AllowedOrigin,
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	syncService.Start(ctx, cfg.SyncInterval)
	go func() {
		importCtx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		importedCount, err := syncService.ImportAccessLogs(importCtx)
		if err != nil {
			logger.Warn("initial access log import failed", "error", err)
			return
		}
		logger.Info("initial access log import finished", "imported_count", importedCount)
	}()

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("api listening", "addr", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server stopped unexpectedly", "error", err)
			stop()
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown failed", "error", err)
	}
}
