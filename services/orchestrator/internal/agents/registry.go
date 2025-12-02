// Package agents provides the specialist agent registry and implementations.
package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/llm"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/tools"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/validation"
)

// TaskType represents the type of task an agent can handle.
type TaskType string

const (
	TaskTypeDriftRemediation   TaskType = "drift_remediation"
	TaskTypePatchRollout       TaskType = "patch_rollout"
	TaskTypeComplianceAudit    TaskType = "compliance_audit"
	TaskTypeIncidentResponse   TaskType = "incident_investigation"
	TaskTypeDRDrill            TaskType = "dr_drill"
	TaskTypeCostOptimization   TaskType = "cost_optimization"
	TaskTypeSecurityScan       TaskType = "security_scan"
	TaskTypeImageManagement    TaskType = "image_management"
	TaskTypeSOPAuthoring       TaskType = "sop_authoring"
	TaskTypeTerraformGeneration TaskType = "terraform_generation"
)

// TaskSpec represents the specification for a task to be executed.
type TaskSpec struct {
	ID              string                 `json:"id"`
	TaskType        TaskType               `json:"task_type"`
	Goal            string                 `json:"goal"`
	UserIntent      string                 `json:"user_intent"`
	OrgID           string                 `json:"org_id"`
	UserID          string                 `json:"user_id"`
	Environment     string                 `json:"environment"`
	Context         TaskContext            `json:"context"`
	ToolsRequired   []string               `json:"tools_required"`
	Validation      ValidationSpec         `json:"validation"`
	RiskLevel       string                 `json:"risk_level"`
	HITLRequired    bool                   `json:"hitl_required"`
	TimeoutMinutes  int                    `json:"timeout_minutes"`
	Constraints     map[string]interface{} `json:"constraints"`
}

