// Package vulnmatch turns persisted CVE data into per-organization alerts.
//
// It bridges the CVE aggregator (which populates cve_cache + cve_package_matches)
// and the blast-radius engine: it discovers which (CVE, org) pairs have package
// matches against the SBOM inventory, upserts a cve_alert for each, and computes
// and stores the blast radius (affected packages/images/assets + counts + urgency).
package vulnmatch

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/blastradius"
)

// Service matches CVEs to organizations and maintains their alerts.
type Service struct {
	db     *pgxpool.Pool
	engine *blastradius.Engine
	logger *slog.Logger
}

// NewService creates a vulnerability matching service.
func NewService(db *pgxpool.Pool, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		db:     db,
		engine: blastradius.NewEngine(db, logger),
		logger: logger.With("component", "vulnmatch"),
	}
}

// candidate is a (CVE, org) pair that has at least one SBOM package match.
type candidate struct {
	CVEID      string
	CVECacheID uuid.UUID
	OrgID      uuid.UUID
	Severity   string
}

// Stats summarizes one scan pass.
type Stats struct {
	Candidates      int
	AlertsUpserted  int
	BlastRadiusRuns int
	Errors          int
}

// ScanAndAlert performs one full matching pass: find (CVE, org) pairs whose
// cve_package_matches match the org's SBOM packages, upsert a cve_alert for each,
// then compute and store the blast radius.
//
// This is the single RunOnce entrypoint for the matcher — usable directly from a
// CLI flag, a dedicated worker, or a Temporal activity. The background scheduler
// in cmd/orchestrator/main.go is only a thin loop around this method; no core
// logic lives in the goroutine.
func (s *Service) ScanAndAlert(ctx context.Context) (Stats, error) {
	var stats Stats

	candidates, err := s.findCandidates(ctx)
	if err != nil {
		return stats, fmt.Errorf("find candidates: %w", err)
	}
	stats.Candidates = len(candidates)

	for _, c := range candidates {
		alertID, err := s.upsertAlert(ctx, c)
		if err != nil {
			stats.Errors++
			s.logger.Error("failed to upsert cve alert",
				"cve_id", c.CVEID, "org_id", c.OrgID, "error", err)
			continue
		}
		stats.AlertsUpserted++

		cacheID := c.CVECacheID
		result, err := s.engine.Calculate(ctx, blastradius.CalculateInput{
			OrgID:      c.OrgID,
			CVEID:      c.CVEID,
			CVECacheID: &cacheID,
		})
		if err != nil {
			stats.Errors++
			s.logger.Error("failed to calculate blast radius",
				"cve_id", c.CVEID, "org_id", c.OrgID, "error", err)
			continue
		}

		if err := s.engine.StoreBlastRadius(ctx, alertID, result); err != nil {
			stats.Errors++
			s.logger.Error("failed to store blast radius",
				"alert_id", alertID, "cve_id", c.CVEID, "error", err)
			continue
		}
		stats.BlastRadiusRuns++
	}

	s.logger.Info("vulnerability scan complete",
		"candidates", stats.Candidates,
		"alerts_upserted", stats.AlertsUpserted,
		"blast_radius_runs", stats.BlastRadiusRuns,
		"errors", stats.Errors,
	)
	return stats, nil
}

// findCandidates returns the distinct (CVE, org) pairs that have at least one
// SBOM package matching a cve_package_matches pattern. The match join mirrors
// blastradius.Engine.findAffectedPackages but is grouped by org so we can decide
// which alerts to create.
//
// TODO(semver): the version comparisons use string ordering
// (e.g. sp.version < cpm.version_end), which is wrong for cases like
// '1.10' < '1.9'. This matches the engine's current behavior and is acceptable
// for this pass; replace with a semver-aware comparator (e.g. Masterminds/semver)
// in a follow-up. See vulnmatch tests documenting the limitation.
func (s *Service) findCandidates(ctx context.Context) ([]candidate, error) {
	const query = `
		SELECT DISTINCT cc.cve_id, cc.id, i.org_id, cc.severity
		FROM cve_package_matches cpm
		JOIN cve_cache cc ON cc.id = cpm.cve_cache_id
		JOIN sbom_packages sp ON (
			LOWER(sp.name) = LOWER(cpm.package_name)
			AND (cpm.package_type IS NULL OR LOWER(sp.type) = LOWER(cpm.package_type))
			AND (
				cpm.version_constraint = 'all'
				OR (cpm.version_constraint = 'exact' AND sp.version = cpm.version_start)
				OR (cpm.version_constraint = 'less_than' AND sp.version < cpm.version_end)
				OR (cpm.version_constraint = 'less_than_eq' AND sp.version <= cpm.version_end)
				OR (cpm.version_constraint = 'range' AND sp.version >= cpm.version_start AND sp.version < cpm.version_end)
			)
		)
		JOIN sboms s ON s.id = sp.sbom_id
		JOIN images i ON i.id = s.image_id
	`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query candidates: %w", err)
	}
	defer rows.Close()

	var candidates []candidate
	for rows.Next() {
		var c candidate
		if err := rows.Scan(&c.CVEID, &c.CVECacheID, &c.OrgID, &c.Severity); err != nil {
			return nil, fmt.Errorf("scan candidate: %w", err)
		}
		candidates = append(candidates, c)
	}
	return candidates, rows.Err()
}

// upsertAlert creates or refreshes the cve_alert for a (CVE, org) pair. Idempotent
// via UNIQUE(org_id, cve_id) (migrations/000017). Blast-radius counts and urgency
// are filled in afterward by the engine's StoreBlastRadius.
func (s *Service) upsertAlert(ctx context.Context, c candidate) (uuid.UUID, error) {
	severity := c.Severity
	if severity == "" {
		severity = "unknown"
	}

	const query = `
		INSERT INTO cve_alerts (org_id, cve_cache_id, cve_id, severity, status)
		VALUES ($1, $2, $3, $4, 'new')
		ON CONFLICT (org_id, cve_id) DO UPDATE SET
			cve_cache_id = EXCLUDED.cve_cache_id,
			severity = EXCLUDED.severity,
			last_seen_at = NOW(),
			updated_at = NOW()
		RETURNING id
	`

	var id uuid.UUID
	if err := s.db.QueryRow(ctx, query, c.OrgID, c.CVECacheID, c.CVEID, severity).Scan(&id); err != nil {
		return uuid.Nil, fmt.Errorf("upsert alert: %w", err)
	}
	return id, nil
}
