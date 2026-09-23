package repository

import (
	"context"
	"errors"
	"fmt"
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

func TestImportAccessLogPreservesCheckoutInEitherRowOrder(t *testing.T) {
	for _, externalID := range []string{"", "visit-import-test"} {
		for _, closedFirst := range []bool{false, true} {
			t.Run(fmt.Sprintf("external=%s/closedFirst=%v", externalID, closedFirst), func(t *testing.T) {
				db, err := database.Open(filepath.Join(t.TempDir(), "import.db"))
				if err != nil {
					t.Fatal(err)
				}
				defer db.Close()
				if err := database.Migrate(db, filepath.Join("..", "..", "migrations")); err != nil {
					t.Fatal(err)
				}
				repo := NewSQLiteAccessLogRepository(db)
				open := *activeAccessLog("12345678900")
				open.ID, open.ExternalID, open.SyncStatus = 37, externalID, domain.SyncStatusSynced
				closed := open
				exit := open.EntryAt.Add(time.Hour)
				closed.ExitAt, closed.VisitStatus, closed.UpdatedAt = &exit, domain.VisitStatusFinished, exit
				rows := []domain.AccessLog{open, open, closed, open}
				if closedFirst {
					rows = []domain.AccessLog{closed, open, closed, open}
				}
				for _, row := range rows {
					if err := repo.UpsertImported(context.Background(), &row); err != nil {
						t.Fatal(err)
					}
				}
				logs, err := repo.List(context.Background(), domain.AccessLogFilters{})
				if err != nil {
					t.Fatal(err)
				}
				if len(logs) != 1 || logs[0].ExitAt == nil || !logs[0].ExitAt.Equal(exit) || logs[0].VisitStatus != domain.VisitStatusFinished {
					t.Fatalf("expected one completed visit after duplicate imports, got %+v", logs)
				}
				// A later active visit must not cause a stale duplicate of this
				// completed visit to fail the import or reopen the old visit.
				next := activeAccessLog("12345678900")
				next.ExternalID = "next-visit"
				if err := repo.Create(context.Background(), next); err != nil {
					t.Fatal(err)
				}
				if err := repo.UpsertImported(context.Background(), &open); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestImportAccessLogPreservesUnsyncedChanges(t *testing.T) {
	for _, status := range []domain.SyncStatus{domain.SyncStatusPending, domain.SyncStatusError} {
		t.Run(string(status), func(t *testing.T) {
			db, err := database.Open(filepath.Join(t.TempDir(), "import.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if err := database.Migrate(db, filepath.Join("..", "..", "migrations")); err != nil {
				t.Fatal(err)
			}
			repo := NewSQLiteAccessLogRepository(db)
			local := activeAccessLog("12345678900")
			local.SyncStatus = status
			if err := repo.Create(context.Background(), local); err != nil {
				t.Fatal(err)
			}
			imported := *local
			imported.VisitorName, imported.SyncStatus = "Outdated name", domain.SyncStatusSynced
			if err := repo.UpsertImported(context.Background(), &imported); err != nil {
				t.Fatal(err)
			}
			got, err := repo.FindByID(context.Background(), local.ID)
			if err != nil {
				t.Fatal(err)
			}
			if got.VisitorName != local.VisitorName || got.SyncStatus != status {
				t.Fatalf("local unsynced changes were overwritten: %+v", got)
			}
		})
	}
}
