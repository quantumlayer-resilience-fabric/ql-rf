// Package activities defines Temporal activities for task execution.
package activities

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/activity"

	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/temporal/workflows"
)

// =============================================================================
// Patch Campaign Status Activities
// =============================================================================

// UpdatePatchCampaignStatus updates the status of a patch campaign.
func (a *Activities) UpdatePatchCampaignStatus(ctx context.Context, campaignID, status string) error {
	info := activity.GetInfo(ctx)
	a.log.Debug("Updating patch campaign status",
		"campaign_id", campaignID,
		"status", status,
		"activity_id", info.ActivityID,
	)

	query := `
		UPDATE patch_campaigns
		SET status = $1, updated_at = $2
		WHERE id = $3
	`
	_, err := a.db.Exec(ctx, query, status, time.Now().UTC(), campaignID)
	if err != nil {
		a.log.Error("Failed to update campaign status", "campaign_id", campaignID, "error", err)
		return fmt.Errorf("failed to update campaign status: %w", err)
	}

	// Update started_at if transitioning to in_progress
	if status == "in_progress" {
		_, _ = a.db.Exec(ctx, `
			UPDATE patch_campaigns
			SET started_at = $1, updated_at = $1
			WHERE id = $2 AND started_at IS NULL
		`, time.Now().UTC(), campaignID)
	}

	// Update completed_at if transitioning to a terminal state
	if status == "completed" || status == "failed" || status == "cancelled" || status == "rolled_back" {
		_, _ = a.db.Exec(ctx, `
			UPDATE patch_campaigns
			SET completed_at = $1, updated_at = $1
			WHERE id = $2 AND completed_at IS NULL
		`, time.Now().UTC(), campaignID)
	}

	return nil
}

// UpdatePatchPhaseStatus updates the status of a patch campaign phase.
func (a *Activities) UpdatePatchPhaseStatus(ctx context.Context, phaseID, status string) error {
	info := activity.GetInfo(ctx)
	a.log.Debug("Updating patch phase status",
		"phase_id", phaseID,
		"status", status,
		"activity_id", info.ActivityID,
	)

	query := `
		UPDATE patch_campaign_phases
		SET status = $1, updated_at = $2
		WHERE id = $3
	`
	_, err := a.db.Exec(ctx, query, status, time.Now().UTC(), phaseID)
	if err != nil {
		a.log.Error("Failed to update phase status", "phase_id", phaseID, "error", err)
		return fmt.Errorf("failed to update phase status: %w", err)
	}

	// Update started_at if transitioning to in_progress
	if status == "in_progress" {
		_, _ = a.db.Exec(ctx, `
			UPDATE patch_campaign_phases
			SET started_at = $1, updated_at = $1
			WHERE id = $2 AND started_at IS NULL
		`, time.Now().UTC(), phaseID)
	}

	// Update completed_at if transitioning to a terminal state
	if status == "completed" || status == "failed" || status == "rolled_back" || status == "skipped" {
		_, _ = a.db.Exec(ctx, `
			UPDATE patch_campaign_phases
			SET completed_at = $1, updated_at = $1
			WHERE id = $2 AND completed_at IS NULL
		`, time.Now().UTC(), phaseID)
	}

	return nil
}

// =============================================================================
// Patch Phase Execution Activity
// =============================================================================

// ExecutePatchPhase executes patching for all assets in a phase.
func (a *Activities) ExecutePatchPhase(ctx context.Context, input workflows.PatchPhaseExecuteInput) (*workflows.PatchPhaseExecuteOutput, error) {
	info := activity.GetInfo(ctx)
	a.log.Info("Executing patch phase",
		"campaign_id", input.CampaignID,
		"phase_id", input.PhaseID,
		"phase_name", input.PhaseName,
		"asset_count", len(input.AssetIDs),
		"activity_id", info.ActivityID,
	)

	result := &workflows.PatchPhaseExecuteOutput{
		Success:      true,
		TotalAssets:  len(input.AssetIDs),
		AssetResults: make([]workflows.PatchAssetResult, 0, len(input.AssetIDs)),
	}

	for i, assetID := range input.AssetIDs {
		activity.RecordHeartbeat(ctx, fmt.Sprintf("patching asset %d/%d", i+1, len(input.AssetIDs)))

		assetResult := a.executePatchForAsset(ctx, input.CampaignID, input.PhaseID, assetID)
		result.AssetResults = append(result.AssetResults, assetResult)

		if assetResult.Status == "completed" {
			result.SuccessfulPatches++
		} else if assetResult.Status == "failed" {
			result.FailedPatches++
		}

		// Update asset status in database
		a.updatePatchAssetStatus(ctx, input.CampaignID, assetID, assetResult)
	}

	// Update phase counters
	a.updatePhaseCounters(ctx, input.PhaseID, result.SuccessfulPatches, result.FailedPatches)

	// Update campaign counters
	a.updateCampaignCounters(ctx, input.CampaignID)

	a.log.Info("Patch phase completed",
		"phase_id", input.PhaseID,
		"successful", result.SuccessfulPatches,
		"failed", result.FailedPatches,
	)

	return result, nil
}

