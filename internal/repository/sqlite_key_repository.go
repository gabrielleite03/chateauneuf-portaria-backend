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

type SQLiteKeyRepository struct {
	db *sql.DB
}

func NewSQLiteKeyRepository(db *sql.DB) *SQLiteKeyRepository {
	return &SQLiteKeyRepository{db: db}
}

func (r *SQLiteKeyRepository) List(ctx context.Context) ([]domain.KeyRecord, error) {
	return r.queryKeys(ctx, `
		SELECT id, date, local, resident_name, unit, pickup_time, return_time, gatekeeper, status, sync_status, created_at, updated_at
		FROM key_records
		ORDER BY date DESC, pickup_time DESC, id DESC
	`)
}

func (r *SQLiteKeyRepository) Create(ctx context.Context, key domain.KeyRecord) (*domain.KeyRecord, error) {
	now := time.Now()
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO key_records (
			date, local, resident_name, unit, pickup_time, return_time, gatekeeper, status,
			sync_status, sync_error, created_at, updated_at, synced_at
		) VALUES (?, ?, ?, ?, ?, '', ?, ?, ?, '', ?, ?, ?)
	`, key.Date, key.Local, key.ResidentName, key.Unit, key.PickupTime, key.Gatekeeper,
		key.Status, key.SyncStatus, now, now, nil)
	if err != nil {
		return nil, fmt.Errorf("create key record: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("read key record id: %w", err)
	}
	return r.FindByID(ctx, id)
}

func (r *SQLiteKeyRepository) Return(ctx context.Context, id string, returnTime string) (*domain.KeyRecord, error) {
	numericID, err := parseKeyID(id)
	if err != nil {
		return nil, domain.ErrInvalidInput
	}

	result, err := r.db.ExecContext(ctx, `
		UPDATE key_records
		SET return_time = ?, status = ?, sync_status = ?, sync_error = '', updated_at = ?
		WHERE id = ? AND status = ?
	`, returnTime, domain.KeyStatusReturned, domain.SyncStatusPending, time.Now(), numericID, domain.KeyStatusPickedUp)
	if err != nil {
		return nil, fmt.Errorf("return key record: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("read return key rows affected: %w", err)
	}
	if affected == 0 {
		return nil, domain.ErrNotFound
	}

	return r.FindByID(ctx, numericID)
}

func (r *SQLiteKeyRepository) ListPendingSync(ctx context.Context, limit int) ([]domain.KeyRecord, error) {
	if limit <= 0 {
		limit = 50
	}
	return r.queryKeys(ctx, `
		SELECT id, date, local, resident_name, unit, pickup_time, return_time, gatekeeper, status, sync_status, created_at, updated_at
		FROM key_records
		WHERE sync_status IN (?, ?)
		ORDER BY updated_at ASC
		LIMIT ?
	`, domain.SyncStatusPending, domain.SyncStatusError, limit)
}

func (r *SQLiteKeyRepository) MarkSynced(ctx context.Context, id string, syncedAt time.Time) error {
	numericID, err := parseKeyID(id)
	if err != nil {
		return domain.ErrInvalidInput
	}

	_, err = r.db.ExecContext(ctx, `
		UPDATE key_records
		SET sync_status = ?, sync_error = '', synced_at = ?, updated_at = ?
		WHERE id = ?
	`, domain.SyncStatusSynced, syncedAt.Round(0), time.Now().Round(0), numericID)
	if err != nil {
		return fmt.Errorf("mark key record synced: %w", err)
	}
	return nil
}

func (r *SQLiteKeyRepository) MarkSyncError(ctx context.Context, id string, syncError string) error {
	numericID, err := parseKeyID(id)
	if err != nil {
		return domain.ErrInvalidInput
	}

	_, err = r.db.ExecContext(ctx, `
		UPDATE key_records
		SET sync_status = ?, sync_error = ?, updated_at = ?
		WHERE id = ?
	`, domain.SyncStatusError, strings.TrimSpace(syncError), time.Now(), numericID)
	if err != nil {
		return fmt.Errorf("mark key record sync error: %w", err)
	}
	return nil
}

func (r *SQLiteKeyRepository) SyncStats(ctx context.Context) (int, error) {
	var pendingCount int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(CASE WHEN sync_status IN ('PENDENTE_SYNC', 'ERRO_SYNC') THEN 1 END)
		FROM key_records
	`).Scan(&pendingCount)
	if err != nil {
		return 0, fmt.Errorf("read key sync stats: %w", err)
	}
	return pendingCount, nil
}

func (r *SQLiteKeyRepository) Delete(ctx context.Context, id string) error {
	numericID, err := parseKeyID(id)
	if err != nil {
		return domain.ErrInvalidInput
	}

	result, err := r.db.ExecContext(ctx, `DELETE FROM key_records WHERE id = ?`, numericID)
	if err != nil {
		return fmt.Errorf("delete key record: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read delete key rows affected: %w", err)
	}
	if affected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *SQLiteKeyRepository) FindByID(ctx context.Context, id int64) (*domain.KeyRecord, error) {
	keys, err := r.queryKeys(ctx, `
		SELECT id, date, local, resident_name, unit, pickup_time, return_time, gatekeeper, status, sync_status, created_at, updated_at
		FROM key_records
		WHERE id = ?
	`, id)
	if err != nil {
		return nil, err
	}
	if len(keys) == 0 {
		return nil, domain.ErrNotFound
	}
	return &keys[0], nil
}

func (r *SQLiteKeyRepository) queryKeys(ctx context.Context, query string, args ...any) ([]domain.KeyRecord, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query key records: %w", err)
	}
	defer rows.Close()

	keys := make([]domain.KeyRecord, 0)
	for rows.Next() {
		var key domain.KeyRecord
		var id int64
		if err := rows.Scan(
			&id,
			&key.Date,
			&key.Local,
			&key.ResidentName,
			&key.Unit,
			&key.PickupTime,
			&key.ReturnTime,
			&key.Gatekeeper,
			&key.Status,
			&key.SyncStatus,
			&key.CreatedAt,
			&key.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan key record: %w", err)
		}
		key.ID = strconv.FormatInt(id, 10)
		key.SyncStatus = keyClientSyncStatus(key.SyncStatus)
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate key records: %w", err)
	}

	return keys, nil
}

func parseKeyID(id string) (int64, error) {
	cleanID := strings.TrimPrefix(strings.TrimSpace(id), "k-")
	return strconv.ParseInt(cleanID, 10, 64)
}

func keyClientSyncStatus(status domain.SyncStatus) domain.SyncStatus {
	switch status {
	case domain.SyncStatusSynced:
		return domain.SyncStatus("synced")
	case domain.SyncStatusError:
		return domain.SyncStatus("failed")
	default:
		return domain.SyncStatus("pending")
	}
}
