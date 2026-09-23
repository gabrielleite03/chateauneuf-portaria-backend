package usecase

import (
	"context"
	"database/sql"
	"strconv"
	"strings"

	"chateauneuf-portaria-backend/internal/domain"
)

type ReservationGuest struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Document  string `json:"document"`
	Confirmed bool   `json:"confirmed"`
}

type ReservationGuestService struct{ db *sql.DB }

func NewReservationGuestService(db *sql.DB) *ReservationGuestService {
	return &ReservationGuestService{db: db}
}

func (s *ReservationGuestService) List(ctx context.Context, reservationID string) ([]ReservationGuest, error) {
	var exists bool
	if err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM common_area_reservations WHERE id = ?)`, reservationID).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, domain.ErrNotFound
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, document, confirmed FROM reservation_guests WHERE reservation_id = ? ORDER BY id`, reservationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	guests := make([]ReservationGuest, 0)
	for rows.Next() {
		var g ReservationGuest
		if err := rows.Scan(&g.ID, &g.Name, &g.Document, &g.Confirmed); err != nil {
			return nil, err
		}
		guests = append(guests, g)
	}
	return guests, rows.Err()
}

func (s *ReservationGuestService) Add(ctx context.Context, reservationID, name, document string) (*ReservationGuest, error) {
	name, document = strings.TrimSpace(name), strings.TrimSpace(document)
	if name == "" || document == "" || len([]rune(name)) > 200 || len([]rune(document)) > 100 {
		return nil, domain.ErrInvalidInput
	}
	result, err := s.db.ExecContext(ctx, `INSERT INTO reservation_guests (reservation_id, name, document) SELECT id, ?, ? FROM common_area_reservations WHERE id = ?`, name, document, reservationID)
	if err != nil {
		return nil, err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, domain.ErrNotFound
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &ReservationGuest{ID: strconv.FormatInt(id, 10), Name: name, Document: document}, nil
}

func (s *ReservationGuestService) Confirm(ctx context.Context, reservationID, guestID string, confirmed bool) error {
	result, err := s.db.ExecContext(ctx, `UPDATE reservation_guests SET confirmed = ? WHERE reservation_id = ? AND id = ?`, confirmed, reservationID, guestID)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return domain.ErrNotFound
	}
	return nil
}
