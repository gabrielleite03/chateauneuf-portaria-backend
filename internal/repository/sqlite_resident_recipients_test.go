package repository

import (
	"context"
	"path/filepath"
	"testing"

	"chateauneuf-portaria-backend/internal/database"
	"chateauneuf-portaria-backend/internal/domain"
)

func TestResidentAuthorizedRecipientsPersistence(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "residents.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(db, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	repo := NewSQLiteResidentRepository(db)
	ctx := context.Background()
	for _, names := range []string{"Ana Silva\nJoão Souza", "Ana Silva", ""} {
		got, err := repo.Upsert(ctx, domain.Resident{Unit: "11", AuthorizedRecipients: names, SyncStatus: domain.SyncStatusPending})
		if err != nil {
			t.Fatal(err)
		}
		if got.AuthorizedRecipients != names {
			t.Fatalf("saved names = %q, want %q", got.AuthorizedRecipients, names)
		}
		rows, err := repo.List(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 1 || rows[0].AuthorizedRecipients != names {
			t.Fatalf("unexpected residents: %+v", rows)
		}
		pending, err := repo.ListPendingSync(ctx, 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(pending) != 1 || pending[0].AuthorizedRecipients != names {
			t.Fatalf("unexpected pending residents: %+v", pending)
		}
		if _, err := repo.UpsertImported(ctx, *got); err != nil {
			t.Fatal(err)
		}
	}
}
