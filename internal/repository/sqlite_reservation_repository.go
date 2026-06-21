package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"chateauneuf-portaria-backend/internal/domain"
)

type SQLiteReservationRepository struct {
	db *sql.DB
}

func NewSQLiteReservationRepository(db *sql.DB) *SQLiteReservationRepository {
	return &SQLiteReservationRepository{db: db}
}

func (r *SQLiteReservationRepository) List(ctx context.Context) ([]domain.CommonAreaReservation, error) {
	return r.queryReservations(ctx, `
		SELECT id, area, resident_name, unit, reservation_date, start_time, end_time,
			guests, notes, status, sync_status, created_at, updated_at
		FROM common_area_reservations
		ORDER BY reservation_date DESC, start_time DESC, id DESC
	`)
}

func (r *SQLiteReservationRepository) Create(ctx context.Context, reservation domain.CommonAreaReservation) (*domain.CommonAreaReservation, error) {
	now := time.Now().Round(0)
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO common_area_reservations (
			area, resident_name, unit, reservation_date, start_time, end_time,
			guests, notes, status, sync_status, sync_error, created_at, updated_at, synced_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, '', ?, ?, ?)
	`, reservation.Area, reservation.ResidentName, reservation.Unit, reservation.ReservationDate,
		reservation.StartTime, reservation.EndTime, reservation.Guests, reservation.Notes,
		reservation.Status, reservation.SyncStatus, now, now, nil)
	if err != nil {
		return nil, fmt.Errorf("create common area reservation: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("read common area reservation id: %w", err)
	}
	return r.FindByID(ctx, id)
}

func (r *SQLiteReservationRepository) UpdateStatus(ctx context.Context, id string, status domain.ReservationStatus) (*domain.CommonAreaReservation, error) {
	numericID, err := parseReservationID(id)
	if err != nil {
		return nil, domain.ErrInvalidInput
	}

	now := time.Now().Round(0)
	result, err := r.db.ExecContext(ctx, `
		UPDATE common_area_reservations
		SET status = ?, sync_status = ?, sync_error = '', updated_at = ?
		WHERE id = ?
	`, status, domain.SyncStatusPending, now, numericID)
	if err != nil {
		return nil, fmt.Errorf("update common area reservation status: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("read common area reservation update rows affected: %w", err)
	}
	if affected == 0 {
		return nil, domain.ErrNotFound
	}

	return r.FindByID(ctx, numericID)
}

func (r *SQLiteReservationRepository) Delete(ctx context.Context, id string) error {
	numericID, err := parseReservationID(id)
	if err != nil {
		return domain.ErrInvalidInput
	}

	result, err := r.db.ExecContext(ctx, `DELETE FROM common_area_reservations WHERE id = ?`, numericID)
	if err != nil {
		return fmt.Errorf("delete common area reservation: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read common area reservation delete rows affected: %w", err)
	}
	if affected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *SQLiteReservationRepository) FindByID(ctx context.Context, id int64) (*domain.CommonAreaReservation, error) {
	reservations, err := r.queryReservations(ctx, `
		SELECT id, area, resident_name, unit, reservation_date, start_time, end_time,
			guests, notes, status, sync_status, created_at, updated_at
		FROM common_area_reservations
		WHERE id = ?
	`, id)
	if err != nil {
		return nil, err
	}
	if len(reservations) == 0 {
		return nil, domain.ErrNotFound
	}
	return &reservations[0], nil
}

func (r *SQLiteReservationRepository) queryReservations(ctx context.Context, query string, args ...any) ([]domain.CommonAreaReservation, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query common area reservations: %w", err)
	}
	defer rows.Close()

	reservations := make([]domain.CommonAreaReservation, 0)
	for rows.Next() {
		var reservation domain.CommonAreaReservation
		var id int64
		if err := rows.Scan(
			&id,
			&reservation.Area,
			&reservation.ResidentName,
			&reservation.Unit,
			&reservation.ReservationDate,
			&reservation.StartTime,
			&reservation.EndTime,
			&reservation.Guests,
			&reservation.Notes,
			&reservation.Status,
			&reservation.SyncStatus,
			&reservation.CreatedAt,
			&reservation.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan common area reservation: %w", err)
		}
		reservation.ID = strconv.FormatInt(id, 10)
		reservation.SyncStatus = reservationClientSyncStatus(reservation.SyncStatus)
		reservations = append(reservations, reservation)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate common area reservations: %w", err)
	}

	return reservations, nil
}

func parseReservationID(id string) (int64, error) {
	cleanID := strings.TrimPrefix(strings.TrimSpace(id), "r-")
	return strconv.ParseInt(cleanID, 10, 64)
}

func reservationClientSyncStatus(status domain.SyncStatus) domain.SyncStatus {
	switch status {
	case domain.SyncStatusSynced:
		return domain.SyncStatus("synced")
	case domain.SyncStatusError:
		return domain.SyncStatus("failed")
	default:
		return domain.SyncStatus("pending")
	}
}
