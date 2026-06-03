package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"chateauneuf-portaria-backend/internal/domain"
	"chateauneuf-portaria-backend/internal/usecase"
)

type SQLiteAccessLogRepository struct {
	db *sql.DB
}

func NewSQLiteAccessLogRepository(db *sql.DB) *SQLiteAccessLogRepository {
	return &SQLiteAccessLogRepository{db: db}
}

func (r *SQLiteAccessLogRepository) Create(ctx context.Context, accessLog *domain.AccessLog) error {
	now := time.Now()
	accessLog.CreatedAt = now
	accessLog.UpdatedAt = now

	result, err := r.db.ExecContext(ctx, `
		INSERT INTO access_logs (
			external_id, visitor_name, document, company, phone, unit, resident_name,
			service_type, vehicle_plate, authorized_by, doorman, photo, entry_at, exit_at,
			visit_status, sync_status, sync_error, created_at, updated_at, synced_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		accessLog.ExternalID, accessLog.VisitorName, accessLog.Document, accessLog.Company,
		accessLog.Phone, accessLog.Unit, accessLog.ResidentName, accessLog.ServiceType,
		accessLog.VehiclePlate, accessLog.AuthorizedBy, accessLog.Doorman, accessLog.Photo, accessLog.EntryAt,
		accessLog.ExitAt, accessLog.VisitStatus, accessLog.SyncStatus, accessLog.SyncError,
		accessLog.CreatedAt, accessLog.UpdatedAt, accessLog.SyncedAt,
	)
	if err != nil {
		return fmt.Errorf("create access log: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("read access log id: %w", err)
	}
	accessLog.ID = id
	return nil
}

func (r *SQLiteAccessLogRepository) UpsertImported(ctx context.Context, accessLog *domain.AccessLog) error {
	if accessLog.CreatedAt.IsZero() {
		accessLog.CreatedAt = accessLog.EntryAt
	}
	if accessLog.UpdatedAt.IsZero() {
		accessLog.UpdatedAt = accessLog.CreatedAt
	}
	if accessLog.SyncStatus == "" {
		accessLog.SyncStatus = domain.SyncStatusSynced
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO access_logs (
			id, external_id, visitor_name, document, company, phone, unit, resident_name,
			service_type, vehicle_plate, authorized_by, doorman, photo, entry_at, exit_at,
			visit_status, sync_status, sync_error, created_at, updated_at, synced_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			external_id = excluded.external_id,
			visitor_name = excluded.visitor_name,
			document = excluded.document,
			company = excluded.company,
			phone = excluded.phone,
			unit = excluded.unit,
			resident_name = excluded.resident_name,
			service_type = excluded.service_type,
			vehicle_plate = excluded.vehicle_plate,
			authorized_by = excluded.authorized_by,
			doorman = excluded.doorman,
			photo = excluded.photo,
			entry_at = excluded.entry_at,
			exit_at = excluded.exit_at,
			visit_status = excluded.visit_status,
			sync_status = excluded.sync_status,
			sync_error = excluded.sync_error,
			created_at = excluded.created_at,
			updated_at = excluded.updated_at,
			synced_at = excluded.synced_at
		WHERE access_logs.sync_status != ?
	`,
		accessLog.ID, accessLog.ExternalID, accessLog.VisitorName, accessLog.Document,
		accessLog.Company, accessLog.Phone, accessLog.Unit, accessLog.ResidentName,
		accessLog.ServiceType, accessLog.VehiclePlate, accessLog.AuthorizedBy,
		accessLog.Doorman, accessLog.Photo, accessLog.EntryAt, accessLog.ExitAt,
		accessLog.VisitStatus, accessLog.SyncStatus, accessLog.SyncError,
		accessLog.CreatedAt, accessLog.UpdatedAt, accessLog.SyncedAt,
		domain.SyncStatusPending,
	)
	if err != nil {
		return fmt.Errorf("upsert imported access log: %w", err)
	}

	return nil
}

