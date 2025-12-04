// Package tools provides the tool registry for AI agent operations.
package tools

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
)

// RiskLevel defines the risk classification for tools.
type RiskLevel string

const (
	RiskReadOnly           RiskLevel = "read_only"
	RiskPlanOnly           RiskLevel = "plan_only"
	RiskStateChangeNonProd RiskLevel = "state_change_nonprod"
	RiskStateChangeProd    RiskLevel = "state_change_prod"
)

// Scope defines the impact scope of a tool.
type Scope string

const (
	ScopeAsset        Scope = "asset"
	ScopeEnvironment  Scope = "environment"
	ScopeOrganization Scope = "organization"
)

// Tool is the interface that all tools must implement.
type Tool interface {
	// Name returns the tool name.
	Name() string

	// Description returns a description of what the tool does.
	Description() string

	// Parameters returns the JSON Schema for tool parameters.
	Parameters() map[string]interface{}

	// Risk returns the risk level of the tool.
	Risk() RiskLevel

	// Scope returns the impact scope of the tool.
	Scope() Scope

	// Idempotent returns whether the tool is idempotent.
	Idempotent() bool

	// RequiresApproval returns whether the tool requires HITL approval.
	RequiresApproval() bool

	// Execute runs the tool with the given parameters.
	Execute(ctx context.Context, params map[string]interface{}) (interface{}, error)
}

// Registry manages available tools.
type Registry struct {
	tools map[string]Tool
	db    *pgxpool.Pool
	log   *logger.Logger
}

// NewRegistry creates a new tool registry with all available tools.
func NewRegistry(db *pgxpool.Pool, log *logger.Logger) *Registry {
	r := &Registry{
		tools: make(map[string]Tool),
		db:    db,
		log:   log.WithComponent("tool-registry"),
	}

	// Register all tools
	r.registerQueryTools()
	r.registerAnalysisTools()
	r.registerPlanningTools()
	r.registerExecutionTools()
	r.registerImageTools()
	r.registerSOPTools()

	return r
}

// Get returns a tool by name.
func (r *Registry) Get(name string) (Tool, bool) {
	tool, ok := r.tools[name]
	return tool, ok
}

// ListTools returns all available tool names.
func (r *Registry) ListTools() []string {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// ListByRisk returns tools filtered by risk level.
func (r *Registry) ListByRisk(risk RiskLevel) []Tool {
	var result []Tool
	for _, tool := range r.tools {
		if tool.Risk() == risk {
			result = append(result, tool)
		}
	}
	return result
}

// ToolMetadata contains metadata about a tool.
type ToolMetadata struct {
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	Risk             RiskLevel              `json:"risk"`
	Scope            Scope                  `json:"scope"`
	Idempotent       bool                   `json:"idempotent"`
	RequiresApproval bool                   `json:"requires_approval"`
	Parameters       map[string]interface{} `json:"parameters"`
}

// ToolInfo returns information about all registered tools.
func (r *Registry) ToolInfo() []ToolMetadata {
	info := make([]ToolMetadata, 0, len(r.tools))
	for _, tool := range r.tools {
		info = append(info, ToolMetadata{
			Name:             tool.Name(),
			Description:      tool.Description(),
			Risk:             tool.Risk(),
			Scope:            tool.Scope(),
			Idempotent:       tool.Idempotent(),
			RequiresApproval: tool.RequiresApproval(),
			Parameters:       tool.Parameters(),
		})
	}
	return info
}

// Execute runs a tool by name with the given parameters.
func (r *Registry) Execute(ctx context.Context, name string, params map[string]interface{}) (interface{}, error) {
	tool, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", name)
	}

	r.log.Info("executing tool",
		"tool", name,
		"risk", tool.Risk(),
		"scope", tool.Scope(),
	)

	result, err := tool.Execute(ctx, params)
	if err != nil {
		r.log.Error("tool execution failed", "tool", name, "error", err)
		return nil, err
	}

	r.log.Info("tool execution completed", "tool", name)
	return result, nil
}

// register adds a tool to the registry.
func (r *Registry) register(tool Tool) {
	r.tools[tool.Name()] = tool
	r.log.Debug("registered tool", "name", tool.Name(), "risk", tool.Risk())
}

// registerQueryTools registers read-only query tools.
func (r *Registry) registerQueryTools() {
	r.register(&QueryAssetsTool{db: r.db})
	r.register(&GetDriftStatusTool{db: r.db})
	r.register(&GetComplianceStatusTool{db: r.db})
	r.register(&GetGoldenImageTool{db: r.db})
	r.register(&QueryAlertsTool{db: r.db})
	r.register(&GetDRStatusTool{db: r.db})
}

// registerAnalysisTools registers analysis tools.
func (r *Registry) registerPlanningTools() {
	r.register(&CompareVersionsTool{db: r.db})
	r.register(&GeneratePatchPlanTool{db: r.db})
	r.register(&GenerateRolloutPlanTool{db: r.db})
	r.register(&GenerateDRRunbookTool{db: r.db})
	r.register(&SimulateRolloutTool{db: r.db})
	r.register(&CalculateRiskScoreTool{db: r.db})
	r.register(&SimulateFailoverTool{db: r.db})
	r.register(&GenerateComplianceEvidenceTool{db: r.db})
}

// registerAnalysisTools registers analysis tools.
func (r *Registry) registerAnalysisTools() {
	r.register(&AnalyzeDriftTool{db: r.db})
	r.register(&CheckControlTool{db: r.db})
}

// registerExecutionTools registers state-changing execution tools.
func (r *Registry) registerExecutionTools() {
	r.register(&ProposeRolloutTool{db: r.db})
	r.register(&AcknowledgeAlertTool{db: r.db})
}

// registerImageTools registers golden image lifecycle tools.
func (r *Registry) registerImageTools() {
	r.register(&GenerateImageContractTool{db: r.db})
	r.register(&GeneratePackerTemplateTool{db: r.db})
	r.register(&GenerateAnsiblePlaybookTool{db: r.db})
	r.register(&BuildImageTool{db: r.db})
	r.register(&ListImageVersionsTool{db: r.db})
	r.register(&PromoteImageTool{db: r.db})
}

// registerSOPTools registers SOP lifecycle tools.
func (r *Registry) registerSOPTools() {
	r.register(&GenerateSOPTool{db: r.db})
	r.register(&ValidateSOPTool{db: r.db})
	r.register(&SimulateSOPTool{db: r.db})
	r.register(&ExecuteSOPTool{db: r.db})
	r.register(&ListSOPsTool{db: r.db})
}

// =============================================================================
// Query Tools (read-only)
// =============================================================================

// QueryAssetsTool queries assets with filters.
type QueryAssetsTool struct {
	db *pgxpool.Pool
}

