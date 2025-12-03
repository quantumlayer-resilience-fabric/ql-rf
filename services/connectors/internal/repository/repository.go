// Package repository provides database access for the connectors service.
package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBTX represents a database connection or transaction.
type DBTX interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Repository handles database operations for connectors.
type Repository struct {
	pool *pgxpool.Pool
	db   DBTX // Can be pool or transaction
}

// New creates a new repository.
func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool, db: pool}
}

// WithTx returns a new repository that uses the given transaction.
func (r *Repository) WithTx(tx pgx.Tx) *Repository {
	return &Repository{pool: r.pool, db: tx}
}

// BeginTx starts a new transaction and returns a repository using it.
func (r *Repository) BeginTx(ctx context.Context) (pgx.Tx, *Repository, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	return tx, r.WithTx(tx), nil
}

// Asset represents an asset in the database.
type Asset struct {
	ID           uuid.UUID
	OrgID        uuid.UUID
	EnvID        *uuid.UUID
	Platform     string
	Account      *string
	Region       *string
	Site         *string
	InstanceID   string
	Name         *string
	ImageRef     *string
	ImageVersion *string
	State        string
	Tags         json.RawMessage
	DiscoveredAt time.Time
	UpdatedAt    time.Time
}

// UpsertAssetParams contains parameters for upserting an asset.
type UpsertAssetParams struct {
	OrgID        uuid.UUID
	EnvID        *uuid.UUID
	Platform     string
	Account      *string
	Region       *string
	InstanceID   string
	Name         *string
	ImageRef     *string
	ImageVersion *string
	State        string
	Tags         json.RawMessage
}

// UpsertAsset creates or updates an asset and returns whether it was created.
func (r *Repository) UpsertAsset(ctx context.Context, params UpsertAssetParams) (*Asset, bool, error) {
	var a Asset
	var isNew bool

	// First check if asset exists
	existingID, err := r.getAssetID(ctx, params.OrgID, params.Platform, params.InstanceID)
	if err != nil && err != pgx.ErrNoRows {
		return nil, false, err
	}
	isNew = existingID == uuid.Nil

	err = r.db.QueryRow(ctx, `
		INSERT INTO assets (org_id, env_id, platform, account, region, instance_id, name, image_ref, image_version, state, tags)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (org_id, platform, instance_id)
		DO UPDATE SET
			env_id = EXCLUDED.env_id,
			account = EXCLUDED.account,
			region = EXCLUDED.region,
			name = EXCLUDED.name,
			image_ref = EXCLUDED.image_ref,
			image_version = EXCLUDED.image_version,
			state = EXCLUDED.state,
			tags = EXCLUDED.tags,
			updated_at = NOW()
		RETURNING id, org_id, env_id, platform, account, region, site,
		          instance_id, name, image_ref, image_version, state, tags,
		          discovered_at, updated_at
	`, params.OrgID, params.EnvID, params.Platform, params.Account, params.Region,
		params.InstanceID, params.Name, params.ImageRef, params.ImageVersion, params.State, params.Tags,
	).Scan(
		&a.ID, &a.OrgID, &a.EnvID, &a.Platform, &a.Account, &a.Region, &a.Site,
		&a.InstanceID, &a.Name, &a.ImageRef, &a.ImageVersion, &a.State, &a.Tags,
		&a.DiscoveredAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, false, err
	}

	return &a, isNew, nil
}

// getAssetID returns the ID of an existing asset.
func (r *Repository) getAssetID(ctx context.Context, orgID uuid.UUID, platform, instanceID string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRow(ctx, `
		SELECT id FROM assets WHERE org_id = $1 AND platform = $2 AND instance_id = $3
	`, orgID, platform, instanceID).Scan(&id)
	return id, err
}

// GetAsset retrieves an asset by ID.
func (r *Repository) GetAsset(ctx context.Context, id uuid.UUID) (*Asset, error) {
	var a Asset
	err := r.db.QueryRow(ctx, `
		SELECT id, org_id, env_id, platform, account, region, site,
		       instance_id, name, image_ref, image_version, state, tags,
		       discovered_at, updated_at
		FROM assets WHERE id = $1
	`, id).Scan(
		&a.ID, &a.OrgID, &a.EnvID, &a.Platform, &a.Account, &a.Region, &a.Site,
		&a.InstanceID, &a.Name, &a.ImageRef, &a.ImageVersion, &a.State, &a.Tags,
		&a.DiscoveredAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// GetAssetByInstanceID retrieves an asset by instance ID.
func (r *Repository) GetAssetByInstanceID(ctx context.Context, orgID uuid.UUID, platform, instanceID string) (*Asset, error) {
	var a Asset
	err := r.db.QueryRow(ctx, `
		SELECT id, org_id, env_id, platform, account, region, site,
		       instance_id, name, image_ref, image_version, state, tags,
		       discovered_at, updated_at
		FROM assets WHERE org_id = $1 AND platform = $2 AND instance_id = $3
	`, orgID, platform, instanceID).Scan(
		&a.ID, &a.OrgID, &a.EnvID, &a.Platform, &a.Account, &a.Region, &a.Site,
		&a.InstanceID, &a.Name, &a.ImageRef, &a.ImageVersion, &a.State, &a.Tags,
		&a.DiscoveredAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// ListAssetsByPlatform returns all assets for an org and platform.
func (r *Repository) ListAssetsByPlatform(ctx context.Context, orgID uuid.UUID, platform string) ([]Asset, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, org_id, env_id, platform, account, region, site,
		       instance_id, name, image_ref, image_version, state, tags,
		       discovered_at, updated_at
		FROM assets WHERE org_id = $1 AND platform = $2
	`, orgID, platform)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []Asset
	for rows.Next() {
		var a Asset
		err := rows.Scan(
			&a.ID, &a.OrgID, &a.EnvID, &a.Platform, &a.Account, &a.Region, &a.Site,
			&a.InstanceID, &a.Name, &a.ImageRef, &a.ImageVersion, &a.State, &a.Tags,
			&a.DiscoveredAt, &a.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		assets = append(assets, a)
	}

	return assets, nil
}

// MarkAssetTerminated updates an asset's state to terminated.
func (r *Repository) MarkAssetTerminated(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE assets SET state = 'terminated', updated_at = NOW() WHERE id = $1
	`, id)
	return err
}

// DeleteAsset removes an asset from the database.
func (r *Repository) DeleteAsset(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM assets WHERE id = $1`, id)
	return err
}
