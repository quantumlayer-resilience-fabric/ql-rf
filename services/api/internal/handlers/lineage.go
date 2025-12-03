package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/pkg/models"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/middleware"
)

// LineageHandler handles image lineage-related requests.
type LineageHandler struct {
	db  *pgxpool.Pool
	log *logger.Logger
}

// NewLineageHandler creates a new LineageHandler.
func NewLineageHandler(db *pgxpool.Pool, log *logger.Logger) *LineageHandler {
	return &LineageHandler{
		db:  db,
		log: log.WithComponent("lineage-handler"),
	}
}

// GetLineage returns the full lineage for an image including parents, children, builds, and vulnerabilities.
func (h *LineageHandler) GetLineage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	imageID := chi.URLParam(r, "id")
	id, err := uuid.Parse(imageID)
	if err != nil {
		http.Error(w, "invalid image ID", http.StatusBadRequest)
		return
	}

	// Get the image
	var image models.Image
	err = h.db.QueryRow(ctx, `
		SELECT id, org_id, family, version, os_name, os_version, cis_level, sbom_url, signed, status, created_at, updated_at
		FROM images WHERE id = $1 AND org_id = $2
	`, id, org.ID).Scan(
		&image.ID, &image.OrgID, &image.Family, &image.Version,
		&image.OSName, &image.OSVersion, &image.CISLevel, &image.SBOMUrl,
		&image.Signed, &image.Status, &image.CreatedAt, &image.UpdatedAt,
	)
	if err != nil {
		http.Error(w, "image not found", http.StatusNotFound)
		return
	}

	// Get parent lineage
	parents, err := h.getParents(ctx, id)
	if err != nil {
		h.log.Error("failed to get parents", "error", err)
	}

	// Get child lineage
	children, err := h.getChildren(ctx, id)
	if err != nil {
		h.log.Error("failed to get children", "error", err)
	}

	// Get builds
	builds, err := h.getBuilds(ctx, id)
	if err != nil {
		h.log.Error("failed to get builds", "error", err)
	}

	// Get vulnerability summary
	vulnSummary, err := h.getVulnSummary(ctx, id)
	if err != nil {
		h.log.Error("failed to get vuln summary", "error", err)
	}

	// Get active deployments count
	var deploymentCount int
	_ = h.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM image_deployments WHERE image_id = $1 AND status = 'active'
	`, id).Scan(&deploymentCount)

	// Get promotions
	promotions, err := h.getPromotions(ctx, id)
	if err != nil {
		h.log.Error("failed to get promotions", "error", err)
	}

	response := models.ImageLineageResponse{
		Image:       &image,
		Parents:     parents,
		Children:    children,
		Builds:      builds,
		Vulns:       vulnSummary,
		Deployments: deploymentCount,
		Promotions:  promotions,
	}

	writeJSON(w, http.StatusOK, response)
}

// GetLineageTree returns the full lineage tree for an image family.
func (h *LineageHandler) GetLineageTree(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	family := chi.URLParam(r, "family")
	if family == "" {
		http.Error(w, "family is required", http.StatusBadRequest)
		return
	}

	// Get all images in the family
	rows, err := h.db.Query(ctx, `
		SELECT id, org_id, family, version, os_name, os_version, cis_level, sbom_url, signed, status, created_at, updated_at
		FROM images WHERE family = $1 AND org_id = $2
		ORDER BY created_at ASC
	`, family, org.ID)
	if err != nil {
		h.log.Error("failed to query images", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	imageMap := make(map[uuid.UUID]*models.Image)
	var images []*models.Image
	for rows.Next() {
		var img models.Image
		err := rows.Scan(
			&img.ID, &img.OrgID, &img.Family, &img.Version,
			&img.OSName, &img.OSVersion, &img.CISLevel, &img.SBOMUrl,
			&img.Signed, &img.Status, &img.CreatedAt, &img.UpdatedAt,
		)
		if err != nil {
			continue
		}
		imageMap[img.ID] = &img
		images = append(images, &img)
	}

	// Get all lineage relationships
	lineageRows, err := h.db.Query(ctx, `
		SELECT il.image_id, il.parent_image_id, il.relationship_type
		FROM image_lineage il
		JOIN images i ON i.id = il.image_id
		WHERE i.family = $1 AND i.org_id = $2
	`, family, org.ID)
	if err != nil {
		h.log.Error("failed to query lineage", "error", err)
	} else {
		defer lineageRows.Close()
	}

	// Build parent-child relationships
	childrenMap := make(map[uuid.UUID][]uuid.UUID)
	parentsMap := make(map[uuid.UUID][]uuid.UUID)

	if lineageRows != nil {
		for lineageRows.Next() {
			var imageID, parentID uuid.UUID
			var relType string
			if err := lineageRows.Scan(&imageID, &parentID, &relType); err != nil {
				continue
			}
			childrenMap[parentID] = append(childrenMap[parentID], imageID)
			parentsMap[imageID] = append(parentsMap[imageID], parentID)
		}
	}

	// Build tree nodes
	nodeMap := make(map[uuid.UUID]*models.LineageNode)
	for _, img := range images {
		nodeMap[img.ID] = &models.LineageNode{
			Image:    img,
			Depth:    0,
			Children: []*models.LineageNode{},
		}
	}

	// Link children
	for parentID, childIDs := range childrenMap {
		parentNode := nodeMap[parentID]
		if parentNode == nil {
			continue
		}
		for _, childID := range childIDs {
			childNode := nodeMap[childID]
			if childNode != nil {
				parentNode.Children = append(parentNode.Children, childNode)
			}
		}
	}

	// Find root nodes (images with no parents in this family)
	var roots []*models.LineageNode
	for id, node := range nodeMap {
		if _, hasParent := parentsMap[id]; !hasParent {
			roots = append(roots, node)
		}
	}

	// Calculate depths
	var calculateDepth func(node *models.LineageNode, depth int)
	calculateDepth = func(node *models.LineageNode, depth int) {
		node.Depth = depth
		for _, child := range node.Children {
			calculateDepth(child, depth+1)
		}
	}
	for _, root := range roots {
		calculateDepth(root, 0)
	}

	tree := models.ImageLineageTree{
		Family: family,
		Roots:  roots,
		Nodes:  len(images),
	}

	writeJSON(w, http.StatusOK, tree)
}

// AddParent adds a parent relationship to an image.
func (h *LineageHandler) AddParent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	imageID := chi.URLParam(r, "id")
	id, err := uuid.Parse(imageID)
	if err != nil {
		http.Error(w, "invalid image ID", http.StatusBadRequest)
		return
	}

	var req models.CreateLineageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Verify both images exist and belong to org
	var count int
	err = h.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM images WHERE id IN ($1, $2) AND org_id = $3
	`, id, req.ParentImageID, org.ID).Scan(&count)
	if err != nil || count != 2 {
		http.Error(w, "one or both images not found", http.StatusNotFound)
		return
	}

	// Prevent self-reference
	if id == req.ParentImageID {
		http.Error(w, "image cannot be its own parent", http.StatusBadRequest)
		return
	}

	// Insert lineage
	var lineage models.ImageLineage
	err = h.db.QueryRow(ctx, `
		INSERT INTO image_lineage (image_id, parent_image_id, relationship_type)
		VALUES ($1, $2, $3)
		ON CONFLICT (image_id, parent_image_id) DO UPDATE SET relationship_type = $3
		RETURNING id, image_id, parent_image_id, relationship_type, created_at
	`, id, req.ParentImageID, req.RelationshipType).Scan(
		&lineage.ID, &lineage.ImageID, &lineage.ParentImageID,
		&lineage.RelationshipType, &lineage.CreatedAt,
	)
	if err != nil {
		h.log.Error("failed to create lineage", "error", err)
		http.Error(w, "failed to create lineage", http.StatusInternalServerError)
		return
	}

	h.log.Info("lineage created",
		"image_id", id,
		"parent_id", req.ParentImageID,
		"type", req.RelationshipType,
	)

	writeJSON(w, http.StatusCreated, lineage)
}

