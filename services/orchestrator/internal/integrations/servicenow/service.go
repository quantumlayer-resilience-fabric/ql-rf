// Package servicenow provides ServiceNow integration service for QL-RF Orchestrator.
package servicenow

import (
	"context"
	"fmt"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/executor"
)

// Service provides high-level ServiceNow integration operations.
type Service struct {
	client  *Client
	log     *logger.Logger
	enabled bool
}

// ServiceConfig holds ServiceNow service configuration.
type ServiceConfig struct {
	Enabled           bool
	InstanceURL       string
	Username          string
	Password          string
	DefaultAssignment string // Default assignment group for changes/incidents
	AutoCreateChanges bool   // Automatically create change requests for tasks
	AutoCreateInc     bool   // Automatically create incidents for failures
	SyncCMDB          bool   // Sync assets to ServiceNow CMDB
}

// NewService creates a new ServiceNow integration service.
func NewService(cfg ServiceConfig, log *logger.Logger) *Service {
	if !cfg.Enabled {
		return &Service{
			enabled: false,
			log:     log.WithComponent("servicenow"),
		}
	}

	client := NewClient(Config{
		InstanceURL: cfg.InstanceURL,
		Username:    cfg.Username,
		Password:    cfg.Password,
	})

	return &Service{
		client:  client,
		log:     log.WithComponent("servicenow"),
		enabled: true,
	}
}

// IsEnabled returns whether ServiceNow integration is enabled.
func (s *Service) IsEnabled() bool {
	return s.enabled
}

// TaskChangeRequest represents the mapping between a QL-RF task and ServiceNow change request.
type TaskChangeRequest struct {
	TaskID        string `json:"task_id"`
	ChangeNumber  string `json:"change_number"`
	ChangeSysID   string `json:"change_sys_id"`
	ChangeState   string `json:"change_state"`
}

// CreateChangeForTask creates a ServiceNow change request for a QL-RF task.
func (s *Service) CreateChangeForTask(ctx context.Context, taskID, taskType, environment, summary, description, riskLevel string) (*TaskChangeRequest, error) {
	if !s.enabled {
		s.log.Debug("ServiceNow integration disabled, skipping change request creation")
		return nil, nil
	}

	// Map QL-RF risk level to ServiceNow risk/priority
	risk, priority := mapRiskLevel(riskLevel)

	cr := ChangeRequest{
		ShortDesc:   fmt.Sprintf("[QL-RF] %s: %s", taskType, summary),
		Description: description,
		Type:        "normal",
		Risk:        risk,
		Priority:    priority,
		Category:    "Infrastructure",
		QLRFID:      taskID,
	}

	result, err := s.client.CreateChangeRequest(ctx, cr)
	if err != nil {
		s.log.Error("failed to create ServiceNow change request", "task_id", taskID, "error", err)
		return nil, err
	}

	s.log.Info("created ServiceNow change request",
		"task_id", taskID,
		"change_number", result.Number,
		"change_sys_id", result.SysID,
	)

	return &TaskChangeRequest{
		TaskID:       taskID,
		ChangeNumber: result.Number,
		ChangeSysID:  result.SysID,
		ChangeState:  result.State,
	}, nil
}

// UpdateChangeState updates the ServiceNow change request state.
func (s *Service) UpdateChangeState(ctx context.Context, sysID, state, workNotes string) error {
	if !s.enabled {
		return nil
	}

	cr := ChangeRequest{
		State:     state,
		WorkNotes: workNotes,
	}

	_, err := s.client.UpdateChangeRequest(ctx, sysID, cr)
	if err != nil {
		s.log.Error("failed to update ServiceNow change request", "sys_id", sysID, "error", err)
		return err
	}

	s.log.Debug("updated ServiceNow change request state", "sys_id", sysID, "state", state)
	return nil
}

// CloseChangeSuccess closes a change request as successful.
func (s *Service) CloseChangeSuccess(ctx context.Context, sysID, notes string) error {
	if !s.enabled {
		return nil
	}

	return s.client.CloseChangeRequest(ctx, sysID, "successful", notes)
}

