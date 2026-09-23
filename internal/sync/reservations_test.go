package sync

import (
	"chateauneuf-portaria-backend/internal/database"
	"chateauneuf-portaria-backend/internal/domain"
	"chateauneuf-portaria-backend/internal/repository"
	"context"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
)

type reservationClient struct {
	SpreadsheetClient
	fail   bool
	sent   []domain.CommonAreaReservation
	onSend func()
}

func (*reservationClient) Ping(context.Context) error { return nil }
func (c *reservationClient) AppendReservation(_ context.Context, r domain.CommonAreaReservation) error {
	if c.fail {
		return errors.New("sheets unavailable")
	}
	c.sent = append(c.sent, r)
	if c.onSend != nil {
		c.onSend()
	}
	return nil
}

func TestReservationSyncRetryAndConcurrentCancellation(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(filepath.Join(t.TempDir(), "sync.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(db, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	repo := repository.NewSQLiteReservationRepository(db)
	created, err := repo.Create(ctx, domain.CommonAreaReservation{Area: "Churrasqueira", Unit: "101", ResidentName: "Test",
		ReservationDate: "2026-10-01", StartTime: "09:00", EndTime: "18:00", Status: domain.ReservationStatusBooked, SyncStatus: domain.SyncStatusPending})
	if err != nil {
		t.Fatal(err)
	}
	client := &reservationClient{fail: true}
	s := NewService(repository.NewSQLiteAccessLogRepository(db), nil, nil, nil, nil, nil, repo, client, slog.New(slog.NewTextHandler(io.Discard, nil)))
	status, err := s.Status(ctx)
	if err != nil || status.PendingCount != 1 {
		t.Fatalf("missing pending reservation: %+v %v", status, err)
	}
	if err := s.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	status, err = s.Status(ctx)
	if err != nil || status.PendingCount != 1 || status.LastError != "sheets unavailable" {
		t.Fatalf("missing sync error: %+v %v", status, err)
	}
	client.fail = false
	client.onSend = func() {
		if _, err := repo.UpdateStatus(ctx, created.ID, domain.ReservationStatusCanceled); err != nil {
			t.Fatal(err)
		}
		client.onSend = nil
	}
	if err := s.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	status, err = s.Status(ctx)
	if err != nil || status.PendingCount != 1 {
		t.Fatalf("concurrent change was incorrectly acknowledged: %+v %v", status, err)
	}
	if err := s.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	status, err = s.Status(ctx)
	if err != nil || status.PendingCount != 0 || status.LastError != "" || status.LastSyncedAt == nil {
		t.Fatalf("retry did not clear queue: %+v %v", status, err)
	}
	if len(client.sent) != 2 || client.sent[1].Status != domain.ReservationStatusCanceled {
		t.Fatalf("cancellation not sent: %+v", client.sent)
	}
	if err := s.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if len(client.sent) != 2 {
		t.Fatal("already synced reservation sent again")
	}
}