// GetVulnerabilities returns vulnerabilities for an image.
func (h *LineageHandler) GetVulnerabilities(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	imageID := chi.URLParam(r, "id")
	id, err := uuid.Parse(imageID)
	if err != nil {
		http.Error(w, "invalid image ID", http.StatusBadRequest)
		return
	}

	// Verify image exists
	var exists bool
	_ = h.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM images WHERE id = $1 AND org_id = $2)`, id, org.ID).Scan(&exists)
	if !exists {
		http.Error(w, "image not found", http.StatusNotFound)
		return
	}

	// Get vulnerabilities
	rows, err := h.db.Query(ctx, `
		SELECT id, image_id, cve_id, severity, cvss_score, cvss_vector,
		       package_name, package_version, package_type, fixed_version,
		       status, status_reason, scanner, scanned_at,
		       fixed_in_image_id, resolved_at, resolved_by, created_at, updated_at
		FROM image_vulnerabilities
		WHERE image_id = $1
		ORDER BY
			CASE severity
				WHEN 'critical' THEN 1
				WHEN 'high' THEN 2
				WHEN 'medium' THEN 3
				WHEN 'low' THEN 4
				ELSE 5
			END,
			created_at DESC
	`, id)
	if err != nil {
		h.log.Error("failed to query vulnerabilities", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var vulns []models.ImageVulnerability
	for rows.Next() {
		var v models.ImageVulnerability
		err := rows.Scan(
			&v.ID, &v.ImageID, &v.CVEID, &v.Severity, &v.CVSSScore, &v.CVSSVector,
			&v.PackageName, &v.PackageVersion, &v.PackageType, &v.FixedVersion,
			&v.Status, &v.StatusReason, &v.Scanner, &v.ScannedAt,
			&v.FixedInImageID, &v.ResolvedAt, &v.ResolvedBy, &v.CreatedAt, &v.UpdatedAt,
		)
		if err != nil {
			continue
		}
		vulns = append(vulns, v)
	}

	writeJSON(w, http.StatusOK, vulns)
}

// AddVulnerability records a vulnerability for an image.
func (h *LineageHandler) AddVulnerability(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	imageID := chi.URLParam(r, "id")
	id, err := uuid.Parse(imageID)
	if err != nil {
		http.Error(w, "invalid image ID", http.StatusBadRequest)
		return
	}

	var req models.CreateVulnerabilityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Verify image exists
	var exists bool
	_ = h.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM images WHERE id = $1 AND org_id = $2)`, id, org.ID).Scan(&exists)
	if !exists {
		http.Error(w, "image not found", http.StatusNotFound)
		return
	}

	// Insert vulnerability
	var vuln models.ImageVulnerability
	err = h.db.QueryRow(ctx, `
		INSERT INTO image_vulnerabilities (
			image_id, cve_id, severity, cvss_score, cvss_vector,
			package_name, package_version, package_type, fixed_version,
			scanner, scanned_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
		RETURNING id, image_id, cve_id, severity, cvss_score, status, created_at
	`, id, req.CVEID, req.Severity, req.CVSSScore, req.CVSSVector,
		req.PackageName, req.PackageVersion, req.PackageType, req.FixedVersion,
		req.Scanner,
	).Scan(&vuln.ID, &vuln.ImageID, &vuln.CVEID, &vuln.Severity, &vuln.CVSSScore, &vuln.Status, &vuln.CreatedAt)
	if err != nil {
		h.log.Error("failed to create vulnerability", "error", err)
		http.Error(w, "failed to record vulnerability", http.StatusInternalServerError)
		return
	}

	h.log.Info("vulnerability recorded",
		"image_id", id,
		"cve_id", req.CVEID,
		"severity", req.Severity,
	)

	writeJSON(w, http.StatusCreated, vuln)
}

