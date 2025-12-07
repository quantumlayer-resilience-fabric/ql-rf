package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/quantumlayerhq/ql-rf/services/api/internal/service"
)

// connectorRow represents a database row for a connector.
type connectorRow struct {
	ID             uuid.UUID       `json:"id"`
	OrgID          uuid.UUID       `json:"org_id"`
	Name           string          `json:"name"`
	Platform       string          `json:"platform"`
	Enabled        bool            `json:"enabled"`
	Config         json.RawMessage `json:"config"`
	LastSyncAt     *time.Time      `json:"last_sync_at,omitempty"`
	LastSyncStatus *string         `json:"last_sync_status,omitempty"`
	LastSyncError  *string         `json:"last_sync_error,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// ConnectorRepositoryAdapter adapts the database to implement service.ConnectorRepository.
type ConnectorRepositoryAdapter struct {
	pool *pgxpool.Pool
}

// NewConnectorRepository creates a new ConnectorRepositoryAdapter.
func NewConnectorRepository(pool *pgxpool.Pool) *ConnectorRepositoryAdapter {
	return &ConnectorRepositoryAdapter{pool: pool}
}

// Create creates a new connector.
func (r *ConnectorRepositoryAdapter) Create(ctx context.Context, params service.CreateConnectorRepoParams) (*service.ConnectorModel, error) {
	var c connectorRow
	// Use default schedule if not provided
	syncSchedule := params.SyncSchedule
	if syncSchedule == "" {
		syncSchedule = "1h"
	}
	err := r.pool.QueryRow(ctx, `
		INSERT INTO connectors (org_id, name, platform, enabled, config, sync_schedule, sync_enabled)
		VALUES ($1, $2, $3, $4, $5, $6, TRUE)
		RETURNING id, org_id, name, platform, enabled, config, last_sync_at,
		          last_sync_status, last_sync_error, created_at, updated_at
	`, params.OrgID, params.Name, params.Platform, params.Enabled, params.Config, syncSchedule).Scan(
		&c.ID, &c.OrgID, &c.Name, &c.Platform, &c.Enabled, &c.Config,
		&c.LastSyncAt, &c.LastSyncStatus, &c.LastSyncError, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return rowToModel(&c), nil
}

// Get retrieves a connector by ID with tenant isolation.
func (r *ConnectorRepositoryAdapter) Get(ctx context.Context, id, orgID uuid.UUID) (*service.ConnectorModel, error) {
	var c connectorRow
	err := r.pool.QueryRow(ctx, `
		SELECT id, org_id, name, platform, enabled, config, last_sync_at,
		       last_sync_status, last_sync_error, created_at, updated_at
		FROM connectors
		WHERE id = $1 AND org_id = $2
	`, id, orgID).Scan(
		&c.ID, &c.OrgID, &c.Name, &c.Platform, &c.Enabled, &c.Config,
		&c.LastSyncAt, &c.LastSyncStatus, &c.LastSyncError, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return rowToModel(&c), nil
}

// ListByOrg retrieves all connectors for an organization.
func (r *ConnectorRepositoryAdapter) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]service.ConnectorModel, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, org_id, name, platform, enabled, config, last_sync_at,
		       last_sync_status, last_sync_error, created_at, updated_at
		FROM connectors
		WHERE org_id = $1
		ORDER BY name
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var connectors []service.ConnectorModel
	for rows.Next() {
		var c connectorRow
		if err := rows.Scan(
			&c.ID, &c.OrgID, &c.Name, &c.Platform, &c.Enabled, &c.Config,
			&c.LastSyncAt, &c.LastSyncStatus, &c.LastSyncError, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		connectors = append(connectors, *rowToModel(&c))
	}
	return connectors, rows.Err()
}

// Delete deletes a connector by ID with tenant isolation.
func (r *ConnectorRepositoryAdapter) Delete(ctx context.Context, id, orgID uuid.UUID) error {
	result, err := r.pool.Exec(ctx, `
		DELETE FROM connectors WHERE id = $1 AND org_id = $2
	`, id, orgID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNoRows
	}
	return nil
}

// UpdateSyncStatus updates the sync status of a connector.
func (r *ConnectorRepositoryAdapter) UpdateSyncStatus(ctx context.Context, id, orgID uuid.UUID, status, syncError string) error {
	var errorPtr *string
	if syncError != "" {
		errorPtr = &syncError
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE connectors
		SET last_sync_at = NOW(),
		    last_sync_status = $3,
		    last_sync_error = $4,
		    updated_at = NOW()
		WHERE id = $1 AND org_id = $2
	`, id, orgID, status, errorPtr)
	return err
}

// UpdateEnabled enables or disables a connector.
func (r *ConnectorRepositoryAdapter) UpdateEnabled(ctx context.Context, id, orgID uuid.UUID, enabled bool) error {
	result, err := r.pool.Exec(ctx, `
		UPDATE connectors
		SET enabled = $3, updated_at = NOW()
		WHERE id = $1 AND org_id = $2
	`, id, orgID, enabled)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNoRows
	}
	return nil
}

// UpdateConfig updates the connector configuration.
func (r *ConnectorRepositoryAdapter) UpdateConfig(ctx context.Context, id, orgID uuid.UUID, config json.RawMessage) error {
	result, err := r.pool.Exec(ctx, `
		UPDATE connectors
		SET config = $3, updated_at = NOW()
		WHERE id = $1 AND org_id = $2
	`, id, orgID, config)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNoRows
	}
	return nil
}

// CountByOrg counts connectors for an organization.
func (r *ConnectorRepositoryAdapter) CountByOrg(ctx context.Context, orgID uuid.UUID) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM connectors WHERE org_id = $1`, orgID).Scan(&count)
	return count, err
}

// ExistsByName checks if a connector with the given name exists for the organization.
func (r *ConnectorRepositoryAdapter) ExistsByName(ctx context.Context, orgID uuid.UUID, name string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM connectors WHERE org_id = $1 AND name = $2)
	`, orgID, name).Scan(&exists)
	return exists, err
}

// rowToModel converts a connectorRow to a service.ConnectorModel.
func rowToModel(c *connectorRow) *service.ConnectorModel {
	return &service.ConnectorModel{
		ID:             c.ID,
		OrgID:          c.OrgID,
		Name:           c.Name,
		Platform:       c.Platform,
		Enabled:        c.Enabled,
		Config:         c.Config,
		LastSyncAt:     c.LastSyncAt,
		LastSyncStatus: c.LastSyncStatus,
		LastSyncError:  c.LastSyncError,
		CreatedAt:      c.CreatedAt,
		UpdatedAt:      c.UpdatedAt,
	}
}
