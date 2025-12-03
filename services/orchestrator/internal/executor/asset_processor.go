// Package executor implements the plan execution engine.
package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/pkg/models"
)

// AssetAction represents an action to perform on an asset.
type AssetAction string

const (
	ActionReimage   AssetAction = "reimage"
	ActionReboot    AssetAction = "reboot"
	ActionTerminate AssetAction = "terminate"
	ActionPatch     AssetAction = "patch"
	ActionUpdate    AssetAction = "update"
	ActionValidate  AssetAction = "validate"
)

// AssetProcessorResult represents the result of processing an asset.
type AssetProcessorResult struct {
	AssetID     string        `json:"asset_id"`
	AssetName   string        `json:"asset_name"`
	Action      AssetAction   `json:"action"`
	Success     bool          `json:"success"`
	Error       string        `json:"error,omitempty"`
	Output      string        `json:"output,omitempty"`
	Duration    time.Duration `json:"duration"`
	Timestamp   time.Time     `json:"timestamp"`
	NeedsReboot bool          `json:"needs_reboot,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// PlatformClient is the interface for platform-specific operations.
type PlatformClient interface {
	// ReimageInstance reimages an instance with a new AMI/image.
	ReimageInstance(ctx context.Context, instanceID, imageID string) error

	// RebootInstance reboots an instance.
	RebootInstance(ctx context.Context, instanceID string) error

	// TerminateInstance terminates an instance.
	TerminateInstance(ctx context.Context, instanceID string) error

	// GetInstanceStatus gets the current status of an instance.
	GetInstanceStatus(ctx context.Context, instanceID string) (string, error)

	// WaitForInstanceState waits for an instance to reach a specific state.
	WaitForInstanceState(ctx context.Context, instanceID, targetState string, timeout time.Duration) error

	// ApplyPatches applies patches to an instance using platform-native tooling (SSM, Azure Update, etc).
	ApplyPatches(ctx context.Context, instanceID string, params map[string]interface{}) error

	// GetPatchStatus retrieves patch compliance status for an instance.
	GetPatchStatus(ctx context.Context, instanceID string) (string, error)

	// GetPatchComplianceData retrieves detailed patch compliance data.
	GetPatchComplianceData(ctx context.Context, instanceID string) (interface{}, error)
}

// AssetProcessor processes assets during execution.
type AssetProcessor struct {
	db             *pgxpool.Pool
	log            *logger.Logger
	platformClients map[models.Platform]PlatformClient
	healthChecker  *HealthChecker
}

// NewAssetProcessor creates a new asset processor.
func NewAssetProcessor(db *pgxpool.Pool, log *logger.Logger) *AssetProcessor {
	return &AssetProcessor{
		db:             db,
		log:            log.WithComponent("asset-processor"),
		platformClients: make(map[models.Platform]PlatformClient),
		healthChecker:  NewHealthChecker(log),
	}
}

// RegisterPlatformClient registers a platform client for asset operations.
func (p *AssetProcessor) RegisterPlatformClient(platform models.Platform, client PlatformClient) {
	p.platformClients[platform] = client
	p.log.Info("registered platform client", "platform", platform)
}

// ProcessAsset processes a single asset with the specified action.
func (p *AssetProcessor) ProcessAsset(ctx context.Context, asset *AssetInfo, action AssetAction, params map[string]interface{}) (*AssetProcessorResult, error) {
	result := &AssetProcessorResult{
		AssetID:   asset.ID,
		AssetName: asset.Name,
		Action:    action,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	start := time.Now()
	defer func() {
		result.Duration = time.Since(start)
	}()

	p.log.Info("processing asset",
		"asset_id", asset.ID,
		"asset_name", asset.Name,
		"action", action,
		"platform", asset.Platform,
	)

	var err error
	switch action {
	case ActionReimage:
		err = p.reimageAsset(ctx, asset, params, result)
	case ActionReboot:
		err = p.rebootAsset(ctx, asset, result)
	case ActionTerminate:
		err = p.terminateAsset(ctx, asset, result)
	case ActionPatch:
		err = p.patchAsset(ctx, asset, params, result)
	case ActionUpdate:
		err = p.updateAsset(ctx, asset, params, result)
	case ActionValidate:
		err = p.validateAsset(ctx, asset, params, result)
	default:
		err = fmt.Errorf("unsupported action: %s", action)
	}

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		p.log.Error("asset processing failed",
			"asset_id", asset.ID,
			"action", action,
			"error", err,
			"duration", result.Duration,
		)
		return result, err
	}

	result.Success = true
	p.log.Info("asset processing completed",
		"asset_id", asset.ID,
		"action", action,
		"duration", result.Duration,
	)

	// Record the operation in the database
	if err := p.recordOperation(ctx, result); err != nil {
		p.log.Warn("failed to record operation", "error", err)
	}

	return result, nil
}

// AssetInfo contains information about an asset to process.
type AssetInfo struct {
	ID           string
	Name         string
	Platform     models.Platform
	Region       string
	InstanceID   string
	CurrentImage string
	TargetImage  string
	Tags         map[string]string
}

// reimageAsset reimages an asset with a new image.
func (p *AssetProcessor) reimageAsset(ctx context.Context, asset *AssetInfo, params map[string]interface{}, result *AssetProcessorResult) error {
	targetImage, ok := params["target_image"].(string)
	if !ok || targetImage == "" {
		targetImage = asset.TargetImage
	}
	if targetImage == "" {
		return fmt.Errorf("target image is required for reimage operation")
	}

	client, ok := p.platformClients[asset.Platform]
	if !ok {
		return fmt.Errorf("no platform client registered for %s", asset.Platform)
	}

	// Store the target image in result metadata
	result.Metadata["target_image"] = targetImage
	result.Metadata["current_image"] = asset.CurrentImage

	// Perform the reimage operation
	if err := client.ReimageInstance(ctx, asset.InstanceID, targetImage); err != nil {
		return fmt.Errorf("failed to reimage instance: %w", err)
	}

	// Wait for instance to be running again
	if err := client.WaitForInstanceState(ctx, asset.InstanceID, "running", 10*time.Minute); err != nil {
		return fmt.Errorf("instance did not return to running state: %w", err)
	}

	// Update asset record in database
	if err := p.updateAssetImage(ctx, asset.ID, targetImage); err != nil {
		p.log.Warn("failed to update asset image in database", "error", err)
	}

	result.Output = fmt.Sprintf("Successfully reimaged %s from %s to %s", asset.InstanceID, asset.CurrentImage, targetImage)
	return nil
}

// rebootAsset reboots an asset.
func (p *AssetProcessor) rebootAsset(ctx context.Context, asset *AssetInfo, result *AssetProcessorResult) error {
	client, ok := p.platformClients[asset.Platform]
	if !ok {
		return fmt.Errorf("no platform client registered for %s", asset.Platform)
	}

	if err := client.RebootInstance(ctx, asset.InstanceID); err != nil {
		return fmt.Errorf("failed to reboot instance: %w", err)
	}

	// Wait for instance to be running again
	if err := client.WaitForInstanceState(ctx, asset.InstanceID, "running", 5*time.Minute); err != nil {
		return fmt.Errorf("instance did not return to running state: %w", err)
	}

	result.Output = fmt.Sprintf("Successfully rebooted %s", asset.InstanceID)
	return nil
}

// terminateAsset terminates an asset.
func (p *AssetProcessor) terminateAsset(ctx context.Context, asset *AssetInfo, result *AssetProcessorResult) error {
	client, ok := p.platformClients[asset.Platform]
	if !ok {
		return fmt.Errorf("no platform client registered for %s", asset.Platform)
	}

	if err := client.TerminateInstance(ctx, asset.InstanceID); err != nil {
		return fmt.Errorf("failed to terminate instance: %w", err)
	}

	// Wait for instance to be terminated
	if err := client.WaitForInstanceState(ctx, asset.InstanceID, "terminated", 5*time.Minute); err != nil {
		// Log but don't fail - termination was initiated
		p.log.Warn("could not verify termination", "error", err)
	}

	// Mark asset as terminated in database
	if err := p.updateAssetState(ctx, asset.ID, "terminated"); err != nil {
		p.log.Warn("failed to update asset state in database", "error", err)
	}

	result.Output = fmt.Sprintf("Successfully terminated %s", asset.InstanceID)
	return nil
}

// patchAsset applies patches to an asset via platform-native tooling (AWS SSM, Azure Update Management, etc).
func (p *AssetProcessor) patchAsset(ctx context.Context, asset *AssetInfo, params map[string]interface{}, result *AssetProcessorResult) error {
	// Get patch operation parameters
	patchGroup, _ := params["patch_group"].(string)
	baselineID, _ := params["baseline_id"].(string)
	operation, _ := params["operation"].(string)
	if operation == "" {
		operation = "Install" // Default to install
	}
	rebootIfNeeded := true
	if rb, ok := params["reboot_if_needed"].(bool); ok {
		rebootIfNeeded = rb
	}
	synchronous := true
	if sync, ok := params["synchronous"].(bool); ok {
		synchronous = sync
	}

	client, ok := p.platformClients[asset.Platform]
	if !ok {
		return fmt.Errorf("no platform client registered for %s", asset.Platform)
	}

	// Verify instance is running
	status, err := client.GetInstanceStatus(ctx, asset.InstanceID)
	if err != nil {
		return fmt.Errorf("failed to get instance status: %w", err)
	}
	if status != "running" {
		return fmt.Errorf("instance is not running (current state: %s)", status)
	}

	p.log.Info("starting patch operation",
		"instance_id", asset.InstanceID,
		"platform", asset.Platform,
		"operation", operation,
		"patch_group", patchGroup,
		"baseline_id", baselineID,
	)

	// Build patch parameters for the platform client
	patchParams := map[string]interface{}{
		"operation":        operation,
		"reboot_if_needed": rebootIfNeeded,
		"synchronous":      synchronous,
		"region":           asset.Region,
	}
	if baselineID != "" {
		patchParams["baseline_override"] = baselineID
	}

	// Apply patches using platform-native tooling
	if err := client.ApplyPatches(ctx, asset.InstanceID, patchParams); err != nil {
		return fmt.Errorf("patch operation failed: %w", err)
	}

	// Get post-patch compliance status
	complianceStatus, err := client.GetPatchStatus(ctx, asset.InstanceID)
	if err != nil {
		p.log.Warn("failed to get post-patch compliance status", "error", err)
		complianceStatus = "unknown"
	}

	// Get detailed compliance data if available
	complianceData, err := client.GetPatchComplianceData(ctx, asset.InstanceID)
	if err == nil && complianceData != nil {
		result.Metadata["compliance_data"] = complianceData
	}

	result.Metadata["patch_group"] = patchGroup
	result.Metadata["baseline_id"] = baselineID
	result.Metadata["operation"] = operation
	result.Metadata["compliance_status"] = complianceStatus
	result.NeedsReboot = rebootIfNeeded && operation == "Install"
	result.Output = fmt.Sprintf("Patch operation completed for %s (operation: %s, compliance: %s)", asset.InstanceID, operation, complianceStatus)

	return nil
}

// updateAsset updates asset configuration.
func (p *AssetProcessor) updateAsset(ctx context.Context, asset *AssetInfo, params map[string]interface{}, result *AssetProcessorResult) error {
	updates := make(map[string]interface{})

	// Extract update parameters
	if tags, ok := params["tags"].(map[string]interface{}); ok {
		updates["tags"] = tags
	}
	if state, ok := params["state"].(string); ok {
		updates["state"] = state
	}

	// Update the asset in database
	if len(updates) > 0 {
		if err := p.updateAssetMetadata(ctx, asset.ID, updates); err != nil {
			return fmt.Errorf("failed to update asset metadata: %w", err)
		}
	}

	result.Output = fmt.Sprintf("Updated asset %s with %d fields", asset.ID, len(updates))
	return nil
}

// validateAsset validates an asset's current state against expected state.
func (p *AssetProcessor) validateAsset(ctx context.Context, asset *AssetInfo, params map[string]interface{}, result *AssetProcessorResult) error {
	client, ok := p.platformClients[asset.Platform]
	if !ok {
		return fmt.Errorf("no platform client registered for %s", asset.Platform)
	}

	// Check instance is running
	status, err := client.GetInstanceStatus(ctx, asset.InstanceID)
	if err != nil {
		return fmt.Errorf("failed to get instance status: %w", err)
	}
	result.Metadata["status"] = status

	if status != "running" {
		return fmt.Errorf("instance is not running: %s", status)
	}

	// Run health checks if specified
	if healthChecks, ok := params["health_checks"].([]interface{}); ok {
		for _, hcRaw := range healthChecks {
			hcMap, ok := hcRaw.(map[string]interface{})
			if !ok {
				continue
			}

			hc := &HealthCheck{
				Name:   getStringParam(hcMap, "name"),
				Type:   getStringParam(hcMap, "type"),
				Target: getStringParam(hcMap, "target"),
				Expected: getStringParam(hcMap, "expected"),
			}

			checkResult, err := p.healthChecker.CheckWithRetry(ctx, hc)
			if err != nil {
				return fmt.Errorf("health check '%s' failed: %w", hc.Name, err)
			}
			result.Metadata[fmt.Sprintf("healthcheck_%s", hc.Name)] = checkResult.Success
		}
	}

	result.Output = fmt.Sprintf("Asset %s validated successfully (status: %s)", asset.InstanceID, status)
	return nil
}

// Helper functions for database operations

func (p *AssetProcessor) updateAssetImage(ctx context.Context, assetID, imageRef string) error {
	if p.db == nil {
		return nil
	}

	query := `
		UPDATE assets
		SET image_ref = $1, updated_at = NOW()
		WHERE id = $2
	`
	_, err := p.db.Exec(ctx, query, imageRef, assetID)
	return err
}

func (p *AssetProcessor) updateAssetState(ctx context.Context, assetID, state string) error {
	if p.db == nil {
		return nil
	}

	query := `
		UPDATE assets
		SET state = $1, updated_at = NOW()
		WHERE id = $2
	`
	_, err := p.db.Exec(ctx, query, state, assetID)
	return err
}

func (p *AssetProcessor) updateAssetMetadata(ctx context.Context, assetID string, updates map[string]interface{}) error {
	if p.db == nil {
		return nil
	}

	// Update tags if provided
	if tags, ok := updates["tags"].(map[string]interface{}); ok {
		tagsJSON, err := json.Marshal(tags)
		if err != nil {
			return fmt.Errorf("failed to marshal tags: %w", err)
		}
		query := `
			UPDATE assets
			SET tags = $1, updated_at = NOW()
			WHERE id = $2
		`
		if _, err := p.db.Exec(ctx, query, tagsJSON, assetID); err != nil {
			return err
		}
	}

	// Update state if provided
	if state, ok := updates["state"].(string); ok {
		if err := p.updateAssetState(ctx, assetID, state); err != nil {
			return err
		}
	}

	return nil
}

func (p *AssetProcessor) recordOperation(ctx context.Context, result *AssetProcessorResult) error {
	if p.db == nil {
		return nil
	}

	metadata, _ := json.Marshal(result.Metadata)

	query := `
		INSERT INTO activities (id, org_id, type, action, detail, asset_id, timestamp)
		SELECT $1, a.org_id, 'execution', $2, $3, a.id, $4
		FROM assets a WHERE a.id = $5
	`

	detail := fmt.Sprintf("%s: %s", result.Action, result.Output)
	if result.Error != "" {
		detail = fmt.Sprintf("%s: ERROR - %s", result.Action, result.Error)
	}

	_, err := p.db.Exec(ctx, query,
		uuid.New(),
		string(result.Action),
		detail,
		result.Timestamp,
		result.AssetID,
	)

	// Don't fail if activity logging fails, but log it
	if err != nil {
		p.log.Debug("failed to record activity", "error", err, "metadata", string(metadata))
	}

	return err
}

// getStringParam safely gets a string parameter from a map.
func getStringParam(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
