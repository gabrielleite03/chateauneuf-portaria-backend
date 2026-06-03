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

type SQLiteDiaristaRepository struct {
	db *sql.DB
}

func NewSQLiteDiaristaRepository(db *sql.DB) *SQLiteDiaristaRepository {
	return &SQLiteDiaristaRepository{db: db}
}

func (r *SQLiteDiaristaRepository) List(ctx context.Context) ([]domain.DiaristaEntry, error) {
	return r.queryDiaristas(ctx, `
		SELECT id, date, name, rg, unit, authorized_by, entry_time, exit_time, gatekeeper, photo, sync_status, created_at, updated_at
		FROM diarista_entries
		ORDER BY date DESC, entry_time DESC, id DESC
	`)
}

func (r *SQLiteDiaristaRepository) Create(ctx context.Context, entry domain.DiaristaEntry) (*domain.DiaristaEntry, error) {
	now := time.Now()
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO diarista_entries (
			date, name, rg, unit, authorized_by, entry_time, exit_time, gatekeeper, photo,
			sync_status, sync_error, created_at, updated_at, synced_at
		) VALUES (?, ?, ?, ?, ?, ?, '', ?, ?, ?, '', ?, ?, ?)
	`, entry.Date, entry.Name, entry.RG, entry.Unit, entry.AuthorizedBy, entry.EntryTime,
		entry.Gatekeeper, entry.Photo, entry.SyncStatus, now, now, nil)
	if err != nil {
		return nil, fmt.Errorf("create diarista entry: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("read diarista entry id: %w", err)
	}
	return r.FindByID(ctx, id)
}

func (r *SQLiteDiaristaRepository) Checkout(ctx context.Context, id string, exitTime string) (*domain.DiaristaEntry, error) {
	numericID, err := parseDiaristaID(id)
	if err != nil {
		return nil, domain.ErrInvalidInput
	}

	result, err := r.db.ExecContext(ctx, `
		UPDATE diarista_entries
		SET exit_time = ?, sync_status = ?, sync_error = '', updated_at = ?
		WHERE id = ? AND exit_time = ''
	`, exitTime, domain.SyncStatusPending, time.Now(), numericID)
	if err != nil {
		return nil, fmt.Errorf("checkout diarista entry: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("read diarista checkout rows affected: %w", err)
	}
	if affected == 0 {
		return nil, domain.ErrNotFound
	}
	return r.FindByID(ctx, numericID)
}

func (r *SQLiteDiaristaRepository) ListPendingSync(ctx context.Context, limit int) ([]domain.DiaristaEntry, error) {
	if limit <= 0 {
		limit = 50
	}
	return r.queryDiaristas(ctx, `
		SELECT id, date, name, rg, unit, authorized_by, entry_time, exit_time, gatekeeper, photo, sync_status, created_at, updated_at
		FROM diarista_entries
		WHERE sync_status IN (?, ?)
		ORDER BY updated_at ASC
		LIMIT ?
	`, domain.SyncStatusPending, domain.SyncStatusError, limit)
}

func (r *SQLiteDiaristaRepository) MarkSynced(ctx context.Context, id string, syncedAt time.Time) error {
	numericID, err := parseDiaristaID(id)
	if err != nil {
		return domain.ErrInvalidInput
	}

	_, err = r.db.ExecContext(ctx, `
		UPDATE diarista_entries
		SET sync_status = ?, sync_error = '', synced_at = ?, updated_at = ?
		WHERE id = ?
	`, domain.SyncStatusSynced, syncedAt.Round(0), time.Now().Round(0), numericID)
	if err != nil {
		return fmt.Errorf("mark diarista entry synced: %w", err)
	}
	return nil
}

func (r *SQLiteDiaristaRepository) MarkSyncError(ctx context.Context, id string, syncError string) error {
	numericID, err := parseDiaristaID(id)
	if err != nil {
		return domain.ErrInvalidInput
	}

	_, err = r.db.ExecContext(ctx, `
		UPDATE diarista_entries
		SET sync_status = ?, sync_error = ?, updated_at = ?
		WHERE id = ?
	`, domain.SyncStatusError, strings.TrimSpace(syncError), time.Now(), numericID)
	if err != nil {
		return fmt.Errorf("mark diarista entry sync error: %w", err)
	}
	return nil
}

func (r *SQLiteDiaristaRepository) SyncStats(ctx context.Context) (int, error) {
	var pendingCount int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(CASE WHEN sync_status IN ('PENDENTE_SYNC', 'ERRO_SYNC') THEN 1 END)
		FROM diarista_entries
	`).Scan(&pendingCount)
	if err != nil {
		return 0, fmt.Errorf("read diarista sync stats: %w", err)
	}
	return pendingCount, nil
}

func (r *SQLiteDiaristaRepository) FindByID(ctx context.Context, id int64) (*domain.DiaristaEntry, error) {
	entries, err := r.queryDiaristas(ctx, `
		SELECT id, date, name, rg, unit, authorized_by, entry_time, exit_time, gatekeeper, photo, sync_status, created_at, updated_at
		FROM diarista_entries
		WHERE id = ?
	`, id)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, domain.ErrNotFound
	}
	return &entries[0], nil
}

func (r *SQLiteDiaristaRepository) queryDiaristas(ctx context.Context, query string, args ...any) ([]domain.DiaristaEntry, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query diarista entries: %w", err)
	}
	defer rows.Close()

	entries := make([]domain.DiaristaEntry, 0)
	for rows.Next() {
		var entry domain.DiaristaEntry
		var id int64
		if err := rows.Scan(
			&id,
			&entry.Date,
			&entry.Name,
			&entry.RG,
			&entry.Unit,
			&entry.AuthorizedBy,
			&entry.EntryTime,
			&entry.ExitTime,
			&entry.Gatekeeper,
			&entry.Photo,
			&entry.SyncStatus,
			&entry.CreatedAt,
			&entry.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan diarista entry: %w", err)
		}
		entry.ID = strconv.FormatInt(id, 10)
		entry.SyncStatus = diaristaClientSyncStatus(entry.SyncStatus)
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate diarista entries: %w", err)
	}
	return entries, nil
}

func parseDiaristaID(id string) (int64, error) {
	cleanID := strings.TrimPrefix(strings.TrimSpace(id), "d-")
	return strconv.ParseInt(cleanID, 10, 64)
}

func diaristaClientSyncStatus(status domain.SyncStatus) domain.SyncStatus {
	switch status {
	case domain.SyncStatusSynced:
		return domain.SyncStatus("synced")
	case domain.SyncStatusError:
		return domain.SyncStatus("failed")
	default:
		return domain.SyncStatus("pending")
	}
}