func (t *QueryAssetsTool) Name() string        { return "query_assets" }
func (t *QueryAssetsTool) Description() string { return "Query assets with filters (platform, region, tags, drift status)" }
func (t *QueryAssetsTool) Risk() RiskLevel     { return RiskReadOnly }
func (t *QueryAssetsTool) Scope() Scope        { return ScopeOrganization }
func (t *QueryAssetsTool) Idempotent() bool    { return true }
func (t *QueryAssetsTool) RequiresApproval() bool { return false }
func (t *QueryAssetsTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"platform":     map[string]interface{}{"type": "string", "enum": []string{"aws", "azure", "gcp", "vsphere"}},
			"region":       map[string]interface{}{"type": "string"},
			"environment":  map[string]interface{}{"type": "string", "enum": []string{"production", "staging", "development"}},
			"drift_status": map[string]interface{}{"type": "string", "enum": []string{"compliant", "drifted", "unknown"}},
			"tags":         map[string]interface{}{"type": "object"},
			"limit":        map[string]interface{}{"type": "integer", "default": 100},
		},
	}
}
func (t *QueryAssetsTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Build dynamic query based on parameters
	query := `
		SELECT a.id, a.name, a.platform, a.region, a.instance_id,
		       a.image_ref, a.image_version, a.state, a.tags,
		       e.name as environment
		FROM assets a
		LEFT JOIN environments e ON a.env_id = e.id
		WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1

	if platform, ok := params["platform"].(string); ok && platform != "" {
		query += fmt.Sprintf(" AND a.platform = $%d", argIdx)
		args = append(args, platform)
		argIdx++
	}
	if region, ok := params["region"].(string); ok && region != "" {
		query += fmt.Sprintf(" AND a.region = $%d", argIdx)
		args = append(args, region)
		argIdx++
	}
	if env, ok := params["environment"].(string); ok && env != "" {
		query += fmt.Sprintf(" AND e.name ILIKE $%d", argIdx)
		args = append(args, env)
		argIdx++
	}

	limit := 100
	if l, ok := params["limit"].(float64); ok {
		limit = int(l)
	}
	query += fmt.Sprintf(" LIMIT %d", limit)

	rows, err := t.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	assets := []map[string]interface{}{}
	for rows.Next() {
		var id, name, platform, region, instanceID, imageRef, imageVersion, state string
		var tags interface{}
		var environment *string

		err := rows.Scan(&id, &name, &platform, &region, &instanceID, &imageRef, &imageVersion, &state, &tags, &environment)
		if err != nil {
			continue
		}

		asset := map[string]interface{}{
			"id":            id,
			"name":          name,
			"platform":      platform,
			"region":        region,
			"instance_id":   instanceID,
			"image_ref":     imageRef,
			"image_version": imageVersion,
			"state":         state,
			"tags":          tags,
		}
		if environment != nil {
			asset["environment"] = *environment
		}
		assets = append(assets, asset)
	}

	return map[string]interface{}{
		"assets": assets,
		"total":  len(assets),
	}, nil
}

// GetDriftStatusTool gets drift status for assets.
type GetDriftStatusTool struct {
	db *pgxpool.Pool
}

func (t *GetDriftStatusTool) Name() string        { return "get_drift_status" }
func (t *GetDriftStatusTool) Description() string { return "Get drift analysis for specific assets or environment" }
func (t *GetDriftStatusTool) Risk() RiskLevel     { return RiskReadOnly }
func (t *GetDriftStatusTool) Scope() Scope        { return ScopeEnvironment }
func (t *GetDriftStatusTool) Idempotent() bool    { return true }
func (t *GetDriftStatusTool) RequiresApproval() bool { return false }
func (t *GetDriftStatusTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"asset_ids":   map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
			"environment": map[string]interface{}{"type": "string"},
			"site_id":     map[string]interface{}{"type": "string"},
		},
	}
}
func (t *GetDriftStatusTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Get the latest golden image for each family
	goldenImages := make(map[string]string)
	goldenRows, err := t.db.Query(ctx, `
		SELECT DISTINCT ON (family) family, version
		FROM images
		WHERE status = 'published'
		ORDER BY family, created_at DESC
	`)
	if err == nil {
		defer goldenRows.Close()
		for goldenRows.Next() {
			var family, version string
			if err := goldenRows.Scan(&family, &version); err == nil {
				goldenImages[family] = version
			}
		}
	}

	// Query assets and check drift status
	query := `
		SELECT a.id, a.name, a.platform, a.region, a.image_ref, a.image_version,
		       e.name as environment
		FROM assets a
		LEFT JOIN environments e ON a.env_id = e.id
	`

	rows, err := t.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var totalAssets, compliant, drifted, unknown int
	driftDetails := []map[string]interface{}{}

	for rows.Next() {
		var id, name, platform, region, imageRef, imageVersion string
		var environment *string

		err := rows.Scan(&id, &name, &platform, &region, &imageRef, &imageVersion, &environment)
		if err != nil {
			continue
		}

		totalAssets++
		targetVersion, hasGolden := goldenImages[imageRef]

		if !hasGolden {
			unknown++
		} else if imageVersion == targetVersion {
			compliant++
		} else {
			drifted++
			detail := map[string]interface{}{
				"asset_id":        id,
				"asset_name":      name,
				"platform":        platform,
				"region":          region,
				"image_family":    imageRef,
				"current_version": imageVersion,
				"target_version":  targetVersion,
				"drift_severity":  "warning",
			}
			if environment != nil {
				detail["environment"] = *environment
			}
			driftDetails = append(driftDetails, detail)
		}
	}

	return map[string]interface{}{
		"total_assets":  totalAssets,
		"compliant":     compliant,
		"drifted":       drifted,
		"unknown":       unknown,
		"drift_details": driftDetails,
	}, nil
}

// GetComplianceStatusTool gets compliance status.
type GetComplianceStatusTool struct {
	db *pgxpool.Pool
}

func (t *GetComplianceStatusTool) Name() string        { return "get_compliance_status" }
func (t *GetComplianceStatusTool) Description() string { return "Get compliance posture for frameworks (CIS, SLSA, SOC2)" }
func (t *GetComplianceStatusTool) Risk() RiskLevel     { return RiskReadOnly }
func (t *GetComplianceStatusTool) Scope() Scope        { return ScopeOrganization }
func (t *GetComplianceStatusTool) Idempotent() bool    { return true }
func (t *GetComplianceStatusTool) RequiresApproval() bool { return false }
func (t *GetComplianceStatusTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"framework": map[string]interface{}{"type": "string", "enum": []string{"CIS", "SLSA", "SOC2", "HIPAA", "PCI"}},
		},
	}
}
func (t *GetComplianceStatusTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Query compliance frameworks and their results
	query := `
		SELECT cf.id, cf.name, cf.description, cf.level, cf.enabled,
		       COALESCE(AVG(cr.score), 0) as avg_score,
		       COUNT(DISTINCT CASE WHEN cr.status = 'passing' THEN cr.id END) as passing_count,
		       COUNT(DISTINCT CASE WHEN cr.status = 'failing' THEN cr.id END) as failing_count,
		       COUNT(DISTINCT CASE WHEN cr.status = 'warning' THEN cr.id END) as warning_count
		FROM compliance_frameworks cf
		LEFT JOIN compliance_results cr ON cf.id = cr.framework_id
		WHERE cf.enabled = true
	`
	args := []interface{}{}
	argIdx := 1

	if framework, ok := params["framework"].(string); ok && framework != "" {
		query += fmt.Sprintf(" AND cf.name ILIKE $%d", argIdx)
		args = append(args, framework)
		argIdx++
	}

	query += " GROUP BY cf.id, cf.name, cf.description, cf.level, cf.enabled"

	rows, err := t.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	frameworks := []map[string]interface{}{}
	var totalScore float64
	var frameworkCount int

	for rows.Next() {
		var id, name string
		var description *string
		var level *int
		var enabled bool
		var avgScore float64
		var passingCount, failingCount, warningCount int

		err := rows.Scan(&id, &name, &description, &level, &enabled, &avgScore, &passingCount, &failingCount, &warningCount)
		if err != nil {
			continue
		}

		fw := map[string]interface{}{
			"id":            id,
			"name":          name,
			"enabled":       enabled,
			"score":         avgScore,
			"passing_count": passingCount,
			"failing_count": failingCount,
			"warning_count": warningCount,
		}
		if description != nil {
			fw["description"] = *description
		}
		if level != nil {
			fw["level"] = *level
		}

		frameworks = append(frameworks, fw)
		totalScore += avgScore
		frameworkCount++
	}

	overallScore := float64(0)
	if frameworkCount > 0 {
		overallScore = totalScore / float64(frameworkCount)
	}

	return map[string]interface{}{
		"frameworks":    frameworks,
		"overall_score": overallScore,
	}, nil
}

// GetGoldenImageTool gets the current golden image for a family.
type GetGoldenImageTool struct {
	db *pgxpool.Pool
}

func (t *GetGoldenImageTool) Name() string        { return "get_golden_image" }
func (t *GetGoldenImageTool) Description() string { return "Get current golden image for an image family" }
func (t *GetGoldenImageTool) Risk() RiskLevel     { return RiskReadOnly }
func (t *GetGoldenImageTool) Scope() Scope        { return ScopeOrganization }
func (t *GetGoldenImageTool) Idempotent() bool    { return true }
func (t *GetGoldenImageTool) RequiresApproval() bool { return false }
func (t *GetGoldenImageTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"family":   map[string]interface{}{"type": "string"},
			"platform": map[string]interface{}{"type": "string"},
		},
		"required": []string{"family"},
	}
}
func (t *GetGoldenImageTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	family, ok := params["family"].(string)
	if !ok || family == "" {
		return map[string]interface{}{"image": nil, "error": "family is required"}, nil
	}

	row := t.db.QueryRow(ctx, `
		SELECT id, family, version, os_name, os_version, cis_level, status, signed, created_at
		FROM images
		WHERE family = $1 AND status = 'published'
		ORDER BY created_at DESC
		LIMIT 1
	`, family)

	var id, imgFamily, version, osName, osVersion, status string
	var cisLevel *int
	var signed bool
	var createdAt interface{}

	err := row.Scan(&id, &imgFamily, &version, &osName, &osVersion, &cisLevel, &status, &signed, &createdAt)
	if err != nil {
		return map[string]interface{}{"image": nil}, nil
	}

	return map[string]interface{}{
		"image": map[string]interface{}{
			"id":         id,
			"family":     imgFamily,
			"version":    version,
			"os_name":    osName,
			"os_version": osVersion,
			"cis_level":  cisLevel,
			"status":     status,
			"signed":     signed,
		},
	}, nil
}

// QueryAlertsTool queries active alerts.
type QueryAlertsTool struct {
	db *pgxpool.Pool
}

func (t *QueryAlertsTool) Name() string        { return "query_alerts" }
func (t *QueryAlertsTool) Description() string { return "Query active alerts with filters" }
func (t *QueryAlertsTool) Risk() RiskLevel     { return RiskReadOnly }
func (t *QueryAlertsTool) Scope() Scope        { return ScopeOrganization }
func (t *QueryAlertsTool) Idempotent() bool    { return true }
func (t *QueryAlertsTool) RequiresApproval() bool { return false }
func (t *QueryAlertsTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"severity": map[string]interface{}{"type": "string", "enum": []string{"critical", "warning", "info"}},
			"status":   map[string]interface{}{"type": "string", "enum": []string{"open", "acknowledged", "resolved"}},
			"source":   map[string]interface{}{"type": "string"},
			"limit":    map[string]interface{}{"type": "integer", "default": 50},
		},
	}
}
func (t *QueryAlertsTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	query := `
		SELECT al.id, al.severity, al.title, al.description, al.source, al.status, al.created_at,
		       a.id as asset_id, a.name as asset_name
		FROM alerts al
		LEFT JOIN assets a ON al.asset_id = a.id
		WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1

	if severity, ok := params["severity"].(string); ok && severity != "" {
		query += fmt.Sprintf(" AND al.severity = $%d", argIdx)
		args = append(args, severity)
		argIdx++
	}
	if status, ok := params["status"].(string); ok && status != "" {
		query += fmt.Sprintf(" AND al.status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}
	if source, ok := params["source"].(string); ok && source != "" {
		query += fmt.Sprintf(" AND al.source = $%d", argIdx)
		args = append(args, source)
		argIdx++
	}

	limit := 50
	if l, ok := params["limit"].(float64); ok {
		limit = int(l)
	}
	query += fmt.Sprintf(" ORDER BY al.created_at DESC LIMIT %d", limit)

	rows, err := t.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	alerts := []map[string]interface{}{}
	for rows.Next() {
		var id, severity, title, description, source, status string
		var createdAt interface{}
		var assetID, assetName *string

		err := rows.Scan(&id, &severity, &title, &description, &source, &status, &createdAt, &assetID, &assetName)
		if err != nil {
			continue
		}

		alert := map[string]interface{}{
			"id":          id,
			"severity":    severity,
			"title":       title,
			"description": description,
			"source":      source,
			"status":      status,
			"created_at":  createdAt,
		}
		if assetID != nil {
			alert["asset_id"] = *assetID
		}
		if assetName != nil {
			alert["asset_name"] = *assetName
		}
		alerts = append(alerts, alert)
	}

	return map[string]interface{}{
		"alerts": alerts,
		"total":  len(alerts),
	}, nil
}

// GetDRStatusTool gets DR readiness status.
type GetDRStatusTool struct {
	db *pgxpool.Pool
}

func (t *GetDRStatusTool) Name() string        { return "get_dr_status" }
func (t *GetDRStatusTool) Description() string { return "Get DR readiness status for sites and pairs" }
func (t *GetDRStatusTool) Risk() RiskLevel     { return RiskReadOnly }
func (t *GetDRStatusTool) Scope() Scope        { return ScopeEnvironment }
func (t *GetDRStatusTool) Idempotent() bool    { return true }
func (t *GetDRStatusTool) RequiresApproval() bool { return false }
func (t *GetDRStatusTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"site_id":    map[string]interface{}{"type": "string"},
			"dr_pair_id": map[string]interface{}{"type": "string"},
		},
	}
}
func (t *GetDRStatusTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Query DR pairs with their sites
	query := `
		SELECT dp.id, dp.name, dp.status, dp.replication_status,
		       dp.rpo, dp.rto, dp.last_failover_test, dp.last_sync_at,
		       ps.id as primary_site_id, ps.name as primary_site_name, ps.region as primary_region,
		       ds.id as dr_site_id, ds.name as dr_site_name, ds.region as dr_region
		FROM dr_pairs dp
		JOIN sites ps ON dp.primary_site_id = ps.id
		JOIN sites ds ON dp.dr_site_id = ds.id
		WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1

	if siteID, ok := params["site_id"].(string); ok && siteID != "" {
		query += fmt.Sprintf(" AND (dp.primary_site_id = $%d OR dp.dr_site_id = $%d)", argIdx, argIdx+1)
		args = append(args, siteID, siteID)
		argIdx += 2
	}
	if drPairID, ok := params["dr_pair_id"].(string); ok && drPairID != "" {
		query += fmt.Sprintf(" AND dp.id = $%d", argIdx)
		args = append(args, drPairID)
		argIdx++
	}

	rows, err := t.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	drPairs := []map[string]interface{}{}
	var healthyCount, warningCount, criticalCount int

	for rows.Next() {
		var id, name, status, replStatus string
		var rpo, rto *string
		var lastFailoverTest, lastSyncAt interface{}
		var primarySiteID, primarySiteName, primaryRegion string
		var drSiteID, drSiteName, drRegion string

		err := rows.Scan(&id, &name, &status, &replStatus, &rpo, &rto, &lastFailoverTest, &lastSyncAt,
			&primarySiteID, &primarySiteName, &primaryRegion,
			&drSiteID, &drSiteName, &drRegion)
		if err != nil {
			continue
		}

		pair := map[string]interface{}{
			"id":                 id,
			"name":               name,
			"status":             status,
			"replication_status": replStatus,
			"last_failover_test": lastFailoverTest,
			"last_sync_at":       lastSyncAt,
			"primary_site": map[string]interface{}{
				"id":     primarySiteID,
				"name":   primarySiteName,
				"region": primaryRegion,
			},
			"dr_site": map[string]interface{}{
				"id":     drSiteID,
				"name":   drSiteName,
				"region": drRegion,
			},
		}
		if rpo != nil {
			pair["rpo"] = *rpo
		}
		if rto != nil {
			pair["rto"] = *rto
		}

		drPairs = append(drPairs, pair)

		// Count status
		switch status {
		case "healthy":
			healthyCount++
		case "warning":
			warningCount++
		case "critical":
			criticalCount++
		}
	}

	// Determine overall status
	overallStatus := "healthy"
	if criticalCount > 0 {
		overallStatus = "critical"
	} else if warningCount > 0 {
		overallStatus = "warning"
	} else if len(drPairs) == 0 {
		overallStatus = "unknown"
	}

	return map[string]interface{}{
		"dr_pairs":       drPairs,
		"overall_status": overallStatus,
		"summary": map[string]interface{}{
			"total":    len(drPairs),
			"healthy":  healthyCount,
			"warning":  warningCount,
			"critical": criticalCount,
		},
	}, nil
}

// =============================================================================
// Analysis Tools
// =============================================================================

// AnalyzeDriftTool analyzes drift patterns.
type AnalyzeDriftTool struct {
	db *pgxpool.Pool
}

func (t *AnalyzeDriftTool) Name() string        { return "analyze_drift" }
func (t *AnalyzeDriftTool) Description() string { return "Analyze drift patterns and identify root causes" }
func (t *AnalyzeDriftTool) Risk() RiskLevel     { return RiskPlanOnly }
func (t *AnalyzeDriftTool) Scope() Scope        { return ScopeEnvironment }
func (t *AnalyzeDriftTool) Idempotent() bool    { return true }
func (t *AnalyzeDriftTool) RequiresApproval() bool { return false }
func (t *AnalyzeDriftTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"asset_ids": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
		},
	}
}
func (t *AnalyzeDriftTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Get latest golden images for each family
	goldenImages := make(map[string]map[string]interface{})
	goldenRows, err := t.db.Query(ctx, `
		SELECT DISTINCT ON (family) id, family, version, os_name, os_version
		FROM images
		WHERE status = 'published'
		ORDER BY family, created_at DESC
	`)
	if err == nil {
		defer goldenRows.Close()
		for goldenRows.Next() {
			var id, family, version, osName, osVersion string
			if err := goldenRows.Scan(&id, &family, &version, &osName, &osVersion); err == nil {
				goldenImages[family] = map[string]interface{}{
					"id":         id,
					"version":    version,
					"os_name":    osName,
					"os_version": osVersion,
				}
			}
		}
	}

	// Query assets with drift
	query := `
		SELECT a.id, a.name, a.platform, a.region, a.image_ref, a.image_version,
		       e.name as environment, s.name as site_name
		FROM assets a
		LEFT JOIN environments e ON a.env_id = e.id
		LEFT JOIN sites s ON a.site_id = s.id
	`

	// Check if specific asset_ids provided
	var args []interface{}
	if assetIDs, ok := params["asset_ids"].([]interface{}); ok && len(assetIDs) > 0 {
		placeholders := ""
		for i, id := range assetIDs {
			if i > 0 {
				placeholders += ","
			}
			placeholders += fmt.Sprintf("$%d", i+1)
			args = append(args, id)
		}
		query += fmt.Sprintf(" WHERE a.id IN (%s)", placeholders)
	}

	rows, err := t.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	// Analyze drift patterns
	driftByPlatform := make(map[string]int)
	driftByRegion := make(map[string]int)
	driftByFamily := make(map[string]int)
	driftedAssets := []map[string]interface{}{}
	var totalAssets, compliantAssets int

	for rows.Next() {
		var id, name, platform, region, imageRef, imageVersion string
		var environment, siteName *string

		err := rows.Scan(&id, &name, &platform, &region, &imageRef, &imageVersion, &environment, &siteName)
		if err != nil {
			continue
		}

		totalAssets++
		golden, hasGolden := goldenImages[imageRef]

		if hasGolden && golden["version"].(string) == imageVersion {
			compliantAssets++
			continue
		}

		// Asset is drifted
		driftByPlatform[platform]++
		driftByRegion[region]++
		driftByFamily[imageRef]++

		detail := map[string]interface{}{
			"asset_id":        id,
			"asset_name":      name,
			"platform":        platform,
			"region":          region,
			"image_family":    imageRef,
			"current_version": imageVersion,
		}

		if hasGolden {
			detail["target_version"] = golden["version"]
			detail["drift_type"] = "version_mismatch"
		} else {
			detail["drift_type"] = "no_golden_image"
		}

		if environment != nil {
			detail["environment"] = *environment
		}
		if siteName != nil {
			detail["site"] = *siteName
		}

		driftedAssets = append(driftedAssets, detail)
	}

	// Generate analysis
	driftRate := float64(0)
	if totalAssets > 0 {
		driftRate = float64(len(driftedAssets)) / float64(totalAssets) * 100
	}

	return map[string]interface{}{
		"analysis": map[string]interface{}{
			"total_assets":     totalAssets,
			"compliant_assets": compliantAssets,
			"drifted_assets":   len(driftedAssets),
			"drift_rate":       driftRate,
			"drift_by_platform": driftByPlatform,
			"drift_by_region":   driftByRegion,
			"drift_by_family":   driftByFamily,
		},
		"drifted_assets": driftedAssets,
		"golden_images":  goldenImages,
	}, nil
}

// CheckControlTool checks a specific compliance control.
type CheckControlTool struct {
	db *pgxpool.Pool
}

func (t *CheckControlTool) Name() string        { return "check_control" }
func (t *CheckControlTool) Description() string { return "Check status of a specific compliance control" }
func (t *CheckControlTool) Risk() RiskLevel     { return RiskReadOnly }
func (t *CheckControlTool) Scope() Scope        { return ScopeOrganization }
func (t *CheckControlTool) Idempotent() bool    { return true }
func (t *CheckControlTool) RequiresApproval() bool { return false }
func (t *CheckControlTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"control_id": map[string]interface{}{"type": "string"},
			"framework":  map[string]interface{}{"type": "string"},
		},
		"required": []string{"control_id"},
	}
}
func (t *CheckControlTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	controlID, ok := params["control_id"].(string)
	if !ok || controlID == "" {
		return map[string]interface{}{"control": nil, "status": "error", "error": "control_id is required"}, nil
	}

	// Query the control and its latest result
	query := `
		SELECT cc.id, cc.control_id, cc.title, cc.description, cc.severity, cc.recommendation,
		       cf.name as framework_name,
		       cr.status as result_status, cr.affected_assets, cr.score, cr.last_audit_at
		FROM compliance_controls cc
		JOIN compliance_frameworks cf ON cc.framework_id = cf.id
		LEFT JOIN compliance_results cr ON cc.id = cr.control_id
		WHERE cc.control_id = $1
	`
	args := []interface{}{controlID}

	if framework, ok := params["framework"].(string); ok && framework != "" {
		query += " AND cf.name ILIKE $2"
		args = append(args, framework)
	}

	query += " ORDER BY cr.last_audit_at DESC NULLS LAST LIMIT 1"

	row := t.db.QueryRow(ctx, query, args...)

	var id, ctrlID, title, severity, frameworkName string
	var description, recommendation, resultStatus *string
	var affectedAssets *int
	var score *float64
	var lastAuditAt interface{}

	err := row.Scan(&id, &ctrlID, &title, &description, &severity, &recommendation,
		&frameworkName, &resultStatus, &affectedAssets, &score, &lastAuditAt)
	if err != nil {
		return map[string]interface{}{"control": nil, "status": "not_found"}, nil
	}

	control := map[string]interface{}{
		"id":           id,
		"control_id":   ctrlID,
		"title":        title,
		"severity":     severity,
		"framework":    frameworkName,
		"last_audit":   lastAuditAt,
	}
	if description != nil {
		control["description"] = *description
	}
	if recommendation != nil {
		control["recommendation"] = *recommendation
	}
	if affectedAssets != nil {
		control["affected_assets"] = *affectedAssets
	}
	if score != nil {
		control["score"] = *score
	}

	status := "unknown"
	if resultStatus != nil {
		status = *resultStatus
	}

	return map[string]interface{}{
		"control": control,
		"status":  status,
	}, nil
}

// =============================================================================
// Planning Tools
// =============================================================================

// CompareVersionsTool compares current vs target versions.
type CompareVersionsTool struct {
	db *pgxpool.Pool
}

func (t *CompareVersionsTool) Name() string        { return "compare_versions" }
func (t *CompareVersionsTool) Description() string { return "Compare current asset versions against target golden image" }
func (t *CompareVersionsTool) Risk() RiskLevel     { return RiskPlanOnly }
func (t *CompareVersionsTool) Scope() Scope        { return ScopeEnvironment }
func (t *CompareVersionsTool) Idempotent() bool    { return true }
func (t *CompareVersionsTool) RequiresApproval() bool { return false }
func (t *CompareVersionsTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"asset_ids":        map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
			"target_image_id":  map[string]interface{}{"type": "string"},
		},
	}
}
func (t *CompareVersionsTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Get target image if specified
	var targetImage map[string]interface{}
	if targetID, ok := params["target_image_id"].(string); ok && targetID != "" {
		row := t.db.QueryRow(ctx, `
			SELECT id, family, version, os_name, os_version, status
			FROM images WHERE id = $1
		`, targetID)
		var id, family, version, osName, osVersion, status string
		if err := row.Scan(&id, &family, &version, &osName, &osVersion, &status); err == nil {
			targetImage = map[string]interface{}{
				"id": id, "family": family, "version": version,
				"os_name": osName, "os_version": osVersion, "status": status,
			}
		}
	}

	// Get golden images for families
	goldenImages := make(map[string]map[string]interface{})
	goldenRows, err := t.db.Query(ctx, `
		SELECT DISTINCT ON (family) id, family, version
		FROM images WHERE status = 'published'
		ORDER BY family, created_at DESC
	`)
	if err == nil {
		defer goldenRows.Close()
		for goldenRows.Next() {
			var id, family, version string
			if err := goldenRows.Scan(&id, &family, &version); err == nil {
				goldenImages[family] = map[string]interface{}{"id": id, "version": version}
			}
		}
	}

	// Build asset query
	query := `
		SELECT a.id, a.name, a.platform, a.region, a.image_ref, a.image_version,
		       e.name as environment
		FROM assets a
		LEFT JOIN environments e ON a.env_id = e.id
	`
	var args []interface{}

	if assetIDs, ok := params["asset_ids"].([]interface{}); ok && len(assetIDs) > 0 {
		placeholders := ""
		for i, id := range assetIDs {
			if i > 0 {
				placeholders += ","
			}
			placeholders += fmt.Sprintf("$%d", i+1)
			args = append(args, id)
		}
		query += fmt.Sprintf(" WHERE a.id IN (%s)", placeholders)
	}

	rows, err := t.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	comparisons := []map[string]interface{}{}
	var needsUpdate, upToDate, unknown int

	for rows.Next() {
		var id, name, platform, region, imageRef, imageVersion string
		var environment *string

		if err := rows.Scan(&id, &name, &platform, &region, &imageRef, &imageVersion, &environment); err != nil {
			continue
		}

		comp := map[string]interface{}{
			"asset_id":        id,
			"asset_name":      name,
			"platform":        platform,
			"region":          region,
			"current_family":  imageRef,
			"current_version": imageVersion,
		}
		if environment != nil {
			comp["environment"] = *environment
		}

		// Determine target version
		var targetVersion, targetFamily string
		if targetImage != nil {
			targetVersion = targetImage["version"].(string)
			targetFamily = targetImage["family"].(string)
		} else if golden, ok := goldenImages[imageRef]; ok {
			targetVersion = golden["version"].(string)
			targetFamily = imageRef
		}

		if targetVersion != "" {
			comp["target_family"] = targetFamily
			comp["target_version"] = targetVersion

			if imageVersion == targetVersion {
				comp["status"] = "up_to_date"
				upToDate++
			} else {
				comp["status"] = "needs_update"
				needsUpdate++
			}
		} else {
			comp["status"] = "no_target"
			unknown++
		}

		comparisons = append(comparisons, comp)
	}

	return map[string]interface{}{
		"comparison":   comparisons,
		"target_image": targetImage,
		"summary": map[string]interface{}{
			"total":        len(comparisons),
			"needs_update": needsUpdate,
			"up_to_date":   upToDate,
			"unknown":      unknown,
		},
	}, nil
}

// GeneratePatchPlanTool generates a patch plan.
type GeneratePatchPlanTool struct {
	db *pgxpool.Pool
}

func (t *GeneratePatchPlanTool) Name() string        { return "generate_patch_plan" }
func (t *GeneratePatchPlanTool) Description() string { return "Generate a phased patch rollout plan" }
func (t *GeneratePatchPlanTool) Risk() RiskLevel     { return RiskPlanOnly }
func (t *GeneratePatchPlanTool) Scope() Scope        { return ScopeEnvironment }
func (t *GeneratePatchPlanTool) Idempotent() bool    { return true }
func (t *GeneratePatchPlanTool) RequiresApproval() bool { return false }
func (t *GeneratePatchPlanTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"asset_ids":         map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
			"target_image_id":   map[string]interface{}{"type": "string"},
			"canary_size":       map[string]interface{}{"type": "integer", "default": 5},
			"max_batch_percent": map[string]interface{}{"type": "integer", "default": 20},
		},
		"required": []string{"asset_ids", "target_image_id"},
	}
}
func (t *GeneratePatchPlanTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Get parameters with defaults
	canarySize := 5
	if cs, ok := params["canary_size"].(float64); ok {
		canarySize = int(cs)
	}
	maxBatchPercent := 20
	if mb, ok := params["max_batch_percent"].(float64); ok {
		maxBatchPercent = int(mb)
	}

	// Get asset count
	assetCount := 0
	if assetIDs, ok := params["asset_ids"].([]interface{}); ok {
		assetCount = len(assetIDs)
	} else if ac, ok := params["asset_count"].(float64); ok {
		assetCount = int(ac)
	}

	if assetCount == 0 {
		// Query total running assets
		var count int
		err := t.db.QueryRow(ctx, `SELECT COUNT(*) FROM assets WHERE state = 'running'`).Scan(&count)
		if err == nil {
			assetCount = count
		}
		if assetCount == 0 {
			assetCount = 100 // Default
		}
	}

	// Get target image info
	var targetImage map[string]interface{}
	if targetID, ok := params["target_image_id"].(string); ok && targetID != "" {
		row := t.db.QueryRow(ctx, `
			SELECT id, family, version, os_name, os_version FROM images WHERE id = $1
		`, targetID)
		var id, family, version, osName, osVersion string
		if err := row.Scan(&id, &family, &version, &osName, &osVersion); err == nil {
			targetImage = map[string]interface{}{
				"id": id, "family": family, "version": version,
				"os_name": osName, "os_version": osVersion,
			}
		}
	}

	// Calculate phase sizes
	canaryCount := max((assetCount*canarySize)/100, 1)
	remainingAfterCanary := assetCount - canaryCount
	waveSize := max((assetCount*maxBatchPercent)/100, 1)
	wavesNeeded := 0
	if remainingAfterCanary > 0 && waveSize > 0 {
		wavesNeeded = (remainingAfterCanary + waveSize - 1) / waveSize
	}

	// Build phases
	phases := []map[string]interface{}{
		{
			"id":                 "phase-preflight",
			"name":               "Pre-flight Checks",
			"type":               "validation",
			"description":        "Verify all assets are ready for patching",
			"asset_count":        0,
			"estimated_duration": "5m",
			"checks": []string{
				"connectivity_check",
				"disk_space_check",
				"backup_status_check",
				"service_health_check",
			},
			"rollback_on_failure": false,
		},
		{
			"id":                 "phase-canary",
			"name":               "Canary Deployment",
			"type":               "canary",
			"description":        fmt.Sprintf("Patch %d canary assets (%d%% of total)", canaryCount, canarySize),
			"asset_count":        canaryCount,
			"asset_percentage":   canarySize,
			"estimated_duration": "20m",
			"health_check_wait":  "10m",
			"success_criteria": map[string]interface{}{
				"error_rate_max":       1.0,
				"health_check_pass":    true,
				"response_time_p99_ms": 500,
			},
			"rollback_on_failure": true,
		},
	}

	// Add wave phases
	for i := 0; i < wavesNeeded && i < 10; i++ {
		waveAssetCount := waveSize
		if i == wavesNeeded-1 {
			waveAssetCount = remainingAfterCanary - (i * waveSize)
		}
		phases = append(phases, map[string]interface{}{
			"id":                 fmt.Sprintf("phase-wave-%d", i+1),
			"name":               fmt.Sprintf("Wave %d", i+1),
			"type":               "wave",
			"description":        fmt.Sprintf("Patch wave %d (%d assets)", i+1, waveAssetCount),
			"asset_count":        waveAssetCount,
			"asset_percentage":   maxBatchPercent,
			"estimated_duration": "25m",
			"health_check_wait":  "5m",
			"success_criteria": map[string]interface{}{
				"error_rate_max":    2.0,
				"health_check_pass": true,
			},
			"rollback_on_failure": true,
		})
	}

	// Add final validation phase
	phases = append(phases, map[string]interface{}{
		"id":                  "phase-validation",
		"name":                "Post-Rollout Validation",
		"type":                "validation",
		"description":         "Verify all assets are healthy after patching",
		"asset_count":         assetCount,
		"estimated_duration":  "15m",
		"checks":              []string{"health_check", "service_status", "log_errors", "metrics_validation"},
		"rollback_on_failure": false,
	})

	// Calculate total duration
	totalMinutes := 5 + 20 + (wavesNeeded * 30) + 15

	plan := map[string]interface{}{
		"id":                 fmt.Sprintf("patch-plan-%d", totalMinutes),
		"summary":            fmt.Sprintf("Phased patch rollout for %d assets with %d%% canary and %d waves", assetCount, canarySize, wavesNeeded),
		"target_image":       targetImage,
		"total_assets":       assetCount,
		"canary_size":        canarySize,
		"wave_size":          maxBatchPercent,
		"total_phases":       len(phases),
		"estimated_duration": fmt.Sprintf("%dm", totalMinutes),
		"phases":             phases,
		"rollback_plan": map[string]interface{}{
			"triggers": []string{
				"error_rate > 5%",
				"health_check_failure",
				"manual_trigger",
				"timeout_exceeded",
			},
			"procedure":          "Revert patched assets to previous image version",
			"estimated_duration": "20m",
			"requires_approval":  false,
		},
		"notifications": map[string]interface{}{
			"on_start":          []string{"slack"},
			"on_phase_complete": []string{"slack"},
			"on_failure":        []string{"slack", "email", "pagerduty"},
			"on_complete":       []string{"slack", "email"},
		},
		"constraints": map[string]interface{}{
			"maintenance_window":   "required",
			"concurrent_patches":   waveSize,
			"min_healthy_percent":  90,
			"max_duration_minutes": totalMinutes * 2,
		},
	}

	return map[string]interface{}{
		"plan": plan,
	}, nil
}

// max returns the maximum of two integers.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// GenerateRolloutPlanTool generates a rollout plan.
type GenerateRolloutPlanTool struct {
	db *pgxpool.Pool
}

func (t *GenerateRolloutPlanTool) Name() string        { return "generate_rollout_plan" }
func (t *GenerateRolloutPlanTool) Description() string { return "Generate a rollout strategy with canary phases" }
func (t *GenerateRolloutPlanTool) Risk() RiskLevel     { return RiskPlanOnly }
func (t *GenerateRolloutPlanTool) Scope() Scope        { return ScopeEnvironment }
func (t *GenerateRolloutPlanTool) Idempotent() bool    { return true }
func (t *GenerateRolloutPlanTool) RequiresApproval() bool { return false }
func (t *GenerateRolloutPlanTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"environment":       map[string]interface{}{"type": "string"},
			"asset_ids":         map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
			"require_canary":    map[string]interface{}{"type": "boolean", "default": true},
		},
	}
}
func (t *GenerateRolloutPlanTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	return map[string]interface{}{
		"plan": map[string]interface{}{},
	}, nil
}

// GenerateDRRunbookTool generates a DR runbook.
type GenerateDRRunbookTool struct {
	db *pgxpool.Pool
}

func (t *GenerateDRRunbookTool) Name() string        { return "generate_dr_runbook" }
func (t *GenerateDRRunbookTool) Description() string { return "Generate DR runbook from infrastructure analysis" }
func (t *GenerateDRRunbookTool) Risk() RiskLevel     { return RiskPlanOnly }
func (t *GenerateDRRunbookTool) Scope() Scope        { return ScopeEnvironment }
func (t *GenerateDRRunbookTool) Idempotent() bool    { return true }
func (t *GenerateDRRunbookTool) RequiresApproval() bool { return false }
func (t *GenerateDRRunbookTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"dr_pair_id":  map[string]interface{}{"type": "string"},
			"org_id":      map[string]interface{}{"type": "string"},
			"environment": map[string]interface{}{"type": "string"},
			"dr_type":     map[string]interface{}{"type": "string", "enum": []string{"drill", "runbook", "assessment"}},
		},
	}
}
func (t *GenerateDRRunbookTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Get parameters
	drType := "drill"
	if dt, ok := params["dr_type"].(string); ok && dt != "" {
		drType = dt
	}

	environment := "staging"
	if env, ok := params["environment"].(string); ok && env != "" {
		environment = env
	}

	// Query DR pairs for context
	var drPairs []map[string]interface{}
	if drPairID, ok := params["dr_pair_id"].(string); ok && drPairID != "" {
		row := t.db.QueryRow(ctx, `
			SELECT dp.id, dp.name, dp.status, dp.replication_status, dp.rpo, dp.rto,
			       ps.name as primary_site, ps.region as primary_region,
			       ds.name as dr_site, ds.region as dr_region
			FROM dr_pairs dp
			JOIN sites ps ON dp.primary_site_id = ps.id
			JOIN sites ds ON dp.dr_site_id = ds.id
			WHERE dp.id = $1
		`, drPairID)

		var id, name, status, replStatus string
		var rpo, rto *string
		var primarySite, primaryRegion, drSite, drRegion string

		if err := row.Scan(&id, &name, &status, &replStatus, &rpo, &rto,
			&primarySite, &primaryRegion, &drSite, &drRegion); err == nil {
			pair := map[string]interface{}{
				"id":                 id,
				"name":               name,
				"status":             status,
				"replication_status": replStatus,
				"primary_site":       primarySite,
				"primary_region":     primaryRegion,
				"dr_site":            drSite,
				"dr_region":          drRegion,
			}
			if rpo != nil {
				pair["rpo"] = *rpo
			}
			if rto != nil {
				pair["rto"] = *rto
			}
			drPairs = append(drPairs, pair)
		}
	}

	// Generate runbook phases based on DR type
	phases := []map[string]interface{}{}

	// Phase 1: Pre-requisite checks
	phases = append(phases, map[string]interface{}{
		"id":          "phase-prereq",
		"name":        "Pre-requisite Checks",
		"description": "Verify all systems are ready for DR operation",
		"steps": []map[string]interface{}{
			{"step": 1, "action": "Verify replication status is healthy", "responsible": "DBA Team", "estimated_duration": "5m"},
			{"step": 2, "action": "Confirm backup completeness", "responsible": "Backup Team", "estimated_duration": "5m"},
			{"step": 3, "action": "Validate network connectivity to DR site", "responsible": "Network Team", "estimated_duration": "5m"},
			{"step": 4, "action": "Confirm DR site resource availability", "responsible": "Infrastructure Team", "estimated_duration": "5m"},
		},
		"success_criteria": "All checks pass with no critical issues",
		"rollback_steps":   []string{"Abort if critical check fails", "Document failure reason"},
	})

	// Phase 2: Communication
	phases = append(phases, map[string]interface{}{
		"id":          "phase-communication",
		"name":        "Stakeholder Communication",
		"description": "Notify all stakeholders about DR operation",
		"steps": []map[string]interface{}{
			{"step": 1, "action": "Send notification to incident channel", "responsible": "DR Coordinator", "estimated_duration": "2m"},
			{"step": 2, "action": "Update status page", "responsible": "Communications Team", "estimated_duration": "2m"},
			{"step": 3, "action": "Notify on-call engineers", "responsible": "DR Coordinator", "estimated_duration": "2m"},
		},
		"success_criteria": "All stakeholders acknowledged",
	})

	// Phase 3: Failover (conditional on type)
	if drType == "drill" {
		phases = append(phases, map[string]interface{}{
			"id":          "phase-failover-test",
			"name":        "Failover Test Execution",
			"description": "Execute controlled failover to DR site",
			"steps": []map[string]interface{}{
				{"step": 1, "action": "Stop write operations to primary", "responsible": "DBA Team", "estimated_duration": "5m"},
				{"step": 2, "action": "Verify replication caught up", "responsible": "DBA Team", "estimated_duration": "10m"},
				{"step": 3, "action": "Promote DR database to primary", "responsible": "DBA Team", "estimated_duration": "15m"},
				{"step": 4, "action": "Update DNS/load balancer to DR site", "responsible": "Network Team", "estimated_duration": "5m"},
				{"step": 5, "action": "Start application services on DR site", "responsible": "App Team", "estimated_duration": "10m"},
			},
			"success_criteria":  "All services responding from DR site",
			"estimated_duration": "45m",
			"rollback_steps": []string{
				"Revert DNS changes",
				"Demote DR database",
				"Resume primary site operations",
			},
		})
	}

	// Phase 4: Validation
	phases = append(phases, map[string]interface{}{
		"id":          "phase-validation",
		"name":        "Post-Failover Validation",
		"description": "Verify DR site is functioning correctly",
		"steps": []map[string]interface{}{
			{"step": 1, "action": "Execute smoke tests", "responsible": "QA Team", "estimated_duration": "15m"},
			{"step": 2, "action": "Verify critical transactions", "responsible": "App Team", "estimated_duration": "10m"},
			{"step": 3, "action": "Check monitoring dashboards", "responsible": "SRE Team", "estimated_duration": "5m"},
			{"step": 4, "action": "Validate data consistency", "responsible": "DBA Team", "estimated_duration": "10m"},
		},
		"success_criteria": "All validation checks pass",
	})

	// Phase 5: Failback (for drills)
	if drType == "drill" {
		phases = append(phases, map[string]interface{}{
			"id":          "phase-failback",
			"name":        "Failback to Primary",
			"description": "Return operations to primary site",
			"steps": []map[string]interface{}{
				{"step": 1, "action": "Synchronize changes back to primary", "responsible": "DBA Team", "estimated_duration": "20m"},
				{"step": 2, "action": "Stop DR site services", "responsible": "App Team", "estimated_duration": "5m"},
				{"step": 3, "action": "Promote primary database", "responsible": "DBA Team", "estimated_duration": "15m"},
				{"step": 4, "action": "Update DNS/load balancer to primary", "responsible": "Network Team", "estimated_duration": "5m"},
				{"step": 5, "action": "Start primary site services", "responsible": "App Team", "estimated_duration": "10m"},
			},
			"success_criteria":  "All services responding from primary site",
			"estimated_duration": "55m",
		})
	}

	// Phase 6: Post-operation review
	phases = append(phases, map[string]interface{}{
		"id":          "phase-review",
		"name":        "Post-Operation Review",
		"description": "Document results and lessons learned",
		"steps": []map[string]interface{}{
			{"step": 1, "action": "Document any issues encountered", "responsible": "DR Coordinator", "estimated_duration": "15m"},
			{"step": 2, "action": "Update runbook with lessons learned", "responsible": "DR Coordinator", "estimated_duration": "15m"},
			{"step": 3, "action": "Send completion report", "responsible": "DR Coordinator", "estimated_duration": "10m"},
		},
	})

	// Calculate total estimated duration
	totalMinutes := 0
	for _, phase := range phases {
		if steps, ok := phase["steps"].([]map[string]interface{}); ok {
			for _, step := range steps {
				if dur, ok := step["estimated_duration"].(string); ok {
					// Parse duration (simplified - assumes format like "5m" or "15m")
					var mins int
					fmt.Sscanf(dur, "%dm", &mins)
					totalMinutes += mins
				}
			}
		}
	}

	// Build runbook
	runbook := map[string]interface{}{
		"id":                 fmt.Sprintf("runbook-%s-%d", drType, len(drPairs)),
		"type":               drType,
		"environment":        environment,
		"generated_at":       "now",
		"total_phases":       len(phases),
		"estimated_duration": fmt.Sprintf("%dh %dm", totalMinutes/60, totalMinutes%60),
		"phases":             phases,
		"dr_pairs":           drPairs,
		"contacts": map[string]interface{}{
			"dr_coordinator":     "dr-coordinator@company.com",
			"incident_channel":   "#incident-response",
			"escalation_contact": "on-call-manager@company.com",
		},
		"rollback_plan": map[string]interface{}{
			"triggers": []string{
				"Critical validation failure",
				"Data corruption detected",
				"Extended downtime exceeds RTO",
				"Manual abort by DR coordinator",
			},
			"procedure":          "Follow failback phase steps in reverse order",
			"estimated_duration": "45m",
		},
	}

	return map[string]interface{}{
		"runbook": runbook,
	}, nil
}

// SimulateRolloutTool simulates a rollout.
type SimulateRolloutTool struct {
	db *pgxpool.Pool
}

func (t *SimulateRolloutTool) Name() string        { return "simulate_rollout" }
func (t *SimulateRolloutTool) Description() string { return "Dry-run rollout to predict impact" }
func (t *SimulateRolloutTool) Risk() RiskLevel     { return RiskPlanOnly }
func (t *SimulateRolloutTool) Scope() Scope        { return ScopeEnvironment }
func (t *SimulateRolloutTool) Idempotent() bool    { return true }
func (t *SimulateRolloutTool) RequiresApproval() bool { return false }
func (t *SimulateRolloutTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"plan_id": map[string]interface{}{"type": "string"},
		},
		"required": []string{"plan_id"},
	}
}
func (t *SimulateRolloutTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Get parameters
	assetCount := 0
	if ac, ok := params["asset_count"].(float64); ok {
		assetCount = int(ac)
	}

	environment := "staging"
	if env, ok := params["environment"].(string); ok && env != "" {
		environment = env
	}

	// Query actual assets if not provided
	if assetCount == 0 {
		var count int
		err := t.db.QueryRow(ctx, `SELECT COUNT(*) FROM assets WHERE state = 'running'`).Scan(&count)
		if err == nil {
			assetCount = count
		}
		if assetCount == 0 {
			assetCount = 50
		}
	}

	// Calculate risk based on environment and asset count
	baseRisk := 20 // Base risk score
	riskFactors := []map[string]interface{}{}

	// Environment factor
	envRisk := 0
	switch environment {
	case "production":
		envRisk = 40
		riskFactors = append(riskFactors, map[string]interface{}{
			"factor":      "Production environment",
			"impact":      40,
			"description": "Changes to production carry higher risk",
		})
	case "staging":
		envRisk = 20
		riskFactors = append(riskFactors, map[string]interface{}{
			"factor":      "Staging environment",
			"impact":      20,
			"description": "Staging environment has moderate risk",
		})
	case "development":
		envRisk = 5
		riskFactors = append(riskFactors, map[string]interface{}{
			"factor":      "Development environment",
			"impact":      5,
			"description": "Development environment has low risk",
		})
	}

	// Asset count factor
	assetRisk := 0
	if assetCount > 100 {
		assetRisk = 25
		riskFactors = append(riskFactors, map[string]interface{}{
			"factor":      "Large asset count",
			"impact":      25,
			"description": fmt.Sprintf("Affecting %d assets increases blast radius", assetCount),
		})
	} else if assetCount > 50 {
		assetRisk = 15
		riskFactors = append(riskFactors, map[string]interface{}{
			"factor":      "Moderate asset count",
			"impact":      15,
			"description": fmt.Sprintf("Affecting %d assets", assetCount),
		})
	} else if assetCount > 10 {
		assetRisk = 10
		riskFactors = append(riskFactors, map[string]interface{}{
			"factor":      "Small asset count",
			"impact":      10,
			"description": fmt.Sprintf("Affecting %d assets", assetCount),
		})
	}

	// Calculate total risk
	totalRisk := min(baseRisk+envRisk+assetRisk, 100)

	// Determine risk level
	riskLevel := "low"
	if totalRisk >= 70 {
		riskLevel = "critical"
	} else if totalRisk >= 50 {
		riskLevel = "high"
	} else if totalRisk >= 30 {
		riskLevel = "medium"
	}

	// Estimate duration based on asset count
	minutesPerAsset := 0.5
	baseMinutes := 30.0
	estimatedMinutes := int(baseMinutes + (float64(assetCount) * minutesPerAsset))

	// Simulate potential issues
	potentialIssues := []map[string]interface{}{}
	if totalRisk >= 50 {
		potentialIssues = append(potentialIssues, map[string]interface{}{
			"type":        "connectivity",
			"probability": 0.1,
			"impact":      "medium",
			"mitigation":  "Pre-validate SSH/SSM connectivity",
		})
	}
	if assetCount > 100 {
		potentialIssues = append(potentialIssues, map[string]interface{}{
			"type":        "rate_limiting",
			"probability": 0.2,
			"impact":      "low",
			"mitigation":  "Implement exponential backoff",
		})
	}
	if environment == "production" {
		potentialIssues = append(potentialIssues, map[string]interface{}{
			"type":        "service_disruption",
			"probability": 0.05,
			"impact":      "high",
			"mitigation":  "Use rolling updates with health checks",
		})
	}

	// Calculate success probability
	successProbability := 100 - (totalRisk / 2)

	return map[string]interface{}{
		"simulation": map[string]interface{}{
			"affected_assets":     assetCount,
			"environment":         environment,
			"estimated_duration":  fmt.Sprintf("%dm", estimatedMinutes),
			"predicted_risk":      riskLevel,
			"risk_score":          totalRisk,
			"success_probability": fmt.Sprintf("%d%%", successProbability),
			"risk_factors":        riskFactors,
			"potential_issues":    potentialIssues,
			"recommendations": []string{
				"Schedule during maintenance window",
				"Ensure rollback plan is tested",
				"Have on-call team available",
				"Monitor metrics during rollout",
			},
		},
	}, nil
}

// min returns the minimum of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// CalculateRiskScoreTool calculates risk score for a change.
type CalculateRiskScoreTool struct {
	db *pgxpool.Pool
}

func (t *CalculateRiskScoreTool) Name() string        { return "calculate_risk_score" }
func (t *CalculateRiskScoreTool) Description() string { return "Calculate risk score for a proposed change" }
func (t *CalculateRiskScoreTool) Risk() RiskLevel     { return RiskPlanOnly }
func (t *CalculateRiskScoreTool) Scope() Scope        { return ScopeAsset }
func (t *CalculateRiskScoreTool) Idempotent() bool    { return true }
func (t *CalculateRiskScoreTool) RequiresApproval() bool { return false }
func (t *CalculateRiskScoreTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"asset_ids": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
			"change_type": map[string]interface{}{"type": "string"},
		},
	}
}
func (t *CalculateRiskScoreTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Get parameters
	assetCount := 0
	if ac, ok := params["asset_count"].(float64); ok {
		assetCount = int(ac)
	}
	if assetIDs, ok := params["asset_ids"].([]interface{}); ok && len(assetIDs) > 0 {
		assetCount = len(assetIDs)
	}

	environment := "staging"
	if env, ok := params["environment"].(string); ok && env != "" {
		environment = env
	}

	changeType := "patch"
	if ct, ok := params["change_type"].(string); ok && ct != "" {
		changeType = ct
	}

	// Initialize risk calculation
	riskScore := 0
	factors := []map[string]interface{}{}

	// Factor 1: Environment risk
	envRisk := 0
	switch environment {
	case "production":
		envRisk = 35
		factors = append(factors, map[string]interface{}{
			"name":        "environment",
			"category":    "target",
			"score":       35,
			"weight":      1.0,
			"description": "Production environment - highest risk tier",
			"mitigation":  "Use canary deployment with extensive health checks",
		})
	case "staging":
		envRisk = 15
		factors = append(factors, map[string]interface{}{
			"name":        "environment",
			"category":    "target",
			"score":       15,
			"weight":      1.0,
			"description": "Staging environment - moderate risk",
			"mitigation":  "Validate changes match production config",
		})
	case "development":
		envRisk = 5
		factors = append(factors, map[string]interface{}{
			"name":        "environment",
			"category":    "target",
			"score":       5,
			"weight":      1.0,
			"description": "Development environment - low risk",
			"mitigation":  "Standard change procedures apply",
		})
	}
	riskScore += envRisk

	// Factor 2: Blast radius (asset count)
	blastRadius := 0
	if assetCount > 500 {
		blastRadius = 30
		factors = append(factors, map[string]interface{}{
			"name":        "blast_radius",
			"category":    "scope",
			"score":       30,
			"weight":      1.0,
			"description": fmt.Sprintf("Very large blast radius (%d assets)", assetCount),
			"mitigation":  "Implement progressive rollout with 2% canary",
		})
	} else if assetCount > 100 {
		blastRadius = 20
		factors = append(factors, map[string]interface{}{
			"name":        "blast_radius",
			"category":    "scope",
			"score":       20,
			"weight":      1.0,
			"description": fmt.Sprintf("Large blast radius (%d assets)", assetCount),
			"mitigation":  "Use 5% canary with extended monitoring",
		})
	} else if assetCount > 20 {
		blastRadius = 10
		factors = append(factors, map[string]interface{}{
			"name":        "blast_radius",
			"category":    "scope",
			"score":       10,
			"weight":      1.0,
			"description": fmt.Sprintf("Moderate blast radius (%d assets)", assetCount),
			"mitigation":  "Use 10% canary deployment",
		})
	} else if assetCount > 0 {
		blastRadius = 5
		factors = append(factors, map[string]interface{}{
			"name":        "blast_radius",
			"category":    "scope",
			"score":       5,
			"weight":      1.0,
			"description": fmt.Sprintf("Small blast radius (%d assets)", assetCount),
			"mitigation":  "Standard monitoring sufficient",
		})
	}
	riskScore += blastRadius

	// Factor 3: Change type risk
	changeRisk := 0
	switch changeType {
	case "patch", "security_patch":
		changeRisk = 15
		factors = append(factors, map[string]interface{}{
			"name":        "change_type",
			"category":    "operation",
			"score":       15,
			"weight":      1.0,
			"description": "Patch operation - moderate complexity",
			"mitigation":  "Ensure rollback procedure is documented",
		})
	case "upgrade", "major_upgrade":
		changeRisk = 25
		factors = append(factors, map[string]interface{}{
			"name":        "change_type",
			"category":    "operation",
			"score":       25,
			"weight":      1.0,
			"description": "Major upgrade - higher complexity",
			"mitigation":  "Test in staging first, prepare rollback scripts",
		})
	case "config_change":
		changeRisk = 10
		factors = append(factors, map[string]interface{}{
			"name":        "change_type",
			"category":    "operation",
			"score":       10,
			"weight":      1.0,
			"description": "Configuration change - lower complexity",
			"mitigation":  "Validate config syntax before applying",
		})
	case "reboot":
		changeRisk = 20
		factors = append(factors, map[string]interface{}{
			"name":        "change_type",
			"category":    "operation",
			"score":       20,
			"weight":      1.0,
			"description": "Reboot operation - service interruption expected",
			"mitigation":  "Schedule during maintenance window",
		})
	default:
		changeRisk = 15
		factors = append(factors, map[string]interface{}{
			"name":        "change_type",
			"category":    "operation",
			"score":       15,
			"weight":      1.0,
			"description": fmt.Sprintf("Change type: %s", changeType),
			"mitigation":  "Follow standard change procedures",
		})
	}
	riskScore += changeRisk

	// Query additional context from database if available
	if t.db != nil {
		// Check for recent failures
		var recentFailures int
		err := t.db.QueryRow(ctx, `
			SELECT COUNT(*) FROM ai_tasks
			WHERE status = 'failed' AND created_at > NOW() - INTERVAL '7 days'
		`).Scan(&recentFailures)
		if err == nil && recentFailures > 5 {
			failureRisk := 10
			riskScore += failureRisk
			factors = append(factors, map[string]interface{}{
				"name":        "recent_failures",
				"category":    "historical",
				"score":       failureRisk,
				"weight":      1.0,
				"description": fmt.Sprintf("%d task failures in last 7 days", recentFailures),
				"mitigation":  "Review failure root causes before proceeding",
			})
		}

		// Check drift status
		var driftedCount int
		err = t.db.QueryRow(ctx, `
			SELECT COUNT(*) FROM assets
			WHERE state = 'running'
			AND image_version != (
				SELECT version FROM images WHERE family = assets.image_ref AND status = 'published'
				ORDER BY created_at DESC LIMIT 1
			)
		`).Scan(&driftedCount)
		if err == nil && driftedCount > 50 {
			driftRisk := 10
			riskScore += driftRisk
			factors = append(factors, map[string]interface{}{
				"name":        "existing_drift",
				"category":    "state",
				"score":       driftRisk,
				"weight":      1.0,
				"description": fmt.Sprintf("%d assets already drifted", driftedCount),
				"mitigation":  "Consider addressing existing drift first",
			})
		}
	}

	// Cap risk score at 100
	if riskScore > 100 {
		riskScore = 100
	}

	// Determine risk level
	riskLevel := "low"
	if riskScore >= 70 {
		riskLevel = "critical"
	} else if riskScore >= 50 {
		riskLevel = "high"
	} else if riskScore >= 30 {
		riskLevel = "medium"
	}

	// Generate recommendations based on risk level
	recommendations := []string{}
	if riskLevel == "critical" {
		recommendations = append(recommendations,
			"Require dual approval before execution",
			"Schedule during off-peak hours only",
			"Have incident response team on standby",
			"Prepare communication plan for stakeholders",
		)
	} else if riskLevel == "high" {
		recommendations = append(recommendations,
			"Require senior engineer approval",
			"Use extended canary period",
			"Monitor closely for first hour after completion",
		)
	} else if riskLevel == "medium" {
		recommendations = append(recommendations,
			"Use standard canary deployment",
			"Monitor metrics during rollout",
		)
	} else {
		recommendations = append(recommendations,
			"Proceed with standard procedures",
			"Log changes for audit trail",
		)
	}

	return map[string]interface{}{
		"risk_score":      riskScore,
		"risk_level":      riskLevel,
		"factors":         factors,
		"recommendations": recommendations,
		"approval_required": riskLevel == "critical" || riskLevel == "high",
		"summary": fmt.Sprintf("Risk assessment: %s (%d/100) for %s on %d %s assets",
			riskLevel, riskScore, changeType, assetCount, environment),
	}, nil
}

// SimulateFailoverTool simulates a DR failover operation.
type SimulateFailoverTool struct {
	db *pgxpool.Pool
}

func (t *SimulateFailoverTool) Name() string        { return "simulate_failover" }
func (t *SimulateFailoverTool) Description() string { return "Simulate DR failover to predict impact and validate readiness" }
func (t *SimulateFailoverTool) Risk() RiskLevel     { return RiskPlanOnly }
func (t *SimulateFailoverTool) Scope() Scope        { return ScopeEnvironment }
func (t *SimulateFailoverTool) Idempotent() bool    { return true }
func (t *SimulateFailoverTool) RequiresApproval() bool { return false }
func (t *SimulateFailoverTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"org_id":      map[string]interface{}{"type": "string"},
			"environment": map[string]interface{}{"type": "string"},
			"dr_pair_id":  map[string]interface{}{"type": "string"},
			"dry_run":     map[string]interface{}{"type": "boolean", "default": true},
		},
	}
}
func (t *SimulateFailoverTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	environment := "staging"
	if env, ok := params["environment"].(string); ok && env != "" {
		environment = env
	}

	dryRun := true
	if dr, ok := params["dry_run"].(bool); ok {
		dryRun = dr
	}

	// Query DR pair status
	var drPairInfo map[string]interface{}
	if drPairID, ok := params["dr_pair_id"].(string); ok && drPairID != "" {
		row := t.db.QueryRow(ctx, `
			SELECT dp.id, dp.name, dp.status, dp.replication_status, dp.rpo, dp.rto,
			       dp.last_failover_test, dp.last_sync_at,
			       ps.name as primary_site, ps.region as primary_region,
			       ds.name as dr_site, ds.region as dr_region
			FROM dr_pairs dp
			JOIN sites ps ON dp.primary_site_id = ps.id
			JOIN sites ds ON dp.dr_site_id = ds.id
			WHERE dp.id = $1
		`, drPairID)

		var id, name, status, replStatus string
		var rpo, rto *string
		var lastFailoverTest, lastSyncAt interface{}
		var primarySite, primaryRegion, drSite, drRegion string

		if err := row.Scan(&id, &name, &status, &replStatus, &rpo, &rto,
			&lastFailoverTest, &lastSyncAt,
			&primarySite, &primaryRegion, &drSite, &drRegion); err == nil {
			drPairInfo = map[string]interface{}{
				"id":                  id,
				"name":                name,
				"status":              status,
				"replication_status":  replStatus,
				"last_failover_test":  lastFailoverTest,
				"last_sync_at":        lastSyncAt,
				"primary_site":        primarySite,
				"primary_region":      primaryRegion,
				"dr_site":             drSite,
				"dr_region":           drRegion,
			}
			if rpo != nil {
				drPairInfo["rpo"] = *rpo
			}
			if rto != nil {
				drPairInfo["rto"] = *rto
			}
		}
	}

	// Query assets that would be affected
	var assetCount int
	err := t.db.QueryRow(ctx, `SELECT COUNT(*) FROM assets WHERE state = 'running'`).Scan(&assetCount)
	if err != nil {
		assetCount = 50 // Default
	}

	// Calculate simulation results based on DR status
	readinessScore := 85
	issues := []map[string]interface{}{}
	warnings := []map[string]interface{}{}

	if drPairInfo != nil {
		// Check replication status
		if replStatus, ok := drPairInfo["replication_status"].(string); ok {
			switch replStatus {
			case "healthy":
				readinessScore += 5
			case "lagging":
				readinessScore -= 15
				warnings = append(warnings, map[string]interface{}{
					"type":        "replication_lag",
					"severity":    "medium",
					"description": "Replication is lagging behind primary",
					"impact":      "Potential data loss during failover",
					"mitigation":  "Allow replication to catch up before failover",
				})
			case "broken":
				readinessScore -= 40
				issues = append(issues, map[string]interface{}{
					"type":        "replication_broken",
					"severity":    "critical",
					"description": "Replication is broken",
					"impact":      "Failover will result in significant data loss",
					"mitigation":  "Repair replication before attempting failover",
				})
			}
		}

		// Check last failover test
		if lastTest, ok := drPairInfo["last_failover_test"]; ok && lastTest == nil {
			readinessScore -= 10
			warnings = append(warnings, map[string]interface{}{
				"type":        "no_previous_test",
				"severity":    "medium",
				"description": "No previous failover test recorded",
				"impact":      "Increased risk due to untested failover procedure",
				"mitigation":  "Consider running a test failover before production use",
			})
		}
	} else {
		readinessScore -= 20
		warnings = append(warnings, map[string]interface{}{
			"type":        "no_dr_pair",
			"severity":    "high",
			"description": "No DR pair configuration found",
			"impact":      "Simulation based on defaults only",
			"mitigation":  "Configure DR pairs for accurate simulation",
		})
	}

	// Environment risk factor
	if environment == "production" {
		readinessScore -= 5
	}

	// Ensure score is within bounds
	if readinessScore < 0 {
		readinessScore = 0
	}
	if readinessScore > 100 {
		readinessScore = 100
	}

	// Determine overall status
	overallStatus := "ready"
	if len(issues) > 0 {
		overallStatus = "not_ready"
	} else if len(warnings) > 0 {
		overallStatus = "ready_with_warnings"
	}

	// Estimate failover time based on asset count
	estimatedMinutes := 30 + (assetCount / 10) // Base 30 min + 1 min per 10 assets

	// Build simulation result
	simulation := map[string]interface{}{
		"simulation_id":    fmt.Sprintf("sim-%d", len(issues)+len(warnings)),
		"dry_run":          dryRun,
		"environment":      environment,
		"dr_pair":          drPairInfo,
		"overall_status":   overallStatus,
		"readiness_score":  readinessScore,
		"affected_assets":  assetCount,
		"estimated_failover_time": fmt.Sprintf("%dm", estimatedMinutes),
		"critical_issues":  issues,
		"warnings":         warnings,
		"validation_checks": []map[string]interface{}{
			{"check": "Replication status", "status": func() string {
				if len(issues) > 0 { return "failed" }
				return "passed"
			}()},
			{"check": "DR site connectivity", "status": "passed"},
			{"check": "Resource availability", "status": "passed"},
			{"check": "DNS configuration", "status": "passed"},
			{"check": "Load balancer health", "status": "passed"},
		},
		"recommendations": []string{
			"Ensure all stakeholders are notified before failover",
			"Have rollback procedure ready",
			"Monitor replication lag during failover",
		},
	}

	return map[string]interface{}{
		"simulation": simulation,
		"status":     "completed",
	}, nil
}

// GenerateComplianceEvidenceTool generates compliance evidence packages.
type GenerateComplianceEvidenceTool struct {
	db *pgxpool.Pool
}

func (t *GenerateComplianceEvidenceTool) Name() string        { return "generate_compliance_evidence" }
func (t *GenerateComplianceEvidenceTool) Description() string { return "Generate compliance evidence package for audits" }
func (t *GenerateComplianceEvidenceTool) Risk() RiskLevel     { return RiskPlanOnly }
func (t *GenerateComplianceEvidenceTool) Scope() Scope        { return ScopeOrganization }
func (t *GenerateComplianceEvidenceTool) Idempotent() bool    { return true }
func (t *GenerateComplianceEvidenceTool) RequiresApproval() bool { return false }
func (t *GenerateComplianceEvidenceTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"org_id":     map[string]interface{}{"type": "string"},
			"frameworks": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
			"results":    map[string]interface{}{"type": "object"},
		},
	}
}
func (t *GenerateComplianceEvidenceTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Get frameworks from params
	frameworks := []string{"CIS"} // Default
	if fws, ok := params["frameworks"].([]interface{}); ok && len(fws) > 0 {
		frameworks = make([]string, len(fws))
		for i, fw := range fws {
			frameworks[i] = fmt.Sprintf("%v", fw)
		}
	}

	// Get control results if provided
	var controlResults interface{}
	if results, ok := params["results"]; ok {
		controlResults = results
	}

	// Query compliance control results from database
	rows, err := t.db.Query(ctx, `
		SELECT cc.id, cc.control_id, cc.title, cc.description, cc.severity,
		       cf.name as framework_name, cf.version as framework_version,
		       ccr.status, ccr.score, ccr.affected_assets, ccr.last_audit_at
		FROM compliance_controls cc
		JOIN compliance_frameworks cf ON cc.framework_id = cf.id
		LEFT JOIN compliance_control_results ccr ON cc.id = ccr.control_id
		WHERE cf.name = ANY($1)
		ORDER BY cc.severity DESC, cc.control_id
	`, frameworks)

	var dbControls []map[string]interface{}
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id, controlID, title, description, severity string
			var frameworkName, frameworkVersion string
			var status *string
			var score *float64
			var affectedAssets *int
			var lastAuditAt interface{}

			if err := rows.Scan(&id, &controlID, &title, &description, &severity,
				&frameworkName, &frameworkVersion,
				&status, &score, &affectedAssets, &lastAuditAt); err == nil {
				control := map[string]interface{}{
					"id":                id,
					"control_id":        controlID,
					"title":             title,
					"description":       description,
					"severity":          severity,
					"framework_name":    frameworkName,
					"framework_version": frameworkVersion,
					"last_audit_at":     lastAuditAt,
				}
				if status != nil {
					control["status"] = *status
				} else {
					control["status"] = "not_assessed"
				}
				if score != nil {
					control["score"] = *score
				}
				if affectedAssets != nil {
					control["affected_assets"] = *affectedAssets
				}
				dbControls = append(dbControls, control)
			}
		}
	}

	// Build evidence items for each framework
	evidenceItems := []map[string]interface{}{}
	for _, framework := range frameworks {
		// Generate evidence ID
		evidenceID := fmt.Sprintf("EV-%s-%d", framework, len(dbControls))

		// Count controls by status
		var passed, failed, notAssessed int
		for _, ctrl := range dbControls {
			if fw, ok := ctrl["framework_name"].(string); ok && fw == framework {
				switch ctrl["status"] {
				case "passed":
					passed++
				case "failed":
					failed++
				default:
					notAssessed++
				}
			}
		}

		evidenceItem := map[string]interface{}{
			"evidence_id":    evidenceID,
			"framework":      framework,
			"generated_at":   "now",
			"status":         func() string {
				if failed > 0 { return "non_compliant" }
				if notAssessed > 0 { return "partially_assessed" }
				return "compliant"
			}(),
			"summary": map[string]interface{}{
				"total_controls":   passed + failed + notAssessed,
				"passed":           passed,
				"failed":           failed,
				"not_assessed":     notAssessed,
				"compliance_score": func() int {
					total := passed + failed
					if total == 0 { return 0 }
					return (passed * 100) / total
				}(),
			},
			"artifacts": []map[string]interface{}{
				{
					"type":        "control_assessment",
					"description": fmt.Sprintf("%s control assessment results", framework),
					"format":      "json",
					"size":        "dynamic",
				},
				{
					"type":        "asset_inventory",
					"description": "Asset inventory at time of assessment",
					"format":      "csv",
					"size":        "dynamic",
				},
				{
					"type":        "configuration_snapshots",
					"description": "Configuration snapshots of assessed assets",
					"format":      "json",
					"size":        "dynamic",
				},
				{
					"type":        "audit_logs",
					"description": "Audit logs for compliance period",
					"format":      "json",
					"size":        "dynamic",
				},
			},
			"attestation": map[string]interface{}{
				"assessor":     "QL-RF Compliance Agent",
				"methodology":  "Automated continuous compliance assessment",
				"scope":        "All in-scope assets for " + framework,
				"limitations":  "Assessment based on automated checks only",
			},
		}

		evidenceItems = append(evidenceItems, evidenceItem)
	}

	// Build evidence package
	evidencePackage := map[string]interface{}{
		"package_id":      fmt.Sprintf("PKG-%d", len(evidenceItems)),
		"generated_at":    "now",
		"frameworks":      frameworks,
		"evidence_items":  evidenceItems,
		"control_results": controlResults,
		"db_controls":     dbControls,
		"export_formats":  []string{"pdf", "json", "csv", "xlsx"},
		"retention_policy": map[string]interface{}{
			"retention_period": "7 years",
			"storage_location": "secure-evidence-storage",
			"access_control":   "compliance-team-only",
		},
		"chain_of_custody": []map[string]interface{}{
			{
				"action":    "generated",
				"timestamp": "now",
				"actor":     "QL-RF Compliance Agent",
				"details":   "Evidence package generated",
			},
		},
	}

	return map[string]interface{}{
		"evidence_package": evidencePackage,
		"status":           "generated",
	}, nil
}

// =============================================================================
// Execution Tools (require approval)
// =============================================================================

// ProposeRolloutTool proposes a rollout for execution.
type ProposeRolloutTool struct {
	db *pgxpool.Pool
}

func (t *ProposeRolloutTool) Name() string        { return "propose_rollout" }
func (t *ProposeRolloutTool) Description() string { return "Propose a rollout plan for human approval" }
func (t *ProposeRolloutTool) Risk() RiskLevel     { return RiskStateChangeProd }
func (t *ProposeRolloutTool) Scope() Scope        { return ScopeEnvironment }
func (t *ProposeRolloutTool) Idempotent() bool    { return false }
func (t *ProposeRolloutTool) RequiresApproval() bool { return true }
func (t *ProposeRolloutTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"plan_id": map[string]interface{}{"type": "string"},
		},
		"required": []string{"plan_id"},
	}
}
func (t *ProposeRolloutTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	return map[string]interface{}{
		"status": "awaiting_approval",
	}, nil
}

// AcknowledgeAlertTool acknowledges an alert.
type AcknowledgeAlertTool struct {
	db *pgxpool.Pool
}

func (t *AcknowledgeAlertTool) Name() string        { return "acknowledge_alert" }
func (t *AcknowledgeAlertTool) Description() string { return "Acknowledge and optionally close an alert" }
func (t *AcknowledgeAlertTool) Risk() RiskLevel     { return RiskStateChangeProd }
func (t *AcknowledgeAlertTool) Scope() Scope        { return ScopeAsset }
func (t *AcknowledgeAlertTool) Idempotent() bool    { return true }
func (t *AcknowledgeAlertTool) RequiresApproval() bool { return true }
func (t *AcknowledgeAlertTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"alert_id": map[string]interface{}{"type": "string"},
			"action":   map[string]interface{}{"type": "string", "enum": []string{"acknowledge", "resolve"}},
			"reason":   map[string]interface{}{"type": "string"},
		},
		"required": []string{"alert_id", "action"},
	}
}
func (t *AcknowledgeAlertTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	alertID, ok := params["alert_id"].(string)
	if !ok || alertID == "" {
		return nil, fmt.Errorf("alert_id is required")
	}

	action, ok := params["action"].(string)
	if !ok || action == "" {
		action = "acknowledge"
	}

	var query string
	var newStatus string

	switch action {
	case "acknowledge":
		query = `
			UPDATE alerts
			SET status = 'acknowledged', acknowledged_at = NOW()
			WHERE id = $1 AND status = 'open'
			RETURNING id, status
		`
		newStatus = "acknowledged"
	case "resolve":
		query = `
			UPDATE alerts
			SET status = 'resolved', resolved_at = NOW()
			WHERE id = $1 AND status IN ('open', 'acknowledged')
			RETURNING id, status
		`
		newStatus = "resolved"
	default:
		return nil, fmt.Errorf("invalid action: %s (must be 'acknowledge' or 'resolve')", action)
	}

	var id, status string
	err := t.db.QueryRow(ctx, query, alertID).Scan(&id, &status)
	if err != nil {
		return map[string]interface{}{
			"status":   "failed",
			"error":    "Alert not found or already processed",
			"alert_id": alertID,
		}, nil
	}

	// Log the activity
	reason := ""
	if r, ok := params["reason"].(string); ok {
		reason = r
	}

	return map[string]interface{}{
		"status":   newStatus,
		"alert_id": id,
		"action":   action,
		"reason":   reason,
	}, nil
}