func (r *SQLiteAccessLogRepository) List(ctx context.Context, filters domain.AccessLogFilters) ([]domain.AccessLog, error) {
	query := `
		SELECT id, external_id, visitor_name, document, company, phone, unit, resident_name,
			service_type, vehicle_plate, authorized_by, doorman, photo, entry_at, exit_at,
			visit_status, sync_status, sync_error, created_at, updated_at, synced_at
		FROM access_logs
		WHERE 1 = 1
	`

	var args []any
	if filters.Date != "" {
		query += " AND date(entry_at) = date(?)"
		args = append(args, filters.Date)
	}
	if filters.Unit != "" {
		query += " AND unit = ?"
		args = append(args, filters.Unit)
	}
	if filters.Status != "" {
		query += " AND visit_status = ?"
		args = append(args, filters.Status)
	}
	if filters.VisitorName != "" {
		query += " AND visitor_name LIKE ?"
		args = append(args, "%"+filters.VisitorName+"%")
	}
	if filters.Document != "" {
		query += " AND document = ?"
		args = append(args, filters.Document)
	}

	query += " ORDER BY entry_at DESC"
	return r.queryAccessLogs(ctx, query, args...)
}

func (r *SQLiteAccessLogRepository) ListOpen(ctx context.Context) ([]domain.AccessLog, error) {
	return r.queryAccessLogs(ctx, `
		SELECT id, external_id, visitor_name, document, company, phone, unit, resident_name,
			service_type, vehicle_plate, authorized_by, doorman, photo, entry_at, exit_at,
			visit_status, sync_status, sync_error, created_at, updated_at, synced_at
		FROM access_logs
		WHERE exit_at IS NULL AND visit_status = ?
		ORDER BY entry_at DESC
	`, domain.VisitStatusInProgress)
}

func (r *SQLiteAccessLogRepository) FindByID(ctx context.Context, id int64) (*domain.AccessLog, error) {
	rows, err := r.queryAccessLogs(ctx, `
		SELECT id, external_id, visitor_name, document, company, phone, unit, resident_name,
			service_type, vehicle_plate, authorized_by, doorman, photo, entry_at, exit_at,
			visit_status, sync_status, sync_error, created_at, updated_at, synced_at
		FROM access_logs
		WHERE id = ?
	`, id)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, domain.ErrNotFound
	}
	return &rows[0], nil
}