// CloseChangeFailed closes a change request as failed.
func (s *Service) CloseChangeFailed(ctx context.Context, sysID, notes string) error {
	if !s.enabled {
		return nil
	}

	return s.client.CloseChangeRequest(ctx, sysID, "unsuccessful", notes)
}

// TaskIncident represents the mapping between a QL-RF task and ServiceNow incident.
type TaskIncident struct {
	TaskID        string `json:"task_id"`
	IncidentNumber string `json:"incident_number"`
	IncidentSysID  string `json:"incident_sys_id"`
}

// CreateIncidentForFailure creates a ServiceNow incident for a failed execution.
func (s *Service) CreateIncidentForFailure(ctx context.Context, taskID, taskType string, exec *executor.Execution) (*TaskIncident, error) {
	if !s.enabled {
		s.log.Debug("ServiceNow integration disabled, skipping incident creation")
		return nil, nil
	}

	errorMsg := ""
	if exec != nil {
		errorMsg = exec.Error
	}

	inc := Incident{
		ShortDesc:   fmt.Sprintf("[QL-RF] Execution failed: %s", taskType),
		Description: fmt.Sprintf("QL-RF task execution failed.\n\nTask ID: %s\nTask Type: %s\nError: %s", taskID, taskType, errorMsg),
		Category:    "Infrastructure",
		Priority:    2, // High priority for execution failures
		Impact:      "2",
		Urgency:     "2",
		QLRFID:      taskID,
	}

	result, err := s.client.CreateIncident(ctx, inc)
	if err != nil {
		s.log.Error("failed to create ServiceNow incident", "task_id", taskID, "error", err)
		return nil, err
	}

	s.log.Info("created ServiceNow incident",
		"task_id", taskID,
		"incident_number", result.Number,
		"incident_sys_id", result.SysID,
	)

	return &TaskIncident{
		TaskID:         taskID,
		IncidentNumber: result.Number,
		IncidentSysID:  result.SysID,
	}, nil
}

// SyncAssetToCMDB syncs a QL-RF asset to ServiceNow CMDB.
func (s *Service) SyncAssetToCMDB(ctx context.Context, assetID, name, platform, region, environment, ipAddress, imageVersion, driftStatus string) error {
	if !s.enabled {
		return nil
	}

	ci := CMDBConfigurationItem{
		Name:         name,
		Class:        mapPlatformToCIClass(platform),
		IPAddress:    ipAddress,
		Environment:  environment,
		Platform:     platform,
		Region:       region,
		ImageVersion: imageVersion,
		DriftStatus:  driftStatus,
		QLRFID:       assetID,
	}

	_, err := s.client.UpsertCMDBCI(ctx, ci)
	if err != nil {
		s.log.Error("failed to sync asset to ServiceNow CMDB", "asset_id", assetID, "error", err)
		return err
	}

	s.log.Debug("synced asset to ServiceNow CMDB", "asset_id", assetID, "name", name)
	return nil
}

// mapRiskLevel maps QL-RF risk levels to ServiceNow risk and priority.
func mapRiskLevel(level string) (risk string, priority int) {
	switch level {
	case "state_change_prod":
		return "high", 1
	case "state_change_nonprod":
		return "moderate", 2
	case "plan_only":
		return "low", 3
	case "read_only":
		return "low", 4
	default:
		return "moderate", 3
	}
}

// mapPlatformToCIClass maps QL-RF platform to ServiceNow CI class.
func mapPlatformToCIClass(platform string) string {
	switch platform {
	case "aws":
		return "cmdb_ci_ec2_instance"
	case "azure":
		return "cmdb_ci_azure_vm"
	case "gcp":
		return "cmdb_ci_google_compute_instance"
	case "vsphere":
		return "cmdb_ci_vmware_instance"
	case "k8s":
		return "cmdb_ci_kubernetes_node"
	default:
		return "cmdb_ci_server"
	}
}
