package repository

import (
	"chateauneuf-portaria-backend/internal/domain"
	"chateauneuf-portaria-backend/internal/usecase"
	"context"
	"fmt"
	"strings"
	"time"
)

func (r *SQLiteReservationRepository) ListPendingSync(ctx context.Context, limit int) ([]domain.CommonAreaReservation, error) {
	if limit <= 0 {
		limit = 50
	}
	return r.queryReservations(ctx, `SELECT id, area, resident_name, unit, reservation_date, start_time, end_time,
		guests, notes, status, sync_status, created_at, updated_at,
		EXISTS(SELECT 1 FROM reservation_signatures rs WHERE rs.reservation_id = common_area_reservations.id AND rs.consumed_at IS NOT NULL)
		FROM common_area_reservations WHERE sync_status IN (?, ?) ORDER BY updated_at, id LIMIT ?`,
		domain.SyncStatusPending, domain.SyncStatusError, limit)
}

// Only acknowledge the revision actually sent; a concurrent cancellation must remain pending.
func (r *SQLiteReservationRepository) MarkSynced(ctx context.Context, id string, updatedAt, syncedAt time.Time) error {
	_, err := r.db.ExecContext(ctx, `UPDATE common_area_reservations SET sync_status = ?, sync_error = '', synced_at = ?
		WHERE id = ? AND updated_at = ?`, domain.SyncStatusSynced, syncedAt.Round(0), id, updatedAt)
	return err
}

func (r *SQLiteReservationRepository) MarkSyncError(ctx context.Context, id string, updatedAt time.Time, syncError string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE common_area_reservations SET sync_status = ?, sync_error = ?
		WHERE id = ? AND updated_at = ?`, domain.SyncStatusError, strings.TrimSpace(syncError), id, updatedAt)
	return err
}

func (r *SQLiteReservationRepository) SyncStats(ctx context.Context) (usecase.SyncStats, error) {
	var stats usecase.SyncStats
	var lastSyncedAt any
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(CASE WHEN sync_status IN ('PENDENTE_SYNC', 'ERRO_SYNC') THEN 1 END),
		MAX(synced_at), COALESCE((SELECT sync_error FROM common_area_reservations WHERE sync_error != '' ORDER BY updated_at DESC LIMIT 1), '')
		FROM common_area_reservations`).Scan(&stats.PendingCount, &lastSyncedAt, &stats.LastError)
	if err != nil {
		return stats, fmt.Errorf("read reservation sync stats: %w", err)
	}
	if lastSyncedAt != nil {
		parsed, err := parseSQLiteTime(lastSyncedAt)
		if err != nil {
			return stats, err
		}
		stats.LastSyncedAt = &parsed
	}
	return stats, nil
}
