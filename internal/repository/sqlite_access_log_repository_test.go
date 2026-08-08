package repository

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"chateauneuf-portaria-backend/internal/database"
	"chateauneuf-portaria-backend/internal/domain"
)

func TestAccessLogRejectsDuplicateActiveDocumentAndAllowsAfterCheckout(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "access-log-test.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()

	if err := database.Migrate(db, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatalf("migrate database: %v", err)
	}

	repository := NewSQLiteAccessLogRepository(db)
	first := activeAccessLog("123.456.789-00")
	if err := repository.Create(context.Background(), first); err != nil {
		t.Fatalf("create first access log: %v", err)
	}

	exists, err := repository.HasOpenByDocument(context.Background(), "12345678900")
	if err != nil || !exists {
		t.Fatalf("formatted document should be found as active: exists=%v err=%v", exists, err)
	}

	duplicate := activeAccessLog("12345678900")
	if err := repository.Create(context.Background(), duplicate); !errors.Is(err, domain.ErrActiveVisitExists) {
		t.Fatalf("expected active visit conflict, got %v", err)
	}

	if _, err := repository.Checkout(context.Background(), first.ID, time.Now()); err != nil {
		t.Fatalf("checkout first access log: %v", err)
	}
	if err := repository.Create(context.Background(), duplicate); err != nil {
		t.Fatalf("new access log should be allowed after checkout: %v", err)
	}
}

func activeAccessLog(document string) *domain.AccessLog {
	now := time.Now()
	return &domain.AccessLog{
		ExternalID:  "visit-" + document,
		VisitorName: "Visitante Teste",
		Document:    document,
		Unit:        "101",
		EntryAt:     now,
		VisitStatus: domain.VisitStatusInProgress,
		SyncStatus:  domain.SyncStatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}