func (r *SQLiteAccessLogRepository) Checkout(ctx context.Context, id int64, exitAt time.Time) (*domain.AccessLog, error) {
	result, err := r.db.ExecContext(ctx, `
		UPDATE access_logs
		SET exit_at = ?, visit_status = ?, sync_status = ?, sync_error = '', updated_at = ?
		WHERE id = ? AND exit_at IS NULL
	`, exitAt, domain.VisitStatusFinished, domain.SyncStatusPending, time.Now(), id)
	if err != nil {
		return nil, fmt.Errorf("checkout access log: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("read checkout rows affected: %w", err)
	}
	if affected == 0 {
		return nil, domain.ErrNotFound
	}

	return r.FindByID(ctx, id)
}

func (r *SQLiteAccessLogRepository) ListPendingSync(ctx context.Context, limit int) ([]domain.AccessLog, error) {
	if limit <= 0 {
		limit = 50
	}

	return r.queryAccessLogs(ctx, `
		SELECT id, external_id, visitor_name, document, company, phone, unit, resident_name,
			service_type, vehicle_plate, authorized_by, doorman, photo, entry_at, exit_at,
			visit_status, sync_status, sync_error, created_at, updated_at, synced_at
		FROM access_logs
		WHERE sync_status IN (?, ?)
		ORDER BY updated_at ASC
		LIMIT ?
	`, domain.SyncStatusPending, domain.SyncStatusError, limit)
}

func (r *SQLiteAccessLogRepository) MarkSynced(ctx context.Context, id int64, syncedAt time.Time) error {
	now := time.Now().Round(0)
	syncedAt = syncedAt.Round(0)
	_, err := r.db.ExecContext(ctx, `
		UPDATE access_logs
		SET sync_status = ?, sync_error = '', synced_at = ?, updated_at = ?
		WHERE id = ?
	`, domain.SyncStatusSynced, syncedAt, now, id)
	if err != nil {
		return fmt.Errorf("mark access log synced: %w", err)
	}
	return nil
}

func (r *SQLiteAccessLogRepository) MarkSyncError(ctx context.Context, id int64, syncError string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE access_logs
		SET sync_status = ?, sync_error = ?, updated_at = ?
		WHERE id = ?
	`, domain.SyncStatusError, strings.TrimSpace(syncError), time.Now(), id)
	if err != nil {
		return fmt.Errorf("mark access log sync error: %w", err)
	}
	return nil
}

func (r *SQLiteAccessLogRepository) SyncStats(ctx context.Context) (usecase.SyncStats, error) {
	var stats usecase.SyncStats
	var lastSyncedAt any
	err := r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(CASE WHEN sync_status IN ('PENDENTE_SYNC', 'ERRO_SYNC') THEN 1 END),
			MAX(synced_at),
			COALESCE((SELECT sync_error FROM access_logs WHERE sync_error != '' ORDER BY updated_at DESC LIMIT 1), '')
		FROM access_logs
	`).Scan(&stats.PendingCount, &lastSyncedAt, &stats.LastError)
	if err != nil {
		return usecase.SyncStats{}, fmt.Errorf("read sync stats: %w", err)
	}
	if lastSyncedAt != nil {
		parsed, err := parseSQLiteTime(lastSyncedAt)
		if err != nil {
			return usecase.SyncStats{}, fmt.Errorf("parse last synced at: %w", err)
		}
		if !parsed.IsZero() {
			stats.LastSyncedAt = &parsed
		}
	}
	return stats, nil
}

func parseSQLiteTime(value any) (time.Time, error) {
	switch typed := value.(type) {
	case time.Time:
		return typed, nil
	case string:
		return parseSQLiteTimeString(typed)
	case []byte:
		return parseSQLiteTimeString(string(typed))
	default:
		return time.Time{}, fmt.Errorf("unsupported value type %T", value)
	}
}

func parseSQLiteTimeString(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, nil
	}
	if base, _, found := strings.Cut(value, " m="); found {
		value = base
	}
	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05.999999999 -0700 MST",
		"2006-01-02 15:04:05 -0700 MST",
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999Z07:00",
		"2006-01-02 15:04:05-07:00",
		"2006-01-02 15:04:05Z07:00",
	}
	var lastErr error
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed, nil
		}
		lastErr = err
	}
	return time.Time{}, lastErr
}

func (r *SQLiteAccessLogRepository) queryAccessLogs(ctx context.Context, query string, args ...any) ([]domain.AccessLog, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query access logs: %w", err)
	}
	defer rows.Close()

	accessLogs := make([]domain.AccessLog, 0)
	for rows.Next() {
		var accessLog domain.AccessLog
		if err := rows.Scan(
			&accessLog.ID,
			&accessLog.ExternalID,
			&accessLog.VisitorName,
			&accessLog.Document,
			&accessLog.Company,
			&accessLog.Phone,
			&accessLog.Unit,
			&accessLog.ResidentName,
			&accessLog.ServiceType,
			&accessLog.VehiclePlate,
			&accessLog.AuthorizedBy,
			&accessLog.Doorman,
			&accessLog.Photo,
			&accessLog.EntryAt,
			&accessLog.ExitAt,
			&accessLog.VisitStatus,
			&accessLog.SyncStatus,
			&accessLog.SyncError,
			&accessLog.CreatedAt,
			&accessLog.UpdatedAt,
			&accessLog.SyncedAt,
		); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, domain.ErrNotFound
			}
			return nil, fmt.Errorf("scan access log: %w", err)
		}
		accessLogs = append(accessLogs, accessLog)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate access logs: %w", err)
	}

	return accessLogs, nil
}