// GetBuilds returns build history for an image.
func (h *LineageHandler) GetBuilds(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	imageID := chi.URLParam(r, "id")
	id, err := uuid.Parse(imageID)
	if err != nil {
		http.Error(w, "invalid image ID", http.StatusBadRequest)
		return
	}

	builds, err := h.getBuilds(ctx, id)
	if err != nil {
		h.log.Error("failed to get builds", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, builds)
}

// GetDeployments returns deployment history for an image.
func (h *LineageHandler) GetDeployments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	imageID := chi.URLParam(r, "id")
	id, err := uuid.Parse(imageID)
	if err != nil {
		http.Error(w, "invalid image ID", http.StatusBadRequest)
		return
	}

	rows, err := h.db.Query(ctx, `
		SELECT d.id, d.image_id, d.asset_id, d.deployed_at, d.deployed_by,
		       d.deployment_method, d.status, d.replaced_at, d.replaced_by_image_id,
		       a.name as asset_name, a.platform, a.region
		FROM image_deployments d
		JOIN assets a ON a.id = d.asset_id
		WHERE d.image_id = $1 AND a.org_id = $2
		ORDER BY d.deployed_at DESC
		LIMIT 100
	`, id, org.ID)
	if err != nil {
		h.log.Error("failed to query deployments", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type DeploymentWithAsset struct {
		models.ImageDeployment
		AssetName string `json:"asset_name"`
		Platform  string `json:"platform"`
		Region    string `json:"region"`
	}

	var deployments []DeploymentWithAsset
	for rows.Next() {
		var d DeploymentWithAsset
		err := rows.Scan(
			&d.ID, &d.ImageID, &d.AssetID, &d.DeployedAt, &d.DeployedBy,
			&d.DeploymentMethod, &d.Status, &d.ReplacedAt, &d.ReplacedByImageID,
			&d.AssetName, &d.Platform, &d.Region,
		)
		if err != nil {
			continue
		}
		deployments = append(deployments, d)
	}

	writeJSON(w, http.StatusOK, deployments)
}

// GetComponents returns SBOM components for an image.
func (h *LineageHandler) GetComponents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	imageID := chi.URLParam(r, "id")
	id, err := uuid.Parse(imageID)
	if err != nil {
		http.Error(w, "invalid image ID", http.StatusBadRequest)
		return
	}

	rows, err := h.db.Query(ctx, `
		SELECT c.id, c.image_id, c.name, c.version, c.component_type,
		       c.package_manager, c.license, c.license_url, c.source_url, c.checksum, c.created_at
		FROM image_components c
		JOIN images i ON i.id = c.image_id
		WHERE c.image_id = $1 AND i.org_id = $2
		ORDER BY c.name ASC
	`, id, org.ID)
	if err != nil {
		h.log.Error("failed to query components", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var components []models.ImageComponent
	for rows.Next() {
		var c models.ImageComponent
		err := rows.Scan(
			&c.ID, &c.ImageID, &c.Name, &c.Version, &c.ComponentType,
			&c.PackageManager, &c.License, &c.LicenseURL, &c.SourceURL, &c.Checksum, &c.CreatedAt,
		)
		if err != nil {
			continue
		}
		components = append(components, c)
	}

	writeJSON(w, http.StatusOK, components)
}

// Helper functions

func (h *LineageHandler) getParents(ctx context.Context, imageID uuid.UUID) ([]models.ImageLineage, error) {
	rows, err := h.db.Query(ctx, `
		SELECT il.id, il.image_id, il.parent_image_id, il.relationship_type, il.created_at,
		       i.family, i.version, i.status
		FROM image_lineage il
		JOIN images i ON i.id = il.parent_image_id
		WHERE il.image_id = $1
	`, imageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lineages []models.ImageLineage
	for rows.Next() {
		var l models.ImageLineage
		var parentFamily, parentVersion, parentStatus string
		err := rows.Scan(
			&l.ID, &l.ImageID, &l.ParentImageID, &l.RelationshipType, &l.CreatedAt,
			&parentFamily, &parentVersion, &parentStatus,
		)
		if err != nil {
			continue
		}
		l.ParentImage = &models.Image{
			ID:      l.ParentImageID,
			Family:  parentFamily,
			Version: parentVersion,
			Status:  models.ImageStatus(parentStatus),
		}
		lineages = append(lineages, l)
	}
	return lineages, nil
}

func (h *LineageHandler) getChildren(ctx context.Context, imageID uuid.UUID) ([]models.ImageLineage, error) {
	rows, err := h.db.Query(ctx, `
		SELECT il.id, il.image_id, il.parent_image_id, il.relationship_type, il.created_at,
		       i.family, i.version, i.status
		FROM image_lineage il
		JOIN images i ON i.id = il.image_id
		WHERE il.parent_image_id = $1
	`, imageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lineages []models.ImageLineage
	for rows.Next() {
		var l models.ImageLineage
		var childFamily, childVersion, childStatus string
		err := rows.Scan(
			&l.ID, &l.ImageID, &l.ParentImageID, &l.RelationshipType, &l.CreatedAt,
			&childFamily, &childVersion, &childStatus,
		)
		if err != nil {
			continue
		}
		l.Image = &models.Image{
			ID:      l.ImageID,
			Family:  childFamily,
			Version: childVersion,
			Status:  models.ImageStatus(childStatus),
		}
		lineages = append(lineages, l)
	}
	return lineages, nil
}

func (h *LineageHandler) getBuilds(ctx context.Context, imageID uuid.UUID) ([]models.ImageBuild, error) {
	rows, err := h.db.Query(ctx, `
		SELECT id, image_id, build_number, source_repo, source_commit, source_branch,
		       builder_type, builder_version, build_runner, build_runner_id, build_runner_url,
		       build_log_url, build_duration_seconds, built_by, signed_by,
		       status, error_message, started_at, completed_at, created_at
		FROM image_builds
		WHERE image_id = $1
		ORDER BY build_number DESC
	`, imageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var builds []models.ImageBuild
	for rows.Next() {
		var b models.ImageBuild
		err := rows.Scan(
			&b.ID, &b.ImageID, &b.BuildNumber, &b.SourceRepo, &b.SourceCommit, &b.SourceBranch,
			&b.BuilderType, &b.BuilderVersion, &b.BuildRunner, &b.BuildRunnerID, &b.BuildRunnerURL,
			&b.BuildLogURL, &b.BuildDurationSeconds, &b.BuiltBy, &b.SignedBy,
			&b.Status, &b.ErrorMessage, &b.StartedAt, &b.CompletedAt, &b.CreatedAt,
		)
		if err != nil {
			continue
		}
		builds = append(builds, b)
	}
	return builds, nil
}

func (h *LineageHandler) getVulnSummary(ctx context.Context, imageID uuid.UUID) (models.VulnerabilitySummary, error) {
	var summary models.VulnerabilitySummary
	summary.ImageID = imageID

	err := h.db.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE severity = 'critical' AND status = 'open'),
			COUNT(*) FILTER (WHERE severity = 'high' AND status = 'open'),
			COUNT(*) FILTER (WHERE severity = 'medium' AND status = 'open'),
			COUNT(*) FILTER (WHERE severity = 'low' AND status = 'open'),
			COUNT(*) FILTER (WHERE status = 'fixed'),
			MAX(scanned_at)
		FROM image_vulnerabilities
		WHERE image_id = $1
	`, imageID).Scan(
		&summary.CriticalOpen, &summary.HighOpen, &summary.MediumOpen,
		&summary.LowOpen, &summary.FixedCount, &summary.LastScannedAt,
	)
	return summary, err
}

func (h *LineageHandler) getPromotions(ctx context.Context, imageID uuid.UUID) ([]models.ImagePromotion, error) {
	rows, err := h.db.Query(ctx, `
		SELECT id, image_id, from_status, to_status, promoted_by, approved_by,
		       approval_ticket, reason, validation_passed, promoted_at
		FROM image_promotions
		WHERE image_id = $1
		ORDER BY promoted_at DESC
	`, imageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var promotions []models.ImagePromotion
	for rows.Next() {
		var p models.ImagePromotion
		err := rows.Scan(
			&p.ID, &p.ImageID, &p.FromStatus, &p.ToStatus, &p.PromotedBy, &p.ApprovedBy,
			&p.ApprovalTicket, &p.Reason, &p.ValidationPassed, &p.PromotedAt,
		)
		if err != nil {
			continue
		}
		promotions = append(promotions, p)
	}
	return promotions, nil
}

// ImportScanResults imports vulnerability scan results from security scanners (Trivy, Grype, Snyk, etc.)
func (h *LineageHandler) ImportScanResults(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	imageID := chi.URLParam(r, "id")
	id, err := uuid.Parse(imageID)
	if err != nil {
		http.Error(w, "invalid image ID", http.StatusBadRequest)
		return
	}

	// Verify image exists
	var exists bool
	_ = h.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM images WHERE id = $1 AND org_id = $2)`, id, org.ID).Scan(&exists)
	if !exists {
		http.Error(w, "image not found", http.StatusNotFound)
		return
	}

	var req models.ImportScanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate scanner type
	validScanners := map[string]bool{
		"trivy": true, "grype": true, "snyk": true, "clair": true,
		"anchore": true, "aqua": true, "twistlock": true, "qualys": true,
	}
	if !validScanners[req.Scanner] {
		http.Error(w, "unsupported scanner type", http.StatusBadRequest)
		return
	}

	// Begin transaction for bulk insert
	tx, err := h.db.Begin(ctx)
	if err != nil {
		h.log.Error("failed to begin transaction", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(ctx)

	// Mark all existing open vulnerabilities from this scanner as potentially stale
	_, err = tx.Exec(ctx, `
		UPDATE image_vulnerabilities
		SET status = 'stale'
		WHERE image_id = $1 AND scanner = $2 AND status = 'open'
	`, id, req.Scanner)
	if err != nil {
		h.log.Error("failed to mark stale vulns", "error", err)
	}

	var imported, updated, fixed int
	for _, v := range req.Vulnerabilities {
		// Check if vulnerability already exists
		var existingID uuid.UUID
		var existingStatus string
		err := tx.QueryRow(ctx, `
			SELECT id, status FROM image_vulnerabilities
			WHERE image_id = $1 AND cve_id = $2 AND package_name = $3
		`, id, v.CVEID, v.PackageName).Scan(&existingID, &existingStatus)

		if err == nil {
			// Update existing vulnerability
			_, err = tx.Exec(ctx, `
				UPDATE image_vulnerabilities
				SET severity = $1, cvss_score = $2, cvss_vector = $3,
				    package_version = $4, fixed_version = $5,
				    scanner = $6, scanned_at = NOW(), status = 'open', updated_at = NOW()
				WHERE id = $7
			`, v.Severity, v.CVSSScore, v.CVSSVector,
				v.PackageVersion, v.FixedVersion, req.Scanner, existingID)
			if err != nil {
				h.log.Error("failed to update vulnerability", "cve", v.CVEID, "error", err)
				continue
			}
			if existingStatus == "fixed" {
				// Re-opened vulnerability
				updated++
			}
			updated++
		} else {
			// Insert new vulnerability
			_, err = tx.Exec(ctx, `
				INSERT INTO image_vulnerabilities (
					image_id, cve_id, severity, cvss_score, cvss_vector,
					package_name, package_version, package_type, fixed_version,
					scanner, scanned_at, status
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), 'open')
			`, id, v.CVEID, v.Severity, v.CVSSScore, v.CVSSVector,
				v.PackageName, v.PackageVersion, v.PackageType, v.FixedVersion, req.Scanner)
			if err != nil {
				h.log.Error("failed to insert vulnerability", "cve", v.CVEID, "error", err)
				continue
			}
			imported++
		}
	}

	// Mark any remaining stale vulnerabilities as fixed (they weren't in the new scan)
	result, err := tx.Exec(ctx, `
		UPDATE image_vulnerabilities
		SET status = 'fixed', resolved_at = NOW()
		WHERE image_id = $1 AND scanner = $2 AND status = 'stale'
	`, id, req.Scanner)
	if err == nil {
		fixed = int(result.RowsAffected())
	}

	if err := tx.Commit(ctx); err != nil {
		h.log.Error("failed to commit transaction", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	h.log.Info("scan results imported",
		"image_id", id,
		"scanner", req.Scanner,
		"imported", imported,
		"updated", updated,
		"fixed", fixed,
	)

	writeJSON(w, http.StatusOK, models.ImportScanResponse{
		ImageID:  id,
		Scanner:  req.Scanner,
		Imported: imported,
		Updated:  updated,
		Fixed:    fixed,
	})
}

// ImportSBOM imports Software Bill of Materials for an image
func (h *LineageHandler) ImportSBOM(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	imageID := chi.URLParam(r, "id")
	id, err := uuid.Parse(imageID)
	if err != nil {
		http.Error(w, "invalid image ID", http.StatusBadRequest)
		return
	}

	// Verify image exists
	var exists bool
	_ = h.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM images WHERE id = $1 AND org_id = $2)`, id, org.ID).Scan(&exists)
	if !exists {
		http.Error(w, "image not found", http.StatusNotFound)
		return
	}

	var req models.ImportSBOMRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate format
	validFormats := map[string]bool{
		"spdx": true, "cyclonedx": true, "syft": true,
	}
	if !validFormats[req.Format] {
		http.Error(w, "unsupported SBOM format (supported: spdx, cyclonedx, syft)", http.StatusBadRequest)
		return
	}

	// Begin transaction
	tx, err := h.db.Begin(ctx)
	if err != nil {
		h.log.Error("failed to begin transaction", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(ctx)

	// Clear existing components for this image (full replace)
	_, err = tx.Exec(ctx, `DELETE FROM image_components WHERE image_id = $1`, id)
	if err != nil {
		h.log.Error("failed to clear existing components", "error", err)
	}

	var imported int
	for _, c := range req.Components {
		_, err = tx.Exec(ctx, `
			INSERT INTO image_components (
				image_id, name, version, component_type, package_manager,
				license, license_url, source_url, checksum
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, id, c.Name, c.Version, c.ComponentType, c.PackageManager,
			c.License, c.LicenseURL, c.SourceURL, c.Checksum)
		if err != nil {
			h.log.Error("failed to insert component", "name", c.Name, "error", err)
			continue
		}
		imported++
	}

	// Update image SBOM URL if provided
	if req.SBOMUrl != "" {
		_, err = tx.Exec(ctx, `UPDATE images SET sbom_url = $1, updated_at = NOW() WHERE id = $2`, req.SBOMUrl, id)
		if err != nil {
			h.log.Error("failed to update sbom_url", "error", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		h.log.Error("failed to commit transaction", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	h.log.Info("SBOM imported",
		"image_id", id,
		"format", req.Format,
		"components", imported,
	)

	writeJSON(w, http.StatusOK, models.ImportSBOMResponse{
		ImageID:    id,
		Format:     req.Format,
		Components: imported,
	})
}