// executePatchForAsset patches a single asset.
func (a *Activities) executePatchForAsset(ctx context.Context, campaignID, phaseID, assetID string) workflows.PatchAssetResult {
	result := workflows.PatchAssetResult{
		AssetID: assetID,
		Status:  "pending",
	}

	// Get asset information
	var platform, imageRef string
	var assetName string
	query := `
		SELECT platform, COALESCE(image_ref, ''), asset_name
		FROM assets
		WHERE id = $1
	`
	err := a.db.QueryRow(ctx, query, assetID).Scan(&platform, &imageRef, &assetName)
	if err != nil {
		result.Status = "failed"
		result.ErrorMessage = fmt.Sprintf("Failed to get asset info: %v", err)
		return result
	}

	// Determine executor based on platform
	executor := getPatchExecutor(platform)
	result.Executor = executor

	// In a real implementation, this would:
	// 1. Create a snapshot for rollback
	// 2. Execute platform-specific patching (SSM, Azure Update, GCP OS Config, etc.)
	// 3. Wait for patching to complete
	// 4. Verify the patch was applied

	// For now, simulate patching
	a.log.Debug("Patching asset",
		"asset_id", assetID,
		"asset_name", assetName,
		"platform", platform,
		"executor", executor,
	)

	// Simulate execution ID
	result.ExecutionID = fmt.Sprintf("exec-%s", uuid.New().String()[:8])

	// Simulate success (with some failures for realism)
	// In production, this would check actual patch results
	simulateSuccess := true // Most patches succeed

	if simulateSuccess {
		result.Status = "completed"
		result.BeforeVersion = "1.0.0"
		result.AfterVersion = "1.0.1"
	} else {
		result.Status = "failed"
		result.ErrorMessage = "Simulated patch failure"
	}

	return result
}

