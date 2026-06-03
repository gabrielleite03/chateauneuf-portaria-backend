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

type SQLiteScheduledServiceRepository struct {
	db *sql.DB
}

func NewSQLiteScheduledServiceRepository(db *sql.DB) *SQLiteScheduledServiceRepository {
	return &SQLiteScheduledServiceRepository{db: db}
}

func (r *SQLiteScheduledServiceRepository) List(ctx context.Context) ([]domain.ScheduledService, error) {
	return r.queryScheduledServices(ctx, `
		SELECT id, date, name, document, company, unit, authorized_by, arrival_time, notes, status, photo, sync_status, created_at, updated_at
		FROM scheduled_services
		ORDER BY date DESC, id DESC
	`)
}

func (r *SQLiteScheduledServiceRepository) Create(ctx context.Context, service domain.ScheduledService) (*domain.ScheduledService, error) {
	now := time.Now()
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO scheduled_services (
			date, name, document, company, unit, authorized_by, arrival_time, notes, status, photo,
			sync_status, sync_error, created_at, updated_at, synced_at
		) VALUES (?, ?, ?, ?, ?, ?, '', ?, ?, '', ?, '', ?, ?, ?)
	`, service.Date, service.Name, service.Document, service.Company, service.Unit, service.AuthorizedBy,
		service.Notes, service.Status, service.SyncStatus, now, now, nil)
	if err != nil {
		return nil, fmt.Errorf("create scheduled service: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("read scheduled service id: %w", err)
	}
	return r.FindByID(ctx, id)
}

func (r *SQLiteScheduledServiceRepository) UpdateStatus(ctx context.Context, id string, status domain.ScheduledServiceStatus, photo string, arrivalTime string) (*domain.ScheduledService, error) {
	numericID, err := parseScheduledServiceID(id)
	if err != nil {
		return nil, domain.ErrInvalidInput
	}

	result, err := r.db.ExecContext(ctx, `
		UPDATE scheduled_services
		SET status = ?, photo = CASE WHEN ? != '' THEN ? ELSE photo END, arrival_time = ?, sync_status = ?, sync_error = '', updated_at = ?
		WHERE id = ?
	`, status, photo, photo, arrivalTime, domain.SyncStatusPending, time.Now(), numericID)
	if err != nil {
		return nil, fmt.Errorf("update scheduled service status: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("read scheduled service update rows affected: %w", err)
	}
	if affected == 0 {
		return nil, domain.ErrNotFound
	}
	return r.FindByID(ctx, numericID)
}

func (r *SQLiteScheduledServiceRepository) ListPendingSync(ctx context.Context, limit int) ([]domain.ScheduledService, error) {
	if limit <= 0 {
		limit = 50
	}
	return r.queryScheduledServices(ctx, `
		SELECT id, date, name, document, company, unit, authorized_by, arrival_time, notes, status, photo, sync_status, created_at, updated_at
		FROM scheduled_services
		WHERE sync_status IN (?, ?)
		ORDER BY updated_at ASC
		LIMIT ?
	`, domain.SyncStatusPending, domain.SyncStatusError, limit)
}

func (r *SQLiteScheduledServiceRepository) MarkSynced(ctx context.Context, id string, syncedAt time.Time) error {
	numericID, err := parseScheduledServiceID(id)
	if err != nil {
		return domain.ErrInvalidInput
	}

	_, err = r.db.ExecContext(ctx, `
		UPDATE scheduled_services
		SET sync_status = ?, sync_error = '', synced_at = ?, updated_at = ?
		WHERE id = ?
	`, domain.SyncStatusSynced, syncedAt.Round(0), time.Now().Round(0), numericID)
	if err != nil {
		return fmt.Errorf("mark scheduled service synced: %w", err)
	}
	return nil
}

func (r *SQLiteScheduledServiceRepository) MarkSyncError(ctx context.Context, id string, syncError string) error {
	numericID, err := parseScheduledServiceID(id)
	if err != nil {
		return domain.ErrInvalidInput
	}

	_, err = r.db.ExecContext(ctx, `
		UPDATE scheduled_services
		SET sync_status = ?, sync_error = ?, updated_at = ?
		WHERE id = ?
	`, domain.SyncStatusError, strings.TrimSpace(syncError), time.Now(), numericID)
	if err != nil {
		return fmt.Errorf("mark scheduled service sync error: %w", err)
	}
	return nil
}

func (r *SQLiteScheduledServiceRepository) SyncStats(ctx context.Context) (int, error) {
	var pendingCount int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(CASE WHEN sync_status IN ('PENDENTE_SYNC', 'ERRO_SYNC') THEN 1 END)
		FROM scheduled_services
	`).Scan(&pendingCount)
	if err != nil {
		return 0, fmt.Errorf("read scheduled services sync stats: %w", err)
	}
	return pendingCount, nil
}

func (r *SQLiteScheduledServiceRepository) Delete(ctx context.Context, id string) error {
	numericID, err := parseScheduledServiceID(id)
	if err != nil {
		return domain.ErrInvalidInput
	}

	result, err := r.db.ExecContext(ctx, `DELETE FROM scheduled_services WHERE id = ?`, numericID)
	if err != nil {
		return fmt.Errorf("delete scheduled service: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read scheduled service delete rows affected: %w", err)
	}
	if affected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *SQLiteScheduledServiceRepository) FindByID(ctx context.Context, id int64) (*domain.ScheduledService, error) {
	services, err := r.queryScheduledServices(ctx, `
		SELECT id, date, name, document, company, unit, authorized_by, arrival_time, notes, status, photo, sync_status, created_at, updated_at
		FROM scheduled_services
		WHERE id = ?
	`, id)
	if err != nil {
		return nil, err
	}
	if len(services) == 0 {
		return nil, domain.ErrNotFound
	}
	return &services[0], nil
}

func (r *SQLiteScheduledServiceRepository) queryScheduledServices(ctx context.Context, query string, args ...any) ([]domain.ScheduledService, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query scheduled services: %w", err)
	}
	defer rows.Close()

	services := make([]domain.ScheduledService, 0)
	for rows.Next() {
		var service domain.ScheduledService
		var id int64
		if err := rows.Scan(
			&id,
			&service.Date,
			&service.Name,
			&service.Document,
			&service.Company,
			&service.Unit,
			&service.AuthorizedBy,
			&service.ArrivalTime,
			&service.Notes,
			&service.Status,
			&service.Photo,
			&service.SyncStatus,
			&service.CreatedAt,
			&service.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan scheduled service: %w", err)
		}
		service.ID = strconv.FormatInt(id, 10)
		service.SyncStatus = scheduledClientSyncStatus(service.SyncStatus)
		services = append(services, service)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate scheduled services: %w", err)
	}
	return services, nil
}

func parseScheduledServiceID(id string) (int64, error) {
	cleanID := strings.TrimPrefix(strings.TrimSpace(id), "s-")
	return strconv.ParseInt(cleanID, 10, 64)
}

func scheduledClientSyncStatus(status domain.SyncStatus) domain.SyncStatus {
	switch status {
	case domain.SyncStatusSynced:
		return domain.SyncStatus("synced")
	case domain.SyncStatusError:
		return domain.SyncStatus("failed")
	default:
		return domain.SyncStatus("pending")
	}
}
