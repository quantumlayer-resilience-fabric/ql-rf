package sbom

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// Service provides SBOM management operations.
type Service struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewService creates a new SBOM service.
func NewService(db *sql.DB, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		db:     db,
		logger: logger.With("component", "sbom-service"),
	}
}

// Create stores a new SBOM in the database.
func (s *Service) Create(ctx context.Context, sbom *SBOM) error {
	if sbom.ID == uuid.Nil {
		sbom.ID = uuid.New()
	}
	if sbom.GeneratedAt.IsZero() {
		sbom.GeneratedAt = time.Now()
	}

	contentJSON, err := json.Marshal(sbom.Content)
	if err != nil {
		return fmt.Errorf("marshal sbom content: %w", err)
	}

	query := `
		INSERT INTO sboms (
			id, image_id, org_id, format, version, content,
			package_count, vuln_count, generated_at, scanner, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
	`

	_, err = s.db.ExecContext(ctx, query,
		sbom.ID,
		sbom.ImageID,
		sbom.OrgID,
		sbom.Format,
		sbom.Version,
		contentJSON,
		sbom.PackageCount,
		sbom.VulnCount,
		sbom.GeneratedAt,
		sbom.Scanner,
	)
	if err != nil {
		return fmt.Errorf("insert sbom: %w", err)
	}

	s.logger.Info("sbom created",
		"sbom_id", sbom.ID,
		"image_id", sbom.ImageID,
		"format", sbom.Format,
		"packages", sbom.PackageCount,
	)

	return nil
}

// Get retrieves an SBOM by ID.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*SBOM, error) {
	query := `
		SELECT
			id, image_id, org_id, format, version, content,
			package_count, vuln_count, generated_at, scanner, created_at, updated_at
		FROM sboms
		WHERE id = $1
	`

	var sbom SBOM
	var contentJSON []byte

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&sbom.ID,
		&sbom.ImageID,
		&sbom.OrgID,
		&sbom.Format,
		&sbom.Version,
		&contentJSON,
		&sbom.PackageCount,
		&sbom.VulnCount,
		&sbom.GeneratedAt,
		&sbom.Scanner,
		&sbom.CreatedAt,
		&sbom.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("sbom not found")
	}
	if err != nil {
		return nil, fmt.Errorf("query sbom: %w", err)
	}

	if err := json.Unmarshal(contentJSON, &sbom.Content); err != nil {
		return nil, fmt.Errorf("unmarshal sbom content: %w", err)
	}

	return &sbom, nil
}

// GetByImageID retrieves the most recent SBOM for an image.
func (s *Service) GetByImageID(ctx context.Context, imageID uuid.UUID) (*SBOM, error) {
	query := `
		SELECT
			id, image_id, org_id, format, version, content,
			package_count, vuln_count, generated_at, scanner, created_at, updated_at
		FROM sboms
		WHERE image_id = $1
		ORDER BY generated_at DESC
		LIMIT 1
	`

	var sbom SBOM
	var contentJSON []byte

	err := s.db.QueryRowContext(ctx, query, imageID).Scan(
		&sbom.ID,
		&sbom.ImageID,
		&sbom.OrgID,
		&sbom.Format,
		&sbom.Version,
		&contentJSON,
		&sbom.PackageCount,
		&sbom.VulnCount,
		&sbom.GeneratedAt,
		&sbom.Scanner,
		&sbom.CreatedAt,
		&sbom.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no sbom found for image")
	}
	if err != nil {
		return nil, fmt.Errorf("query sbom by image: %w", err)
	}

	if err := json.Unmarshal(contentJSON, &sbom.Content); err != nil {
		return nil, fmt.Errorf("unmarshal sbom content: %w", err)
	}

	return &sbom, nil
}

// List retrieves SBOMs for an organization with pagination.
func (s *Service) List(ctx context.Context, orgID uuid.UUID, page, pageSize int) (*SBOMListResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	// Query for SBOMs with vulnerability counts by severity
	query := `
		SELECT
			s.id, s.image_id, s.format, s.package_count, s.generated_at,
			COALESCE(COUNT(*) FILTER (WHERE sv.severity = 'critical'), 0) as critical_count,
			COALESCE(COUNT(*) FILTER (WHERE sv.severity = 'high'), 0) as high_count,
			COALESCE(COUNT(*) FILTER (WHERE sv.severity = 'medium'), 0) as medium_count,
			COALESCE(COUNT(*) FILTER (WHERE sv.severity = 'low'), 0) as low_count,
			COALESCE(COUNT(sv.id), 0) as total_vulns
		FROM sboms s
		LEFT JOIN sbom_vulnerabilities sv ON sv.sbom_id = s.id
		WHERE s.org_id = $1
		GROUP BY s.id, s.image_id, s.format, s.package_count, s.generated_at
		ORDER BY s.generated_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := s.db.QueryContext(ctx, query, orgID, pageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("query sboms: %w", err)
	}
	defer rows.Close()

	var sboms []SBOMSummary
	for rows.Next() {
		var summary SBOMSummary
		if err := rows.Scan(
			&summary.ID,
			&summary.ImageID,
			&summary.Format,
			&summary.PackageCount,
			&summary.GeneratedAt,
			&summary.Critical,
			&summary.High,
			&summary.Medium,
			&summary.Low,
			&summary.VulnCount,
		); err != nil {
			return nil, fmt.Errorf("scan sbom summary: %w", err)
		}
		sboms = append(sboms, summary)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sboms: %w", err)
	}

	// Count total SBOMs
	var total int
	countQuery := `SELECT COUNT(*) FROM sboms WHERE org_id = $1`
	if err := s.db.QueryRowContext(ctx, countQuery, orgID).Scan(&total); err != nil {
		return nil, fmt.Errorf("count sboms: %w", err)
	}

	totalPages := total / pageSize
	if total%pageSize > 0 {
		totalPages++
	}

	return &SBOMListResponse{
		SBOMs:      sboms,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// Delete removes an SBOM and its associated data.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM sboms WHERE id = $1`
	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete sbom: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("sbom not found")
	}

	s.logger.Info("sbom deleted", "sbom_id", id)
	return nil
}

