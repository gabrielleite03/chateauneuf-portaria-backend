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

type SQLiteShoppingRepository struct {
	db *sql.DB
}

func NewSQLiteShoppingRepository(db *sql.DB) *SQLiteShoppingRepository {
	return &SQLiteShoppingRepository{db: db}
}

func (r *SQLiteShoppingRepository) List(ctx context.Context) ([]domain.ShoppingDelivery, error) {
	return r.queryShopping(ctx, `
		SELECT id, unit, courier_name, document, store, product, notes, photo, received_at, withdrawn_at,
			status, sync_status, created_at, updated_at
		FROM shopping_deliveries
		ORDER BY received_at DESC, id DESC
	`)
}

func (r *SQLiteShoppingRepository) Create(ctx context.Context, delivery domain.ShoppingDelivery) (*domain.ShoppingDelivery, error) {
	now := time.Now().Round(0)
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO shopping_deliveries (
			unit, courier_name, document, store, product, notes, photo, received_at, withdrawn_at,
			status, sync_status, sync_error, created_at, updated_at, synced_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, NULL, ?, ?, '', ?, ?, ?)
	`, delivery.Unit, delivery.CourierName, delivery.Document, delivery.Store, delivery.Product, delivery.Notes,
		delivery.Photo, now, delivery.Status, delivery.SyncStatus, now, now, nil)
	if err != nil {
		return nil, fmt.Errorf("create shopping delivery: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("read shopping delivery id: %w", err)
	}
	return r.FindByID(ctx, id)
}

func (r *SQLiteShoppingRepository) Withdraw(ctx context.Context, id string) (*domain.ShoppingDelivery, error) {
	numericID, err := parseShoppingID(id)
	if err != nil {
		return nil, domain.ErrInvalidInput
	}

	now := time.Now().Round(0)
	result, err := r.db.ExecContext(ctx, `
		UPDATE shopping_deliveries
		SET withdrawn_at = ?, status = ?, sync_status = ?, sync_error = '', updated_at = ?
		WHERE id = ? AND status = ?
	`, now, domain.ShoppingStatusWithdrawn, domain.SyncStatusPending, now, numericID, domain.ShoppingStatusWaiting)
	if err != nil {
		return nil, fmt.Errorf("withdraw shopping delivery: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("read withdraw shopping rows affected: %w", err)
	}
	if affected == 0 {
		return nil, domain.ErrNotFound
	}

	return r.FindByID(ctx, numericID)
}

func (r *SQLiteShoppingRepository) ListPendingSync(ctx context.Context, limit int) ([]domain.ShoppingDelivery, error) {
	if limit <= 0 {
		limit = 50
	}
	return r.queryShopping(ctx, `
		SELECT id, unit, courier_name, document, store, product, notes, photo, received_at, withdrawn_at,
			status, sync_status, created_at, updated_at
		FROM shopping_deliveries
		WHERE sync_status IN (?, ?)
		ORDER BY updated_at ASC
		LIMIT ?
	`, domain.SyncStatusPending, domain.SyncStatusError, limit)
}

func (r *SQLiteShoppingRepository) MarkSynced(ctx context.Context, id string, syncedAt time.Time) error {
	numericID, err := parseShoppingID(id)
	if err != nil {
		return domain.ErrInvalidInput
	}

	_, err = r.db.ExecContext(ctx, `
		UPDATE shopping_deliveries
		SET sync_status = ?, sync_error = '', synced_at = ?, updated_at = ?
		WHERE id = ?
	`, domain.SyncStatusSynced, syncedAt.Round(0), time.Now().Round(0), numericID)
	if err != nil {
		return fmt.Errorf("mark shopping delivery synced: %w", err)
	}
	return nil
}

func (r *SQLiteShoppingRepository) MarkSyncError(ctx context.Context, id string, syncError string) error {
	numericID, err := parseShoppingID(id)
	if err != nil {
		return domain.ErrInvalidInput
	}

	_, err = r.db.ExecContext(ctx, `
		UPDATE shopping_deliveries
		SET sync_status = ?, sync_error = ?, updated_at = ?
		WHERE id = ?
	`, domain.SyncStatusError, strings.TrimSpace(syncError), time.Now().Round(0), numericID)
	if err != nil {
		return fmt.Errorf("mark shopping delivery sync error: %w", err)
	}
	return nil
}

func (r *SQLiteShoppingRepository) SyncStats(ctx context.Context) (int, error) {
	var pendingCount int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(CASE WHEN sync_status IN ('PENDENTE_SYNC', 'ERRO_SYNC') THEN 1 END)
		FROM shopping_deliveries
	`).Scan(&pendingCount)
	if err != nil {
		return 0, fmt.Errorf("read shopping sync stats: %w", err)
	}
	return pendingCount, nil
}

func (r *SQLiteShoppingRepository) FindByID(ctx context.Context, id int64) (*domain.ShoppingDelivery, error) {
	deliveries, err := r.queryShopping(ctx, `
		SELECT id, unit, courier_name, document, store, product, notes, photo, received_at, withdrawn_at,
			status, sync_status, created_at, updated_at
		FROM shopping_deliveries
		WHERE id = ?
	`, id)
	if err != nil {
		return nil, err
	}
	if len(deliveries) == 0 {
		return nil, domain.ErrNotFound
	}
	return &deliveries[0], nil
}

func (r *SQLiteShoppingRepository) queryShopping(ctx context.Context, query string, args ...any) ([]domain.ShoppingDelivery, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query shopping deliveries: %w", err)
	}
	defer rows.Close()

	deliveries := make([]domain.ShoppingDelivery, 0)
	for rows.Next() {
		var delivery domain.ShoppingDelivery
		var id int64
		var withdrawnAt sql.NullTime
		if err := rows.Scan(
			&id,
			&delivery.Unit,
			&delivery.CourierName,
			&delivery.Document,
			&delivery.Store,
			&delivery.Product,
			&delivery.Notes,
			&delivery.Photo,
			&delivery.ReceivedAt,
			&withdrawnAt,
			&delivery.Status,
			&delivery.SyncStatus,
			&delivery.CreatedAt,
			&delivery.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan shopping delivery: %w", err)
		}
		delivery.ID = strconv.FormatInt(id, 10)
		if withdrawnAt.Valid {
			delivery.WithdrawnAt = &withdrawnAt.Time
		}
		delivery.SyncStatus = shoppingClientSyncStatus(delivery.SyncStatus)
		deliveries = append(deliveries, delivery)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate shopping deliveries: %w", err)
	}

	return deliveries, nil
}

func parseShoppingID(id string) (int64, error) {
	cleanID := strings.TrimPrefix(strings.TrimSpace(id), "s-")
	return strconv.ParseInt(cleanID, 10, 64)
}

func shoppingClientSyncStatus(status domain.SyncStatus) domain.SyncStatus {
	switch status {
	case domain.SyncStatusSynced:
		return domain.SyncStatus("synced")
	case domain.SyncStatusError:
		return domain.SyncStatus("failed")
	default:
		return domain.SyncStatus("pending")
	}
}
