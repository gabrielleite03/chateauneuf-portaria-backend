package usecase

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"chateauneuf-portaria-backend/internal/database"
	"chateauneuf-portaria-backend/internal/domain"
)

func TestReservationGuestsPersistenceAndIsolation(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "guests.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(db, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	for _, unit := range []string{"101", "102"} {
		if _, err := db.Exec(`INSERT INTO common_area_reservations (area,resident_name,unit,reservation_date,start_time,end_time,status) VALUES ('Churrasqueira','Morador',?,'2026-10-01','09:00','18:00','concluida')`, unit); err != nil {
			t.Fatal(err)
		}
	}
	ctx := context.Background()
	service := NewReservationGuestService(db)
	if _, err := service.Add(ctx, "1", " ", "123"); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("empty name: %v", err)
	}
	if _, err := service.Add(ctx, "1", "Ana", " "); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("empty document: %v", err)
	}
	if _, err := service.Add(ctx, "999", "Ana", "123"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("missing reservation: %v", err)
	}
	guest, err := service.Add(ctx, "1", " Ana ", " 00123-X ")
	if err != nil {
		t.Fatal(err)
	}
	if guest.Name != "Ana" || guest.Document != "00123-X" || guest.Confirmed {
		t.Fatalf("unexpected guest: %+v", guest)
	}
	if err := service.Confirm(ctx, "2", guest.ID, true); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("cross-reservation update: %v", err)
	}
	if err := service.Confirm(ctx, "1", guest.ID, true); err != nil {
		t.Fatal(err)
	}
	rows, err := NewReservationGuestService(db).List(ctx, "1")
	if err != nil || len(rows) != 1 || !rows[0].Confirmed {
		t.Fatalf("confirmation not persisted: %+v %v", rows, err)
	}
	other, err := service.List(ctx, "2")
	if err != nil || len(other) != 0 {
		t.Fatalf("reservation isolation: %+v %v", other, err)
	}
	if err := service.Confirm(ctx, "1", guest.ID, false); err != nil {
		t.Fatal(err)
	}
	rows, err = service.List(ctx, "1")
	if err != nil || len(rows) != 1 || rows[0].Confirmed {
		t.Fatalf("uncheck not persisted: %+v %v", rows, err)
	}
	if _, err := db.Exec(`DELETE FROM common_area_reservations WHERE id = 1`); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM reservation_guests`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("orphan guests: %d %v", count, err)
	}
}