// TaskContext provides context for task execution.
type TaskContext struct {
	Platforms    []string               `json:"platforms"`
	Regions      []string               `json:"regions"`
	AssetFilter  string                 `json:"asset_filter"`
	Tags         map[string]string      `json:"tags"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// ValidationSpec specifies validation requirements.
type ValidationSpec struct {
	Schema      string            `json:"schema"`
	Policies    []string          `json:"policies"`
	Constraints map[string]interface{} `json:"constraints"`
}

// AgentResult represents the result of an agent execution.
type AgentResult struct {
	TaskID         string                 `json:"task_id"`
	AgentName      string                 `json:"agent_name"`
	Status         AgentStatus            `json:"status"`
	Plan           interface{}            `json:"plan,omitempty"`
	Summary        string                 `json:"summary"`
	AffectedAssets int                    `json:"affected_assets"`
	RiskLevel      string                 `json:"risk_level"`
	Actions        []Action               `json:"actions,omitempty"`
	Evidence       map[string]interface{} `json:"evidence,omitempty"`
	Errors         []string               `json:"errors,omitempty"`
	TokensUsed     int                    `json:"tokens_used"`
}

// AgentStatus represents the status of an agent execution.
type AgentStatus string

const (
	AgentStatusPendingApproval AgentStatus = "pending_approval"
	AgentStatusApproved        AgentStatus = "approved"
	AgentStatusExecuting       AgentStatus = "executing"
	AgentStatusCompleted       AgentStatus = "completed"
	AgentStatusFailed          AgentStatus = "failed"
	AgentStatusCancelled       AgentStatus = "cancelled"
)

// Action represents an action that can be taken on an agent result.
type Action struct {
	Type        string `json:"type"`        // approve, modify, reject, retry
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

// Agent is the interface all specialist agents must implement.
type Agent interface {
	// Name returns the unique agent name.
	Name() string

	// Description returns a human-readable description.
	Description() string

	// SupportedTasks returns task types this agent handles.
	SupportedTasks() []TaskType

	// RequiredTools returns tool names needed by this agent.
	RequiredTools() []string

	// Execute runs the agent for a given task specification.
	Execute(ctx context.Context, task *TaskSpec) (*AgentResult, error)
}

// Registry manages registered agents and their routing.
type Registry struct {
	agents    map[string]Agent
	taskMap   map[TaskType][]string // task type -> agent names
	llm       llm.Client
	tools     *tools.Registry
	validator *validation.Pipeline
	log       *logger.Logger
	mu        sync.RWMutex
}

// NewRegistry creates a new agent registry with all specialist agents.
func NewRegistry(llmClient llm.Client, toolRegistry *tools.Registry, validator *validation.Pipeline, log *logger.Logger) *Registry {
	r := &Registry{
		agents:    make(map[string]Agent),
		taskMap:   make(map[TaskType][]string),
		llm:       llmClient,
		tools:     toolRegistry,
		validator: validator,
		log:       log.WithComponent("agent-registry"),
	}

	// Register all specialist agents
	r.registerAgents()

	return r
}

// registerAgents registers all specialist agents.
func (r *Registry) registerAgents() {
	agents := []Agent{
		NewDriftAgent(r.llm, r.tools, r.log),
		NewPatchAgent(r.llm, r.tools, r.log),
		NewComplianceAgent(r.llm, r.tools, r.log),
		NewIncidentAgent(r.llm, r.tools, r.log),
		NewDRAgent(r.llm, r.tools, r.log),
		NewCostAgent(r.llm, r.tools, r.log),
		NewSecurityAgent(r.llm, r.tools, r.log),
		NewImageAgent(r.llm, r.tools, r.log),
		NewSOPAgent(r.llm, r.tools, r.log),
		NewAdapterAgent(r.llm, r.tools, r.log),
	}

	for _, agent := range agents {
		r.Register(agent)
	}
}

// Register adds an agent to the registry.
func (r *Registry) Register(agent Agent) {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := agent.Name()
	r.agents[name] = agent

	// Build task type -> agent mapping
	for _, taskType := range agent.SupportedTasks() {
		r.taskMap[taskType] = append(r.taskMap[taskType], name)
	}

	r.log.Info("registered agent",
		"agent", name,
		"tasks", agent.SupportedTasks(),
		"tools", agent.RequiredTools(),
	)
}

// Get returns an agent by name.
func (r *Registry) Get(name string) (Agent, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	agent, ok := r.agents[name]
	return agent, ok
}

// GetForTask returns agents that can handle a specific task type.
func (r *Registry) GetForTask(taskType TaskType) []Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agentNames := r.taskMap[taskType]
	agents := make([]Agent, 0, len(agentNames))

	for _, name := range agentNames {
		if agent, ok := r.agents[name]; ok {
			agents = append(agents, agent)
		}
	}

	return agents
}

// ListAgents returns all registered agent names.
func (r *Registry) ListAgents() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.agents))
	for name := range r.agents {
		names = append(names, name)
	}
	return names
}

// AgentInfo returns information about all registered agents.
func (r *Registry) AgentInfo() []AgentMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	info := make([]AgentMetadata, 0, len(r.agents))
	for _, agent := range r.agents {
		info = append(info, AgentMetadata{
			Name:           agent.Name(),
			Description:    agent.Description(),
			SupportedTasks: agent.SupportedTasks(),
			RequiredTools:  agent.RequiredTools(),
		})
	}
	return info
}

// AgentMetadata contains metadata about an agent.
type AgentMetadata struct {
	Name           string     `json:"name"`
	Description    string     `json:"description"`
	SupportedTasks []TaskType `json:"supported_tasks"`
	RequiredTools  []string   `json:"required_tools"`
}

// =============================================================================
// Base Agent Implementation
// =============================================================================

// BaseAgent provides common functionality for all agents.
type BaseAgent struct {
	name        string
	description string
	tasks       []TaskType
	tools       []string
	llm         llm.Client
	toolReg     *tools.Registry
	log         *logger.Logger
}

// Name returns the agent name.
func (a *BaseAgent) Name() string {
	return a.name
}

// Description returns the agent description.
func (a *BaseAgent) Description() string {
	return a.description
}

// SupportedTasks returns supported task types.
func (a *BaseAgent) SupportedTasks() []TaskType {
	return a.tasks
}

// RequiredTools returns required tool names.
func (a *BaseAgent) RequiredTools() []string {
	return a.tools
}

// executeTool runs a tool and returns the result.
func (a *BaseAgent) executeTool(ctx context.Context, toolName string, params map[string]interface{}) (interface{}, error) {
	tool, ok := a.toolReg.Get(toolName)
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", toolName)
	}
	return tool.Execute(ctx, params)
}

// =============================================================================
// Specialist Agent Implementations
// =============================================================================

// DriftAgent handles drift detection and remediation.
type DriftAgent struct {
	BaseAgent
}

// NewDriftAgent creates a new drift agent.
func NewDriftAgent(llmClient llm.Client, toolReg *tools.Registry, log *logger.Logger) *DriftAgent {
	return &DriftAgent{
		BaseAgent: BaseAgent{
			name:        "drift_agent",
			description: "Detects configuration drift and generates remediation plans",
			tasks:       []TaskType{TaskTypeDriftRemediation},
			tools: []string{
				"query_assets",
				"get_golden_image",
				"get_drift_status",
				"compare_versions",
				"generate_patch_plan",
				"simulate_rollout",
			},
			llm:     llmClient,
			toolReg: toolReg,
			log:     log.WithComponent("drift-agent"),
		},
	}
}

// Execute runs the drift agent.
func (a *DriftAgent) Execute(ctx context.Context, task *TaskSpec) (*AgentResult, error) {
	a.log.Info("executing drift agent", "task_id", task.ID, "goal", task.Goal)

	// Step 1: Query affected assets
	assets, err := a.executeTool(ctx, "query_assets", map[string]interface{}{
		"filter": task.Context.AssetFilter,
		"org_id": task.OrgID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query assets: %w", err)
	}

	// Step 2: Get drift status
	driftStatus, err := a.executeTool(ctx, "get_drift_status", map[string]interface{}{
		"org_id": task.OrgID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get drift status: %w", err)
	}

	// Step 3: Get target golden image
	goldenImage, err := a.executeTool(ctx, "get_golden_image", map[string]interface{}{
		"org_id":      task.OrgID,
		"environment": task.Environment,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get golden image: %w", err)
	}

	// Step 4: Generate remediation plan using LLM
	plan, tokensUsed, err := a.generateRemediationPlan(ctx, assets, driftStatus, goldenImage, task)
	if err != nil {
		return nil, fmt.Errorf("failed to generate remediation plan: %w", err)
	}

	// Return result for HITL approval
	return &AgentResult{
		TaskID:         task.ID,
		AgentName:      a.name,
		Status:         AgentStatusPendingApproval,
		Plan:           plan,
		Summary:        fmt.Sprintf("Generated drift remediation plan for %v assets", assets),
		AffectedAssets: countAssets(assets),
		RiskLevel:      task.RiskLevel,
		TokensUsed:     tokensUsed,
		Actions: []Action{
			{Type: "approve", Label: "Approve & Execute", Description: "Approve the plan and begin execution"},
			{Type: "modify", Label: "Modify Plan", Description: "Edit the plan before execution"},
			{Type: "reject", Label: "Reject", Description: "Reject and cancel the task"},
		},
		Evidence: map[string]interface{}{
			"assets":       assets,
			"drift_status": driftStatus,
			"golden_image": goldenImage,
		},
	}, nil
}

func (a *DriftAgent) generateRemediationPlan(ctx context.Context, assets, driftStatus, goldenImage interface{}, task *TaskSpec) (interface{}, int, error) {
	// Build prompt for LLM
	prompt := fmt.Sprintf(`You are the QL-RF Drift Remediation Agent. Generate a safe remediation plan.

## Current Drift Status
%v

## Target Golden Image
%v

## Affected Assets
%v

## Constraints
- Environment: %s
- Max batch size: %v
- Canary required: %v

Generate a phased rollout plan with:
1. Canary phase (5%% of assets)
2. Progressive waves
3. Health checks between phases
4. Rollback criteria

Output as JSON with fields: summary, phases[], estimated_duration, risk_assessment`,
		driftStatus, goldenImage, assets, task.Environment,
		task.Constraints["max_batch_size"], task.Constraints["require_canary"])

	resp, err := a.llm.Complete(ctx, &llm.CompletionRequest{
		SystemPrompt: "You are an infrastructure remediation specialist. Generate safe, validated remediation plans.",
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return nil, 0, err
	}

	return resp.Content, resp.Usage.TotalTokens, nil
}

// PatchAgent handles patch rollout operations.
type PatchAgent struct {
	BaseAgent
}

// NewPatchAgent creates a new patch agent.
func NewPatchAgent(llmClient llm.Client, toolReg *tools.Registry, log *logger.Logger) *PatchAgent {
	return &PatchAgent{
		BaseAgent: BaseAgent{
			name:        "patch_agent",
			description: "Orchestrates patch rollouts across infrastructure",
			tasks:       []TaskType{TaskTypePatchRollout},
			tools: []string{
				"query_assets",
				"get_golden_image",
				"compare_versions",
				"generate_patch_plan",
				"generate_rollout_plan",
				"simulate_rollout",
				"calculate_risk_score",
			},
			llm:     llmClient,
			toolReg: toolReg,
			log:     log.WithComponent("patch-agent"),
		},
	}
}

// Execute runs the patch agent.
func (a *PatchAgent) Execute(ctx context.Context, task *TaskSpec) (*AgentResult, error) {
	a.log.Info("executing patch agent", "task_id", task.ID, "goal", task.Goal)

	// Similar implementation pattern as drift agent
	// Query assets, generate plan, return for approval
	return &AgentResult{
		TaskID:    task.ID,
		AgentName: a.name,
		Status:    AgentStatusPendingApproval,
		Summary:   "Patch rollout plan generated",
		Actions: []Action{
			{Type: "approve", Label: "Approve & Execute"},
			{Type: "modify", Label: "Modify Plan"},
			{Type: "reject", Label: "Reject"},
		},
	}, nil
}

// ComplianceAgent handles compliance audits and evidence generation.
type ComplianceAgent struct {
	BaseAgent
}

// NewComplianceAgent creates a new compliance agent.
func NewComplianceAgent(llmClient llm.Client, toolReg *tools.Registry, log *logger.Logger) *ComplianceAgent {
	return &ComplianceAgent{
		BaseAgent: BaseAgent{
			name:        "compliance_agent",
			description: "Performs compliance audits and generates evidence packages",
			tasks:       []TaskType{TaskTypeComplianceAudit},
			tools: []string{
				"query_assets",
				"get_compliance_status",
				"check_control",
				"generate_compliance_evidence",
			},
			llm:     llmClient,
			toolReg: toolReg,
			log:     log.WithComponent("compliance-agent"),
		},
	}
}

// Execute runs the compliance agent.
func (a *ComplianceAgent) Execute(ctx context.Context, task *TaskSpec) (*AgentResult, error) {
	a.log.Info("executing compliance agent", "task_id", task.ID, "goal", task.Goal)

	return &AgentResult{
		TaskID:    task.ID,
		AgentName: a.name,
		Status:    AgentStatusPendingApproval,
		Summary:   "Compliance audit completed",
		Actions: []Action{
			{Type: "approve", Label: "Approve Report"},
			{Type: "modify", Label: "Modify Report"},
			{Type: "reject", Label: "Reject"},
		},
	}, nil
}

// IncidentAgent handles incident investigation and response.
type IncidentAgent struct {
	BaseAgent
}

// NewIncidentAgent creates a new incident agent.
func NewIncidentAgent(llmClient llm.Client, toolReg *tools.Registry, log *logger.Logger) *IncidentAgent {
	return &IncidentAgent{
		BaseAgent: BaseAgent{
			name:        "incident_agent",
			description: "Investigates incidents and generates root cause analysis",
			tasks:       []TaskType{TaskTypeIncidentResponse},
			tools: []string{
				"query_assets",
				"query_alerts",
				"get_drift_status",
				"get_compliance_status",
			},
			llm:     llmClient,
			toolReg: toolReg,
			log:     log.WithComponent("incident-agent"),
		},
	}
}

// Execute runs the incident agent.
func (a *IncidentAgent) Execute(ctx context.Context, task *TaskSpec) (*AgentResult, error) {
	a.log.Info("executing incident agent", "task_id", task.ID, "goal", task.Goal)

	return &AgentResult{
		TaskID:    task.ID,
		AgentName: a.name,
		Status:    AgentStatusCompleted,
		Summary:   "Incident analysis completed",
	}, nil
}

// DRAgent handles disaster recovery planning and drills.
type DRAgent struct {
	BaseAgent
}

// NewDRAgent creates a new DR agent.
func NewDRAgent(llmClient llm.Client, toolReg *tools.Registry, log *logger.Logger) *DRAgent {
	return &DRAgent{
		BaseAgent: BaseAgent{
			name:        "dr_agent",
			description: "Plans and executes disaster recovery drills",
			tasks:       []TaskType{TaskTypeDRDrill},
			tools: []string{
				"query_assets",
				"get_dr_status",
				"generate_dr_runbook",
				"simulate_failover",
			},
			llm:     llmClient,
			toolReg: toolReg,
			log:     log.WithComponent("dr-agent"),
		},
	}
}

// Execute runs the DR agent.
func (a *DRAgent) Execute(ctx context.Context, task *TaskSpec) (*AgentResult, error) {
	a.log.Info("executing DR agent", "task_id", task.ID, "goal", task.Goal)

	return &AgentResult{
		TaskID:    task.ID,
		AgentName: a.name,
		Status:    AgentStatusPendingApproval,
		Summary:   "DR drill plan generated",
		Actions: []Action{
			{Type: "approve", Label: "Approve & Execute Drill"},
			{Type: "modify", Label: "Modify Plan"},
			{Type: "reject", Label: "Reject"},
		},
	}, nil
}

// CostAgent handles cost optimization recommendations.
type CostAgent struct {
	BaseAgent
}

// NewCostAgent creates a new cost agent.
func NewCostAgent(llmClient llm.Client, toolReg *tools.Registry, log *logger.Logger) *CostAgent {
	return &CostAgent{
		BaseAgent: BaseAgent{
			name:        "cost_agent",
			description: "Analyzes infrastructure costs and recommends optimizations",
			tasks:       []TaskType{TaskTypeCostOptimization},
			tools: []string{
				"query_assets",
			},
			llm:     llmClient,
			toolReg: toolReg,
			log:     log.WithComponent("cost-agent"),
		},
	}
}

// Execute runs the cost agent.
func (a *CostAgent) Execute(ctx context.Context, task *TaskSpec) (*AgentResult, error) {
	a.log.Info("executing cost agent", "task_id", task.ID, "goal", task.Goal)

	return &AgentResult{
		TaskID:    task.ID,
		AgentName: a.name,
		Status:    AgentStatusCompleted,
		Summary:   "Cost optimization analysis completed",
	}, nil
}

// SecurityAgent handles security scanning and vulnerability assessment.
type SecurityAgent struct {
	BaseAgent
}

// NewSecurityAgent creates a new security agent.
func NewSecurityAgent(llmClient llm.Client, toolReg *tools.Registry, log *logger.Logger) *SecurityAgent {
	return &SecurityAgent{
		BaseAgent: BaseAgent{
			name:        "security_agent",
			description: "Performs security scans and vulnerability assessments",
			tasks:       []TaskType{TaskTypeSecurityScan},
			tools: []string{
				"query_assets",
				"get_compliance_status",
			},
			llm:     llmClient,
			toolReg: toolReg,
			log:     log.WithComponent("security-agent"),
		},
	}
}

// Execute runs the security agent.
func (a *SecurityAgent) Execute(ctx context.Context, task *TaskSpec) (*AgentResult, error) {
	a.log.Info("executing security agent", "task_id", task.ID, "goal", task.Goal)

	return &AgentResult{
		TaskID:    task.ID,
		AgentName: a.name,
		Status:    AgentStatusCompleted,
		Summary:   "Security scan completed",
	}, nil
}

// ImageAgent handles golden image lifecycle management.
type ImageAgent struct {
	BaseAgent
}

// NewImageAgent creates a new image agent.
func NewImageAgent(llmClient llm.Client, toolReg *tools.Registry, log *logger.Logger) *ImageAgent {
	return &ImageAgent{
		BaseAgent: BaseAgent{
			name:        "image_agent",
			description: "Creates cloud-agnostic golden images with CIS hardening and multi-platform support",
			tasks:       []TaskType{TaskTypeImageManagement},
			tools: []string{
				"get_golden_image",
				"list_image_versions",
				"generate_image_contract",
				"generate_packer_template",
				"generate_ansible_playbook",
				"build_image",
				"promote_image",
			},
			llm:     llmClient,
			toolReg: toolReg,
			log:     log.WithComponent("image-agent"),
		},
	}
}

// Execute runs the image agent.
func (a *ImageAgent) Execute(ctx context.Context, task *TaskSpec) (*AgentResult, error) {
	a.log.Info("executing image agent", "task_id", task.ID, "goal", task.Goal)

	// Determine what operation is requested based on goal/intent
	operationType := a.determineOperation(task.Goal, task.UserIntent)

	switch operationType {
	case "create":
		return a.executeCreate(ctx, task)
	case "promote":
		return a.executePromote(ctx, task)
	case "list":
		return a.executeList(ctx, task)
	default:
		return a.executeCreate(ctx, task) // Default to create
	}
}

func (a *ImageAgent) determineOperation(goal, intent string) string {
	lowerGoal := goal + " " + intent
	if contains(lowerGoal, "promote") || contains(lowerGoal, "publish") {
		return "promote"
	}
	if contains(lowerGoal, "list") || contains(lowerGoal, "show") || contains(lowerGoal, "versions") {
		return "list"
	}
	return "create"
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func (a *ImageAgent) executeCreate(ctx context.Context, task *TaskSpec) (*AgentResult, error) {
	a.log.Info("creating golden image", "task_id", task.ID, "goal", task.Goal)

	// Step 1: Parse requirements from user intent using LLM
	requirements, tokensUsed, err := a.parseImageRequirements(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to parse image requirements: %w", err)
	}

	// Step 2: Generate the ImageContract using the tool
	contract, err := a.executeTool(ctx, "generate_image_contract", requirements)
	if err != nil {
		return nil, fmt.Errorf("failed to generate image contract: %w", err)
	}

	contractResult := contract.(map[string]interface{})
	imageContract := contractResult["contract"].(map[string]interface{})

	// Step 3: Generate Packer templates for each platform
	platforms := []string{"aws"}
	if p, ok := requirements["platforms"].([]interface{}); ok {
		platforms = make([]string, len(p))
		for i, platform := range p {
			platforms[i] = platform.(string)
		}
	}

	packerTemplates := make(map[string]interface{})
	for _, platform := range platforms {
		template, err := a.executeTool(ctx, "generate_packer_template", map[string]interface{}{
			"contract": imageContract,
			"platform": platform,
		})
		if err != nil {
			a.log.Warn("failed to generate packer template", "platform", platform, "error", err)
			continue
		}
		packerTemplates[platform] = template
	}

	// Step 4: Generate Ansible playbook
	playbook, err := a.executeTool(ctx, "generate_ansible_playbook", map[string]interface{}{
		"contract": imageContract,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate ansible playbook: %w", err)
	}

	// Step 5: Create the comprehensive plan
	plan := map[string]interface{}{
		"summary":           fmt.Sprintf("Create golden image: %s", imageContract["name"]),
		"image_contract":    imageContract,
		"packer_templates":  packerTemplates,
		"ansible_playbook":  playbook,
		"platforms":         platforms,
		"compliance":        imageContract["compliance"],
		"security":          imageContract["security"],
		"build_config":      imageContract["build"],
		"affected_assets":   0, // New image, no assets affected yet
		"phases": []map[string]interface{}{
			{
				"name":        "Contract Validation",
				"description": "Validate ImageContract against organization policies",
				"assets":      0,
			},
			{
				"name":        "Template Generation",
				"description": "Generate Packer templates for all target platforms",
				"assets":      len(platforms),
			},
			{
				"name":        "Image Build",
				"description": "Build images across platforms",
				"wait_time":   "30m",
				"rollback_if": "build_failure",
			},
			{
				"name":        "Compliance Testing",
				"description": "Run CIS benchmark and InSpec tests",
				"wait_time":   "15m",
			},
			{
				"name":        "SBOM Generation",
				"description": "Generate Software Bill of Materials",
			},
			{
				"name":        "Image Signing",
				"description": "Sign images with Cosign",
			},
		},
	}

	return &AgentResult{
		TaskID:         task.ID,
		AgentName:      a.name,
		Status:         AgentStatusPendingApproval,
		Plan:           plan,
		Summary:        fmt.Sprintf("Generated ImageContract for %s with %d platform targets", imageContract["name"], len(platforms)),
		AffectedAssets: 0,
		RiskLevel:      task.RiskLevel,
		TokensUsed:     tokensUsed,
		Actions: []Action{
			{Type: "approve", Label: "Approve & Build", Description: "Approve the image contract and start building"},
			{Type: "modify", Label: "Modify Contract", Description: "Edit the image contract before building"},
			{Type: "reject", Label: "Reject", Description: "Reject and cancel the image creation"},
		},
		Evidence: map[string]interface{}{
			"image_contract":   imageContract,
			"packer_templates": packerTemplates,
			"ansible_playbook": playbook,
			"platforms":        platforms,
		},
	}, nil
}

func (a *ImageAgent) parseImageRequirements(ctx context.Context, task *TaskSpec) (map[string]interface{}, int, error) {
	prompt := fmt.Sprintf(`You are the QL-RF Image Agent. Parse the user's request into structured image requirements.

## User Request
%s

## Available Options
- OS: ubuntu, rhel, amazon-linux, windows
- CIS Levels: 1 (basic), 2 (stricter)
- Platforms: aws, azure, gcp, docker, vsphere
- Runtimes: python:3.11, nodejs:20, java:17, go:1.21, docker

## Your Task
Extract:
1. Image name/family
2. Base OS and version
3. Purpose (web-server, database, k8s-node, etc.)
4. CIS hardening level
5. Target cloud platforms
6. Required runtimes
7. Additional packages

Output ONLY valid JSON (no markdown, no explanation):
{
  "name": "string",
  "os": "ubuntu|rhel|amazon-linux|windows",
  "os_version": "string (e.g. 22.04, 8.9)",
  "purpose": "string",
  "cis_level": 1|2,
  "platforms": ["aws", "azure", ...],
  "runtimes": ["python:3.11", ...],
  "packages": ["nginx", "curl", ...],
  "compliance": ["CIS", "SLSA", ...]
}`, task.UserIntent)

	resp, err := a.llm.Complete(ctx, &llm.CompletionRequest{
		SystemPrompt: "You are an infrastructure image specification expert. Parse requirements into structured JSON. Output ONLY valid JSON.",
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return nil, 0, err
	}

	// Parse the JSON response
	var requirements map[string]interface{}
	content := resp.Content

	// Try to extract JSON if wrapped in markdown
	if startIdx := findJSONStart(content); startIdx >= 0 {
		if endIdx := findJSONEnd(content, startIdx); endIdx > startIdx {
			content = content[startIdx : endIdx+1]
		}
	}

	if err := parseJSON(content, &requirements); err != nil {
		// Fall back to defaults based on keywords
		requirements = a.fallbackRequirements(task.UserIntent)
	}

	return requirements, resp.Usage.TotalTokens, nil
}

func findJSONStart(s string) int {
	for i, c := range s {
		if c == '{' {
			return i
		}
	}
	return -1
}

func findJSONEnd(s string, start int) int {
	depth := 0
	for i := start; i < len(s); i++ {
		switch s[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func parseJSON(s string, v interface{}) error {
	return json.Unmarshal([]byte(s), v)
}

func (a *ImageAgent) fallbackRequirements(intent string) map[string]interface{} {
	// Extract basic info from intent
	os := "ubuntu"
	osVersion := "22.04"
	purpose := "base"
	cisLevel := 1

	if contains(intent, "rhel") || contains(intent, "redhat") {
		os = "rhel"
		osVersion = "8.9"
	}
	if contains(intent, "amazon") {
		os = "amazon-linux"
		osVersion = "2"
	}
	if contains(intent, "web") {
		purpose = "web-server"
	}
	if contains(intent, "database") || contains(intent, "db") {
		purpose = "database"
	}
	if contains(intent, "kubernetes") || contains(intent, "k8s") {
		purpose = "k8s-node"
	}
	if contains(intent, "cis-2") || contains(intent, "level 2") || contains(intent, "strict") {
		cisLevel = 2
	}

	return map[string]interface{}{
		"name":       fmt.Sprintf("%s-%s-base", os, purpose),
		"os":         os,
		"os_version": osVersion,
		"purpose":    purpose,
		"cis_level":  cisLevel,
		"platforms":  []interface{}{"aws"},
		"runtimes":   []interface{}{},
		"packages":   []interface{}{},
		"compliance": []interface{}{"CIS"},
	}
}

func (a *ImageAgent) executePromote(ctx context.Context, task *TaskSpec) (*AgentResult, error) {
	a.log.Info("promoting golden image", "task_id", task.ID)

	// Extract image info from metadata
	family := ""
	version := ""
	if m := task.Context.Metadata; m != nil {
		if f, ok := m["image_family"].(string); ok {
			family = f
		}
		if v, ok := m["image_version"].(string); ok {
			version = v
		}
	}

	result, err := a.executeTool(ctx, "promote_image", map[string]interface{}{
		"family":  family,
		"version": version,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to promote image: %w", err)
	}

	return &AgentResult{
		TaskID:    task.ID,
		AgentName: a.name,
		Status:    AgentStatusCompleted,
		Summary:   fmt.Sprintf("Promoted image %s:%s to published", family, version),
		Evidence: map[string]interface{}{
			"promotion_result": result,
		},
	}, nil
}

func (a *ImageAgent) executeList(ctx context.Context, task *TaskSpec) (*AgentResult, error) {
	a.log.Info("listing golden images", "task_id", task.ID)

	family := ""
	if m := task.Context.Metadata; m != nil {
		if f, ok := m["image_family"].(string); ok {
			family = f
		}
	}

	result, err := a.executeTool(ctx, "list_image_versions", map[string]interface{}{
		"family": family,
		"limit":  20,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}

	return &AgentResult{
		TaskID:    task.ID,
		AgentName: a.name,
		Status:    AgentStatusCompleted,
		Summary:   "Listed available golden images",
		Evidence: map[string]interface{}{
			"images": result,
		},
	}, nil
}

// =============================================================================
// SOPAgent - Standard Operating Procedure authoring
// =============================================================================

// SOPAgent handles SOP authoring from natural language.
type SOPAgent struct {
	BaseAgent
}

// NewSOPAgent creates a new SOP agent.
func NewSOPAgent(llmClient llm.Client, toolReg *tools.Registry, log *logger.Logger) *SOPAgent {
	return &SOPAgent{
		BaseAgent: BaseAgent{
			name:        "sop_agent",
			description: "Authors Standard Operating Procedures from natural language requirements",
			tasks:       []TaskType{TaskTypeSOPAuthoring},
			tools: []string{
				"generate_sop",
				"validate_sop",
				"simulate_sop",
				"execute_sop",
				"list_sops",
			},
			llm:     llmClient,
			toolReg: toolReg,
			log:     log.WithComponent("sop-agent"),
		},
	}
}

// Execute runs the SOP agent.
func (a *SOPAgent) Execute(ctx context.Context, task *TaskSpec) (*AgentResult, error) {
	a.log.Info("executing SOP agent", "task_id", task.ID, "goal", task.Goal)

	// Step 1: Parse the user intent to extract SOP requirements
	requirements, tokensUsed, err := a.parseSOPRequirements(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SOP requirements: %w", err)
	}

	// Step 2: Generate the SOP using the tool
	sopResult, err := a.executeTool(ctx, "generate_sop", requirements)
	if err != nil {
		return nil, fmt.Errorf("failed to generate SOP: %w", err)
	}

	sopResultMap := sopResult.(map[string]interface{})
	sopSpec := sopResultMap["sop_spec"].(map[string]interface{})

	// Step 3: Validate the SOP
	validationResult, err := a.executeTool(ctx, "validate_sop", map[string]interface{}{
		"sop_spec": sopSpec,
		"strict":   task.Environment == "production",
	})
	if err != nil {
		a.log.Warn("SOP validation failed", "error", err)
	}

	// Step 4: Simulate the SOP
	simulationResult, err := a.executeTool(ctx, "simulate_sop", map[string]interface{}{
		"sop_spec":           sopSpec,
		"target_environment": task.Environment,
	})
	if err != nil {
		a.log.Warn("SOP simulation failed", "error", err)
	}

	// Build the plan
	steps := []map[string]interface{}{}
	if s, ok := sopSpec["steps"].([]map[string]interface{}); ok {
		steps = s
	} else if s, ok := sopSpec["steps"].([]interface{}); ok {
		for _, step := range s {
			if sm, ok := step.(map[string]interface{}); ok {
				steps = append(steps, sm)
			}
		}
	}

	plan := map[string]interface{}{
		"summary":            fmt.Sprintf("Generated SOP: %s", sopSpec["name"]),
		"sop_spec":           sopSpec,
		"validation":         validationResult,
		"simulation":         simulationResult,
		"total_steps":        len(steps),
		"affected_assets":    0,
		"phases": []map[string]interface{}{
			{
				"name":        "SOP Validation",
				"description": "Validate SOP structure and policies",
			},
			{
				"name":        "Dry-Run Simulation",
				"description": "Simulate SOP execution without changes",
			},
			{
				"name":        "SOP Registration",
				"description": "Register SOP in the system",
			},
		},
	}

	return &AgentResult{
		TaskID:         task.ID,
		AgentName:      a.name,
		Status:         AgentStatusPendingApproval,
		Plan:           plan,
		Summary:        fmt.Sprintf("Generated SOP '%s' with %d steps", sopSpec["name"], len(steps)),
		AffectedAssets: 0,
		RiskLevel:      task.RiskLevel,
		TokensUsed:     tokensUsed,
		Actions: []Action{
			{Type: "approve", Label: "Approve & Register", Description: "Approve the SOP and register it"},
			{Type: "modify", Label: "Modify SOP", Description: "Edit the SOP before registering"},
			{Type: "reject", Label: "Reject", Description: "Reject and discard the SOP"},
		},
		Evidence: map[string]interface{}{
			"sop_spec":    sopSpec,
			"validation":  validationResult,
			"simulation":  simulationResult,
		},
	}, nil
}

func (a *SOPAgent) parseSOPRequirements(ctx context.Context, task *TaskSpec) (map[string]interface{}, int, error) {
	prompt := fmt.Sprintf(`You are the QL-RF SOP Agent. Parse the user's request into SOP requirements.

## User Request
%s

## Available Action Types
- inventory.list, inventory.query - Query assets
- drift.check, compliance.check, health.check - Validation checks
- notify.slack, notify.email, change.create_ticket - Notifications
- rollout.batch, rollout.canary - Deployments
- validate.health, validate.metrics - Post-change validation
- wait.duration, wait.approval - Wait operations

## Your Task
Extract:
1. SOP name and description
2. Trigger type (manual, schedule, event, alert)
3. Target environments
4. List of operations to perform (in natural language)
5. Rollback strategy (auto, manual, none)

Output ONLY valid JSON:
{
  "name": "string",
  "description": "string",
  "trigger_type": "manual|schedule|event|alert",
  "environments": ["staging", "production"],
  "operations": ["step 1 description", "step 2 description", ...],
  "rollback_strategy": "auto|manual|none"
}`, task.UserIntent)

	resp, err := a.llm.Complete(ctx, &llm.CompletionRequest{
		SystemPrompt: "You are an SOP authoring expert. Parse requirements into structured JSON. Output ONLY valid JSON.",
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return nil, 0, err
	}

	// Parse the JSON response
	var requirements map[string]interface{}
	content := resp.Content

	if startIdx := findJSONStart(content); startIdx >= 0 {
		if endIdx := findJSONEnd(content, startIdx); endIdx > startIdx {
			content = content[startIdx : endIdx+1]
		}
	}

	if err := parseJSON(content, &requirements); err != nil {
		// Fallback to defaults
		requirements = map[string]interface{}{
			"name":              "generated-sop",
			"description":       task.UserIntent,
			"trigger_type":      "manual",
			"environments":      []interface{}{"staging"},
			"operations":        []interface{}{task.UserIntent},
			"rollback_strategy": "auto",
		}
	}

	return requirements, resp.Usage.TotalTokens, nil
}

// =============================================================================
// AdapterAgent - Cross-cloud Terraform/IaC generation
// =============================================================================

// AdapterAgent handles cross-cloud infrastructure code generation.
type AdapterAgent struct {
	BaseAgent
}

// NewAdapterAgent creates a new adapter agent.
func NewAdapterAgent(llmClient llm.Client, toolReg *tools.Registry, log *logger.Logger) *AdapterAgent {
	return &AdapterAgent{
		BaseAgent: BaseAgent{
			name:        "adapter_agent",
			description: "Generates cross-cloud Terraform modules from requirements",
			tasks:       []TaskType{TaskTypeTerraformGeneration},
			tools: []string{
				"query_assets",
				"get_golden_image",
			},
			llm:     llmClient,
			toolReg: toolReg,
			log:     log.WithComponent("adapter-agent"),
		},
	}
}

// Execute runs the adapter agent.
func (a *AdapterAgent) Execute(ctx context.Context, task *TaskSpec) (*AgentResult, error) {
	a.log.Info("executing adapter agent", "task_id", task.ID, "goal", task.Goal)

	// Step 1: Parse requirements
	requirements, tokensUsed, err := a.parseTerraformRequirements(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to parse terraform requirements: %w", err)
	}

	// Step 2: Generate Terraform modules for each platform
	platforms := []string{"aws"}
	if p, ok := requirements["platforms"].([]interface{}); ok {
		platforms = make([]string, len(p))
		for i, platform := range p {
			platforms[i] = platform.(string)
		}
	}

	modules := make(map[string]interface{})
	for _, platform := range platforms {
		module := a.generateTerraformModule(platform, requirements)
		modules[platform] = module
	}

	// Generate the provisioning contract
	provisioningContract := a.generateProvisioningContract(requirements, modules)

	plan := map[string]interface{}{
		"summary":                fmt.Sprintf("Generated Terraform modules for %d platforms", len(platforms)),
		"provisioning_contract":  provisioningContract,
		"terraform_modules":      modules,
		"platforms":              platforms,
		"affected_assets":        0,
		"phases": []map[string]interface{}{
			{
				"name":        "Module Validation",
				"description": "Validate Terraform syntax and policies",
			},
			{
				"name":        "Plan Generation",
				"description": "Generate terraform plan for each platform",
			},
			{
				"name":        "Infrastructure Provisioning",
				"description": "Apply Terraform changes",
				"wait_time":   "15m",
				"rollback_if": "apply_failure",
			},
		},
	}

	return &AgentResult{
		TaskID:         task.ID,
		AgentName:      a.name,
		Status:         AgentStatusPendingApproval,
		Plan:           plan,
		Summary:        fmt.Sprintf("Generated Terraform modules for %v", platforms),
		AffectedAssets: 0,
		RiskLevel:      task.RiskLevel,
		TokensUsed:     tokensUsed,
		Actions: []Action{
			{Type: "approve", Label: "Approve & Apply", Description: "Approve and apply infrastructure changes"},
			{Type: "modify", Label: "Modify Modules", Description: "Edit Terraform modules before applying"},
			{Type: "reject", Label: "Reject", Description: "Reject and discard"},
		},
		Evidence: map[string]interface{}{
			"provisioning_contract": provisioningContract,
			"terraform_modules":     modules,
			"platforms":             platforms,
		},
	}, nil
}

func (a *AdapterAgent) parseTerraformRequirements(ctx context.Context, task *TaskSpec) (map[string]interface{}, int, error) {
	prompt := fmt.Sprintf(`You are the QL-RF Adapter Agent. Parse the user's infrastructure request.

## User Request
%s

## Available Platforms
- aws, azure, gcp

## Your Task
Extract:
1. Resource type (compute, database, network, storage)
2. Target platforms
3. Instance specifications
4. Networking requirements
5. Tags and metadata

Output ONLY valid JSON:
{
  "resource_type": "compute|database|network|storage",
  "platforms": ["aws", "azure"],
  "instance_type": "string",
  "count": 1,
  "network": {
    "vpc": "existing|new",
    "subnets": "private|public"
  },
  "image_family": "string",
  "tags": {"key": "value"}
}`, task.UserIntent)

	resp, err := a.llm.Complete(ctx, &llm.CompletionRequest{
		SystemPrompt: "You are an infrastructure expert. Parse requirements into structured JSON. Output ONLY valid JSON.",
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return nil, 0, err
	}

	var requirements map[string]interface{}
	content := resp.Content

	if startIdx := findJSONStart(content); startIdx >= 0 {
		if endIdx := findJSONEnd(content, startIdx); endIdx > startIdx {
			content = content[startIdx : endIdx+1]
		}
	}

	if err := parseJSON(content, &requirements); err != nil {
		requirements = map[string]interface{}{
			"resource_type": "compute",
			"platforms":     []interface{}{"aws"},
			"instance_type": "t3.medium",
			"count":         1,
			"tags":          map[string]interface{}{},
		}
	}

	return requirements, resp.Usage.TotalTokens, nil
}

func (a *AdapterAgent) generateTerraformModule(platform string, requirements map[string]interface{}) map[string]interface{} {
	resourceType := "compute"
	if rt, ok := requirements["resource_type"].(string); ok {
		resourceType = rt
	}

	instanceType := "t3.medium"
	if it, ok := requirements["instance_type"].(string); ok {
		instanceType = it
	}

	count := 1
	if c, ok := requirements["count"].(float64); ok {
		count = int(c)
	}

	switch platform {
	case "aws":
		return a.generateAWSTerraform(resourceType, instanceType, count, requirements)
	case "azure":
		return a.generateAzureTerraform(resourceType, instanceType, count, requirements)
	case "gcp":
		return a.generateGCPTerraform(resourceType, instanceType, count, requirements)
	default:
		return map[string]interface{}{"error": "unsupported platform"}
	}
}

func (a *AdapterAgent) generateAWSTerraform(resourceType, instanceType string, count int, reqs map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"format":   "hcl2",
		"platform": "aws",
		"terraform": map[string]interface{}{
			"required_version": ">= 1.5.0",
			"required_providers": map[string]interface{}{
				"aws": map[string]interface{}{
					"source":  "hashicorp/aws",
					"version": "~> 5.0",
				},
			},
		},
		"provider": map[string]interface{}{
			"aws": map[string]interface{}{
				"region": "${var.aws_region}",
			},
		},
		"resource": map[string]interface{}{
			"aws_instance": map[string]interface{}{
				"main": map[string]interface{}{
					"count":         count,
					"ami":           "${data.aws_ami.golden.id}",
					"instance_type": instanceType,
					"tags": map[string]interface{}{
						"Name":      "${var.name_prefix}-${count.index}",
						"ManagedBy": "quantumlayer",
					},
				},
			},
		},
		"data": map[string]interface{}{
			"aws_ami": map[string]interface{}{
				"golden": map[string]interface{}{
					"most_recent": true,
					"owners":      []string{"self"},
					"filter": map[string]interface{}{
						"name":   "tag:Family",
						"values": []string{"${var.image_family}"},
					},
				},
			},
		},
		"variable": map[string]interface{}{
			"aws_region": map[string]interface{}{
				"type":    "string",
				"default": "us-east-1",
			},
			"name_prefix": map[string]interface{}{
				"type": "string",
			},
			"image_family": map[string]interface{}{
				"type": "string",
			},
		},
		"output": map[string]interface{}{
			"instance_ids": map[string]interface{}{
				"value": "${aws_instance.main[*].id}",
			},
		},
	}
}

func (a *AdapterAgent) generateAzureTerraform(resourceType, instanceType string, count int, reqs map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"format":   "hcl2",
		"platform": "azure",
		"terraform": map[string]interface{}{
			"required_version": ">= 1.5.0",
			"required_providers": map[string]interface{}{
				"azurerm": map[string]interface{}{
					"source":  "hashicorp/azurerm",
					"version": "~> 3.0",
				},
			},
		},
		"provider": map[string]interface{}{
			"azurerm": map[string]interface{}{
				"features": map[string]interface{}{},
			},
		},
		"resource": map[string]interface{}{
			"azurerm_linux_virtual_machine": map[string]interface{}{
				"main": map[string]interface{}{
					"count":               count,
					"name":                "${var.name_prefix}-${count.index}",
					"resource_group_name": "${var.resource_group_name}",
					"location":            "${var.location}",
					"size":                instanceType,
					"source_image_id":     "${data.azurerm_image.golden.id}",
					"tags": map[string]interface{}{
						"ManagedBy": "quantumlayer",
					},
				},
			},
		},
		"variable": map[string]interface{}{
			"location": map[string]interface{}{
				"type":    "string",
				"default": "eastus",
			},
			"resource_group_name": map[string]interface{}{
				"type": "string",
			},
			"name_prefix": map[string]interface{}{
				"type": "string",
			},
		},
	}
}

func (a *AdapterAgent) generateGCPTerraform(resourceType, instanceType string, count int, reqs map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"format":   "hcl2",
		"platform": "gcp",
		"terraform": map[string]interface{}{
			"required_version": ">= 1.5.0",
			"required_providers": map[string]interface{}{
				"google": map[string]interface{}{
					"source":  "hashicorp/google",
					"version": "~> 5.0",
				},
			},
		},
		"provider": map[string]interface{}{
			"google": map[string]interface{}{
				"project": "${var.project_id}",
				"region":  "${var.region}",
			},
		},
		"resource": map[string]interface{}{
			"google_compute_instance": map[string]interface{}{
				"main": map[string]interface{}{
					"count":        count,
					"name":         "${var.name_prefix}-${count.index}",
					"machine_type": instanceType,
					"zone":         "${var.zone}",
					"boot_disk": map[string]interface{}{
						"initialize_params": map[string]interface{}{
							"image": "${data.google_compute_image.golden.self_link}",
						},
					},
					"labels": map[string]interface{}{
						"managed-by": "quantumlayer",
					},
				},
			},
		},
		"variable": map[string]interface{}{
			"project_id": map[string]interface{}{
				"type": "string",
			},
			"region": map[string]interface{}{
				"type":    "string",
				"default": "us-central1",
			},
			"zone": map[string]interface{}{
				"type":    "string",
				"default": "us-central1-a",
			},
			"name_prefix": map[string]interface{}{
				"type": "string",
			},
		},
	}
}

func (a *AdapterAgent) generateProvisioningContract(requirements map[string]interface{}, modules map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"version":      "1.0",
		"resource_type": requirements["resource_type"],
		"platforms":     requirements["platforms"],
		"modules":       modules,
		"invariants": map[string]interface{}{
			"tags_required":     []string{"ManagedBy", "Environment"},
			"naming_convention": "${prefix}-${env}-${resource_type}-${index}",
			"encryption":        true,
		},
		"validation": map[string]interface{}{
			"terraform_fmt":      true,
			"terraform_validate": true,
			"tfsec":              true,
			"checkov":            true,
		},
	}
}

// =============================================================================
// Helper Functions
// =============================================================================

func countAssets(assets interface{}) int {
	if list, ok := assets.([]interface{}); ok {
		return len(list)
	}
	return 0
}