// updatePatchAssetStatus updates the status of a patched asset in the database.
func (a *Activities) updatePatchAssetStatus(ctx context.Context, campaignID, assetID string, result workflows.PatchAssetResult) {
	now := time.Now().UTC()

	// First check if the asset record exists
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM patch_campaign_assets WHERE campaign_id = $1 AND asset_id = $2)`
	_ = a.db.QueryRow(ctx, checkQuery, campaignID, assetID).Scan(&exists)

	if exists {
		// Update existing record
		query := `
			UPDATE patch_campaign_assets
			SET status = $1, executor = $2, execution_id = $3,
				before_version = $4, after_version = $5, error_message = $6,
				completed_at = $7, updated_at = $7
			WHERE campaign_id = $8 AND asset_id = $9
		`
		_, err := a.db.Exec(ctx, query,
			result.Status, result.Executor, result.ExecutionID,
			result.BeforeVersion, result.AfterVersion, result.ErrorMessage,
			now, campaignID, assetID,
		)
		if err != nil {
			a.log.Warn("Failed to update patch asset status", "asset_id", assetID, "error", err)
		}
	} else {
		// Insert new record
		query := `
			INSERT INTO patch_campaign_assets (
				id, campaign_id, asset_id, status, executor, execution_id,
				before_version, after_version, error_message,
				started_at, completed_at, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10, $10, $10
			)
		`
		_, err := a.db.Exec(ctx, query,
			uuid.New(), campaignID, assetID, result.Status, result.Executor, result.ExecutionID,
			result.BeforeVersion, result.AfterVersion, result.ErrorMessage,
			now,
		)
		if err != nil {
			a.log.Warn("Failed to insert patch asset status", "asset_id", assetID, "error", err)
		}
	}
}

// updatePhaseCounters updates the phase asset counters.
func (a *Activities) updatePhaseCounters(ctx context.Context, phaseID string, successful, failed int) {
	query := `
		UPDATE patch_campaign_phases
		SET completed_assets = $1, failed_assets = $2, updated_at = $3
		WHERE id = $4
	`
	_, err := a.db.Exec(ctx, query, successful, failed, time.Now().UTC(), phaseID)
	if err != nil {
		a.log.Warn("Failed to update phase counters", "phase_id", phaseID, "error", err)
	}
}

// updateCampaignCounters updates the campaign asset counters from all assets.
func (a *Activities) updateCampaignCounters(ctx context.Context, campaignID string) {
	query := `
		UPDATE patch_campaigns SET
			completed_assets = (SELECT COUNT(*) FROM patch_campaign_assets WHERE campaign_id = $1 AND status = 'completed'),
			failed_assets = (SELECT COUNT(*) FROM patch_campaign_assets WHERE campaign_id = $1 AND status = 'failed'),
			in_progress_assets = (SELECT COUNT(*) FROM patch_campaign_assets WHERE campaign_id = $1 AND status = 'in_progress'),
			skipped_assets = (SELECT COUNT(*) FROM patch_campaign_assets WHERE campaign_id = $1 AND status = 'skipped'),
			updated_at = $2
		WHERE id = $1
	`
	_, err := a.db.Exec(ctx, query, campaignID, time.Now().UTC())
	if err != nil {
		a.log.Warn("Failed to update campaign counters", "campaign_id", campaignID, "error", err)
	}
}

// getPatchExecutor returns the appropriate patch executor for a platform.
func getPatchExecutor(platform string) string {
	switch platform {
	case "aws", "AWS":
		return "ssm"
	case "azure", "Azure":
		return "azure_update_mgr"
	case "gcp", "GCP":
		return "gcp_os_config"
	case "k8s", "kubernetes", "K8s":
		return "k8s_rollout"
	case "vsphere", "vSphere":
		return "vsphere_update"
	default:
		return "manual"
	}
}

// =============================================================================
// Health Check Activity
// =============================================================================

// RunHealthChecks runs health checks for assets in a phase.
func (a *Activities) RunHealthChecks(ctx context.Context, input workflows.HealthCheckInput) (*workflows.HealthCheckOutput, error) {
	info := activity.GetInfo(ctx)
	a.log.Info("Running health checks",
		"campaign_id", input.CampaignID,
		"phase_id", input.PhaseID,
		"asset_count", len(input.AssetIDs),
		"activity_id", info.ActivityID,
	)

	result := &workflows.HealthCheckOutput{
		Passed:  true,
		Results: make([]workflows.HealthCheckResultWF, 0),
	}

	failedCount := 0

	for i, assetID := range input.AssetIDs {
		activity.RecordHeartbeat(ctx, fmt.Sprintf("health check %d/%d", i+1, len(input.AssetIDs)))

		// Run health checks for this asset
		checkResults := a.runAssetHealthChecks(ctx, assetID)

		for _, check := range checkResults {
			result.Results = append(result.Results, check)
			if !check.Passed {
				failedCount++
				result.Passed = false
			}
		}

		// Update asset health check status
		a.updateAssetHealthCheck(ctx, input.CampaignID, assetID, checkResults)
	}

	if len(input.AssetIDs) > 0 {
		result.FailureRate = float64(failedCount) / float64(len(input.AssetIDs)*3) // 3 checks per asset
	}

	a.log.Info("Health checks completed",
		"phase_id", input.PhaseID,
		"passed", result.Passed,
		"failure_rate", result.FailureRate,
	)

	return result, nil
}

// runAssetHealthChecks runs health checks for a single asset.
func (a *Activities) runAssetHealthChecks(ctx context.Context, assetID string) []workflows.HealthCheckResultWF {
	results := make([]workflows.HealthCheckResultWF, 0)

	// Get asset information for health checks
	var platform, assetName string
	query := `SELECT platform, asset_name FROM assets WHERE id = $1`
	_ = a.db.QueryRow(ctx, query, assetID).Scan(&platform, &assetName)

	// Run connectivity check
	connectCheck := workflows.HealthCheckResultWF{
		CheckName: fmt.Sprintf("%s-connectivity", assetName),
		Passed:    true, // Simulated
		Message:   "Asset reachable",
		Duration:  100,
	}
	results = append(results, connectCheck)

	// Run service check
	serviceCheck := workflows.HealthCheckResultWF{
		CheckName: fmt.Sprintf("%s-services", assetName),
		Passed:    true, // Simulated
		Message:   "Services running",
		Duration:  200,
	}
	results = append(results, serviceCheck)

	// Run patch verification check
	patchCheck := workflows.HealthCheckResultWF{
		CheckName: fmt.Sprintf("%s-patch-verify", assetName),
		Passed:    true, // Simulated
		Message:   "Patch applied successfully",
		Duration:  150,
	}
	results = append(results, patchCheck)

	return results
}

// updateAssetHealthCheck updates the health check results for an asset.
func (a *Activities) updateAssetHealthCheck(ctx context.Context, campaignID, assetID string, results []workflows.HealthCheckResultWF) {
	allPassed := true
	for _, r := range results {
		if !r.Passed {
			allPassed = false
			break
		}
	}

	resultsJSON, _ := json.Marshal(results)

	query := `
		UPDATE patch_campaign_assets
		SET health_check_passed = $1, health_check_results = $2,
			health_check_attempts = health_check_attempts + 1, updated_at = $3
		WHERE campaign_id = $4 AND asset_id = $5
	`
	_, err := a.db.Exec(ctx, query, allPassed, resultsJSON, time.Now().UTC(), campaignID, assetID)
	if err != nil {
		a.log.Warn("Failed to update asset health check", "asset_id", assetID, "error", err)
	}
}

// =============================================================================
// Rollback Activity
// =============================================================================

// ExecuteRollback performs rollback for assets.
func (a *Activities) ExecuteRollback(ctx context.Context, input workflows.RollbackInput) (*workflows.RollbackOutput, error) {
	info := activity.GetInfo(ctx)
	a.log.Info("Executing rollback",
		"campaign_id", input.CampaignID,
		"phase_id", input.PhaseID,
		"scope", input.Scope,
		"trigger", input.TriggerType,
		"asset_count", len(input.AssetIDs),
		"activity_id", info.ActivityID,
	)

	result := &workflows.RollbackOutput{
		Success:        true,
		FailedAssetIDs: make([]string, 0),
	}

	// Create rollback record
	rollbackID := uuid.New()
	_, err := a.db.Exec(ctx, `
		INSERT INTO patch_rollbacks (
			id, campaign_id, phase_id, trigger_type, trigger_reason,
			rollback_scope, asset_ids, status, total_assets, started_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, 'in_progress', $8, $9, $9)
	`,
		rollbackID, input.CampaignID, input.PhaseID, input.TriggerType, input.Reason,
		input.Scope, input.AssetIDs, len(input.AssetIDs), time.Now().UTC(),
	)
	if err != nil {
		a.log.Warn("Failed to create rollback record", "error", err)
	}

	// Execute rollback for each asset
	for i, assetID := range input.AssetIDs {
		activity.RecordHeartbeat(ctx, fmt.Sprintf("rolling back asset %d/%d", i+1, len(input.AssetIDs)))

		success := a.rollbackAsset(ctx, input.CampaignID, assetID)
		if success {
			result.RolledBackCount++
		} else {
			result.FailedCount++
			result.FailedAssetIDs = append(result.FailedAssetIDs, assetID)
		}
	}

	// Update rollback record
	rollbackStatus := "completed"
	if result.FailedCount > 0 {
		if result.RolledBackCount > 0 {
			rollbackStatus = "partial"
		} else {
			rollbackStatus = "failed"
			result.Success = false
		}
	}

	_, _ = a.db.Exec(ctx, `
		UPDATE patch_rollbacks
		SET status = $1, successful_rollbacks = $2, failed_rollbacks = $3, completed_at = $4
		WHERE id = $5
	`, rollbackStatus, result.RolledBackCount, result.FailedCount, time.Now().UTC(), rollbackID)

	a.log.Info("Rollback completed",
		"campaign_id", input.CampaignID,
		"rolled_back", result.RolledBackCount,
		"failed", result.FailedCount,
	)

	return result, nil
}

// rollbackAsset rolls back a single asset.
func (a *Activities) rollbackAsset(ctx context.Context, campaignID, assetID string) bool {
	// Get rollback info
	var snapshotID *string
	query := `
		SELECT rollback_snapshot_id
		FROM patch_campaign_assets
		WHERE campaign_id = $1 AND asset_id = $2
	`
	_ = a.db.QueryRow(ctx, query, campaignID, assetID).Scan(&snapshotID)

	// In a real implementation, this would:
	// 1. Restore from snapshot if available
	// 2. Or reinstall previous package versions
	// 3. Verify rollback was successful

	// Simulate rollback
	a.log.Debug("Rolling back asset", "asset_id", assetID, "snapshot_id", snapshotID)

	// Update asset status
	now := time.Now().UTC()
	_, err := a.db.Exec(ctx, `
		UPDATE patch_campaign_assets
		SET status = 'rolled_back', rolled_back_at = $1, rollback_reason = 'Automatic rollback',
			updated_at = $1
		WHERE campaign_id = $2 AND asset_id = $3
	`, now, campaignID, assetID)

	return err == nil
}

// =============================================================================
// Notification Activity
// =============================================================================

// NotifyPatchCampaignEvent sends notifications for patch campaign events.
func (a *Activities) NotifyPatchCampaignEvent(ctx context.Context, notification workflows.PatchCampaignNotification) error {
	info := activity.GetInfo(ctx)
	a.log.Info("Sending patch campaign notification",
		"campaign_id", notification.CampaignID,
		"event_type", notification.EventType,
		"activity_id", info.ActivityID,
	)

	// Format message based on event type
	var message string
	switch notification.EventType {
	case "started":
		message = fmt.Sprintf("Patch campaign '%s' has started. Patching %d assets.",
			notification.CampaignName, notification.TotalAssets)
	case "phase_complete":
		message = fmt.Sprintf("Patch campaign '%s' - Phase '%s' completed. Successful: %d, Failed: %d",
			notification.CampaignName, notification.PhaseName, notification.Successful, notification.Failed)
	case "completed":
		message = fmt.Sprintf("Patch campaign '%s' completed. Total: %d, Successful: %d, Failed: %d",
			notification.CampaignName, notification.TotalAssets, notification.Successful, notification.Failed)
	case "failed":
		message = fmt.Sprintf("Patch campaign '%s' failed at phase '%s': %s",
			notification.CampaignName, notification.PhaseName, notification.Message)
	case "rollback":
		message = fmt.Sprintf("Patch campaign '%s' rollback triggered: %s",
			notification.CampaignName, notification.Message)
	default:
		message = notification.Message
	}

	// TODO: Send actual notifications via notifier service
	a.log.Info("Patch campaign notification",
		"campaign_id", notification.CampaignID,
		"org_id", notification.OrgID,
		"message", message,
	)

	return nil
}

// =============================================================================
// Result Storage Activity
// =============================================================================

// StorePatchCampaignResult stores the final campaign result.
func (a *Activities) StorePatchCampaignResult(ctx context.Context, result *workflows.PatchCampaignWorkflowResult) error {
	info := activity.GetInfo(ctx)
	a.log.Info("Storing patch campaign result",
		"campaign_id", result.CampaignID,
		"status", result.Status,
		"activity_id", info.ActivityID,
	)

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	// Store as audit trail entry
	query := `
		INSERT INTO ai_tool_invocations (task_id, tool_name, risk_level, parameters, result, created_at)
		VALUES ($1, 'patch_campaign_result', 'execute', $2, $3, $4)
	`
	params := map[string]interface{}{
		"campaign_id":    result.CampaignID,
		"status":         result.Status,
		"duration":       result.Duration.String(),
		"total_assets":   result.TotalAssets,
		"successful":     result.SuccessfulPatches,
		"failed":         result.FailedPatches,
		"success_rate":   result.SuccessRate,
	}
	paramsJSON, _ := json.Marshal(params)

	_, err = a.db.Exec(ctx, query,
		result.CampaignID,
		paramsJSON,
		resultJSON,
		time.Now().UTC(),
	)
	if err != nil {
		a.log.Error("Failed to store patch campaign result", "campaign_id", result.CampaignID, "error", err)
		// Don't fail the campaign if we can't store results
	}

	return nil
}

// =============================================================================
// Evidence Recording Activity
// =============================================================================

// RecordVulnerabilityEvidence records evidence for compliance purposes.
func (a *Activities) RecordVulnerabilityEvidence(ctx context.Context, params map[string]interface{}) error {
	info := activity.GetInfo(ctx)
	a.log.Info("Recording vulnerability evidence",
		"campaign_id", params["campaign_id"],
		"event_type", params["event_type"],
		"activity_id", info.ActivityID,
	)

	evidenceJSON, _ := json.Marshal(params["evidence"])
	cveAlertIDs, _ := params["cve_alert_ids"].([]string)

	// Record evidence for each CVE alert
	for _, alertID := range cveAlertIDs {
		query := `
			INSERT INTO vulnerability_evidence (
				id, org_id, cve_alert_id, evidence_type, evidence_data,
				collected_at, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $6)
		`
		_, err := a.db.Exec(ctx, query,
			uuid.New(),
			params["org_id"],
			alertID,
			params["event_type"],
			evidenceJSON,
			time.Now().UTC(),
		)
		if err != nil {
			a.log.Warn("Failed to record vulnerability evidence", "alert_id", alertID, "error", err)
		}
	}

	return nil
}