// CreatePackage adds a package to an SBOM.
func (s *Service) CreatePackage(ctx context.Context, pkg *Package) error {
	if pkg.ID == uuid.Nil {
		pkg.ID = uuid.New()
	}

	query := `
		INSERT INTO sbom_packages (
			id, sbom_id, name, version, type, purl, cpe,
			license, supplier, checksum, source_url, location, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())
	`

	_, err := s.db.ExecContext(ctx, query,
		pkg.ID,
		pkg.SBOMID,
		pkg.Name,
		pkg.Version,
		pkg.Type,
		nullString(pkg.PURL),
		nullString(pkg.CPE),
		nullString(pkg.License),
		nullString(pkg.Supplier),
		nullString(pkg.Checksum),
		nullString(pkg.SourceURL),
		nullString(pkg.Location),
	)
	if err != nil {
		return fmt.Errorf("insert package: %w", err)
	}

	return nil
}

// CreatePackageBatch creates multiple packages in a single transaction.
func (s *Service) CreatePackageBatch(ctx context.Context, packages []Package) error {
	if len(packages) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO sbom_packages (
			id, sbom_id, name, version, type, purl, cpe,
			license, supplier, checksum, source_url, location, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())
	`)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	for i := range packages {
		if packages[i].ID == uuid.Nil {
			packages[i].ID = uuid.New()
		}

		_, err := stmt.ExecContext(ctx,
			packages[i].ID,
			packages[i].SBOMID,
			packages[i].Name,
			packages[i].Version,
			packages[i].Type,
			nullString(packages[i].PURL),
			nullString(packages[i].CPE),
			nullString(packages[i].License),
			nullString(packages[i].Supplier),
			nullString(packages[i].Checksum),
			nullString(packages[i].SourceURL),
			nullString(packages[i].Location),
		)
		if err != nil {
			return fmt.Errorf("insert package %s: %w", packages[i].Name, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	s.logger.Info("packages created", "count", len(packages))
	return nil
}

// GetPackages retrieves all packages for an SBOM.
func (s *Service) GetPackages(ctx context.Context, sbomID uuid.UUID) ([]Package, error) {
	query := `
		SELECT
			id, sbom_id, name, version, type,
			COALESCE(purl, '') as purl,
			COALESCE(cpe, '') as cpe,
			COALESCE(license, '') as license,
			COALESCE(supplier, '') as supplier,
			COALESCE(checksum, '') as checksum,
			COALESCE(source_url, '') as source_url,
			COALESCE(location, '') as location,
			created_at
		FROM sbom_packages
		WHERE sbom_id = $1
		ORDER BY name, version
	`

	rows, err := s.db.QueryContext(ctx, query, sbomID)
	if err != nil {
		return nil, fmt.Errorf("query packages: %w", err)
	}
	defer rows.Close()

	var packages []Package
	for rows.Next() {
		var pkg Package
		if err := rows.Scan(
			&pkg.ID,
			&pkg.SBOMID,
			&pkg.Name,
			&pkg.Version,
			&pkg.Type,
			&pkg.PURL,
			&pkg.CPE,
			&pkg.License,
			&pkg.Supplier,
			&pkg.Checksum,
			&pkg.SourceURL,
			&pkg.Location,
			&pkg.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan package: %w", err)
		}
		packages = append(packages, pkg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate packages: %w", err)
	}

	return packages, nil
}

// CreateVulnerability adds a vulnerability to an SBOM.
func (s *Service) CreateVulnerability(ctx context.Context, vuln *Vulnerability) error {
	if vuln.ID == uuid.Nil {
		vuln.ID = uuid.New()
	}

	refsJSON, err := json.Marshal(vuln.References)
	if err != nil {
		return fmt.Errorf("marshal references: %w", err)
	}

	query := `
		INSERT INTO sbom_vulnerabilities (
			id, sbom_id, package_id, cve_id, severity, cvss_score, cvss_vector,
			description, fixed_version, published_date, modified_date,
			references, data_source, exploit_available, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NOW(), NOW())
	`

	_, err = s.db.ExecContext(ctx, query,
		vuln.ID,
		vuln.SBOMID,
		vuln.PackageID,
		vuln.CVEID,
		vuln.Severity,
		vuln.CVSSScore,
		nullString(vuln.CVSSVector),
		nullString(vuln.Description),
		nullString(vuln.FixedVersion),
		vuln.PublishedDate,
		vuln.ModifiedDate,
		refsJSON,
		nullString(vuln.DataSource),
		vuln.ExploitAvailable,
	)
	if err != nil {
		return fmt.Errorf("insert vulnerability: %w", err)
	}

	return nil
}

// CreateVulnerabilityBatch creates multiple vulnerabilities in a transaction.
func (s *Service) CreateVulnerabilityBatch(ctx context.Context, vulns []Vulnerability) error {
	if len(vulns) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO sbom_vulnerabilities (
			id, sbom_id, package_id, cve_id, severity, cvss_score, cvss_vector,
			description, fixed_version, published_date, modified_date,
			references, data_source, exploit_available, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NOW(), NOW())
	`)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	for i := range vulns {
		if vulns[i].ID == uuid.Nil {
			vulns[i].ID = uuid.New()
		}

		refsJSON, err := json.Marshal(vulns[i].References)
		if err != nil {
			return fmt.Errorf("marshal references for %s: %w", vulns[i].CVEID, err)
		}

		_, err = stmt.ExecContext(ctx,
			vulns[i].ID,
			vulns[i].SBOMID,
			vulns[i].PackageID,
			vulns[i].CVEID,
			vulns[i].Severity,
			vulns[i].CVSSScore,
			nullString(vulns[i].CVSSVector),
			nullString(vulns[i].Description),
			nullString(vulns[i].FixedVersion),
			vulns[i].PublishedDate,
			vulns[i].ModifiedDate,
			refsJSON,
			nullString(vulns[i].DataSource),
			vulns[i].ExploitAvailable,
		)
		if err != nil {
			return fmt.Errorf("insert vulnerability %s: %w", vulns[i].CVEID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	s.logger.Info("vulnerabilities created", "count", len(vulns))
	return nil
}

// GetVulnerabilities retrieves all vulnerabilities for an SBOM.
func (s *Service) GetVulnerabilities(ctx context.Context, sbomID uuid.UUID, filter *VulnerabilityFilter) ([]Vulnerability, error) {
	query := `
		SELECT
			id, sbom_id, package_id, cve_id, severity,
			cvss_score, COALESCE(cvss_vector, '') as cvss_vector,
			COALESCE(description, '') as description,
			COALESCE(fixed_version, '') as fixed_version,
			published_date, modified_date, references,
			COALESCE(data_source, '') as data_source,
			exploit_available, created_at, updated_at
		FROM sbom_vulnerabilities
		WHERE sbom_id = $1
	`

	args := []interface{}{sbomID}
	argIdx := 2

	// Apply filters
	if filter != nil {
		if len(filter.Severities) > 0 {
			query += fmt.Sprintf(" AND severity = ANY($%d)", argIdx)
			args = append(args, filter.Severities)
			argIdx++
		}
		if filter.MinCVSS != nil {
			query += fmt.Sprintf(" AND cvss_score >= $%d", argIdx)
			args = append(args, *filter.MinCVSS)
			argIdx++
		}
		if filter.HasExploit != nil {
			query += fmt.Sprintf(" AND exploit_available = $%d", argIdx)
			args = append(args, *filter.HasExploit)
			argIdx++
		}
		if filter.FixAvailable != nil {
			if *filter.FixAvailable {
				query += " AND fixed_version IS NOT NULL AND fixed_version != ''"
			} else {
				query += " AND (fixed_version IS NULL OR fixed_version = '')"
			}
		}
	}

	query += " ORDER BY CASE severity WHEN 'critical' THEN 1 WHEN 'high' THEN 2 WHEN 'medium' THEN 3 WHEN 'low' THEN 4 ELSE 5 END, cvss_score DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query vulnerabilities: %w", err)
	}
	defer rows.Close()

	var vulns []Vulnerability
	for rows.Next() {
		var vuln Vulnerability
		var refsJSON []byte

		if err := rows.Scan(
			&vuln.ID,
			&vuln.SBOMID,
			&vuln.PackageID,
			&vuln.CVEID,
			&vuln.Severity,
			&vuln.CVSSScore,
			&vuln.CVSSVector,
			&vuln.Description,
			&vuln.FixedVersion,
			&vuln.PublishedDate,
			&vuln.ModifiedDate,
			&refsJSON,
			&vuln.DataSource,
			&vuln.ExploitAvailable,
			&vuln.CreatedAt,
			&vuln.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan vulnerability: %w", err)
		}

		if len(refsJSON) > 0 {
			if err := json.Unmarshal(refsJSON, &vuln.References); err != nil {
				return nil, fmt.Errorf("unmarshal references: %w", err)
			}
		}

		vulns = append(vulns, vuln)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate vulnerabilities: %w", err)
	}

	return vulns, nil
}

// nullString returns a sql.NullString for empty strings.
func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
