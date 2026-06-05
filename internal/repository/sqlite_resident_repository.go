package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"chateauneuf-portaria-backend/internal/domain"
)

type SQLiteResidentRepository struct {
	db *sql.DB
}

func NewSQLiteResidentRepository(db *sql.DB) *SQLiteResidentRepository {
	return &SQLiteResidentRepository{db: db}
}

func (r *SQLiteResidentRepository) List(ctx context.Context) ([]domain.Resident, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT unit, owner, phones, tenant, tenant_photo, family_members, photo, sync_status, updated_at
		FROM residents
		ORDER BY unit ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query residents: %w", err)
	}
	defer rows.Close()

	residents := make([]domain.Resident, 0)
	for rows.Next() {
		var resident domain.Resident
		if err := rows.Scan(
			&resident.Unit,
			&resident.Owner,
			&resident.Phones,
			&resident.Tenant,
			&resident.TenantPhoto,
			&resident.FamilyMembers,
			&resident.Photo,
			&resident.SyncStatus,
			&resident.LastUpdated,
		); err != nil {
			return nil, fmt.Errorf("scan resident: %w", err)
		}
		residents = append(residents, resident)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate residents: %w", err)
	}

	return residents, nil
}

func (r *SQLiteResidentRepository) Upsert(ctx context.Context, resident domain.Resident) (*domain.Resident, error) {
	now := time.Now()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO residents (
			unit, owner, phones, tenant, tenant_photo, family_members, photo, sync_status, sync_error, created_at, updated_at, synced_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, '', ?, ?, ?)
		ON CONFLICT(unit) DO UPDATE SET
			owner = excluded.owner,
			phones = excluded.phones,
			tenant = excluded.tenant,
			tenant_photo = excluded.tenant_photo,
			family_members = excluded.family_members,
			photo = excluded.photo,
			sync_status = excluded.sync_status,
			sync_error = '',
			updated_at = excluded.updated_at,
			synced_at = excluded.synced_at
	`, resident.Unit, resident.Owner, resident.Phones, resident.Tenant, resident.TenantPhoto, resident.FamilyMembers,
		resident.Photo, resident.SyncStatus, now, now, now)
	if err != nil {
		return nil, fmt.Errorf("upsert resident: %w", err)
	}

	row := r.db.QueryRowContext(ctx, `
		SELECT unit, owner, phones, tenant, tenant_photo, family_members, photo, sync_status, updated_at
		FROM residents
		WHERE unit = ?
	`, resident.Unit)

	var updated domain.Resident
	if err := row.Scan(
		&updated.Unit,
		&updated.Owner,
		&updated.Phones,
		&updated.Tenant,
		&updated.TenantPhoto,
		&updated.FamilyMembers,
		&updated.Photo,
		&updated.SyncStatus,
		&updated.LastUpdated,
	); err != nil {
		return nil, fmt.Errorf("read resident after upsert: %w", err)
	}

	return &updated, nil
}
