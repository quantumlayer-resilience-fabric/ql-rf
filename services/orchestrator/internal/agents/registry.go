// Package agents provides the specialist agent registry and implementations.
package agents

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/llm"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/tools"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/validation"
)

// =============================================================================
// Task Types
// =============================================================================

// TaskType represents the type of task an agent can handle.
type TaskType string

const (
	TaskTypeDriftRemediation      TaskType = "drift_remediation"
	TaskTypePatchRollout          TaskType = "patch_rollout"
	TaskTypeComplianceAudit       TaskType = "compliance_audit"
	TaskTypeIncidentResponse      TaskType = "incident_investigation"
	TaskTypeDRDrill               TaskType = "dr_drill"
	TaskTypeCostOptimization      TaskType = "cost_optimization"
	TaskTypeSecurityScan          TaskType = "security_scan"
	TaskTypeImageManagement       TaskType = "image_management"
	TaskTypeSOPAuthoring          TaskType = "sop_authoring"
	TaskTypeTerraformGeneration   TaskType = "terraform_generation"
	TaskTypeCertificateRotation   TaskType = "certificate_rotation"
	TaskTypeVulnerabilityResponse TaskType = "vulnerability_response"
)

// =============================================================================
// Task Specification
// =============================================================================

// TaskSpec represents the specification for a task to be executed.
type TaskSpec struct {
	ID             string                 `json:"id"`
	TaskType       TaskType               `json:"task_type"`
	Goal           string                 `json:"goal"`
	UserIntent     string                 `json:"user_intent"`
	OrgID          string                 `json:"org_id"`
	UserID         string                 `json:"user_id"`
	Environment    string                 `json:"environment"`
	Context        TaskContext            `json:"context"`
	ToolsRequired  []string               `json:"tools_required"`
	Validation     ValidationSpec         `json:"validation"`
	RiskLevel      string                 `json:"risk_level"`
	HITLRequired   bool                   `json:"hitl_required"`
	TimeoutMinutes int                    `json:"timeout_minutes"`
	Constraints    map[string]interface{} `json:"constraints"`
}

// TaskContext provides context for task execution.
type TaskContext struct {
	Platforms   []string               `json:"platforms"`
	Regions     []string               `json:"regions"`
	AssetFilter string                 `json:"asset_filter"`
	Tags        map[string]string      `json:"tags"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// ValidationSpec specifies validation requirements.
type ValidationSpec struct {
	Schema      string                 `json:"schema"`
	Policies    []string               `json:"policies"`
	Constraints map[string]interface{} `json:"constraints"`
}

// =============================================================================
// Agent Result
// =============================================================================

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
	Type        string `json:"type"` // approve, modify, reject, retry
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

// =============================================================================
// Agent Interface
// =============================================================================

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

// AgentMetadata contains metadata about an agent.
type AgentMetadata struct {
	Name           string     `json:"name"`
	Description    string     `json:"description"`
	SupportedTasks []TaskType `json:"supported_tasks"`
	RequiredTools  []string   `json:"required_tools"`
}

// =============================================================================
// Registry
// =============================================================================

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
		NewCertificateAgent(r.llm, r.tools, r.log),
		NewVulnerabilityAgent(r.llm, r.tools, r.log),
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
		return nil, &ToolNotFoundError{Name: toolName}
	}
	return tool.Execute(ctx, params)
}

// ToolNotFoundError is returned when a required tool is not found.
type ToolNotFoundError struct {
	Name string
}

func (e *ToolNotFoundError) Error() string {
	return "tool not found: " + e.Name
}

// =============================================================================
// Helper Functions
// =============================================================================

// countAssets returns the count of assets from various result formats.
func countAssets(assets interface{}) int {
	if list, ok := assets.([]interface{}); ok {
		return len(list)
	}
	if m, ok := assets.(map[string]interface{}); ok {
		if count, ok := m["count"].(float64); ok {
			return int(count)
		}
		if items, ok := m["items"].([]interface{}); ok {
			return len(items)
		}
	}
	return 0
}

// contains checks if a string contains a substring (case-insensitive).
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// findJSONStart finds the start of a JSON object in a string.
func findJSONStart(s string) int {
	for i, c := range s {
		if c == '{' {
			return i
		}
	}
	return -1
}

// findJSONEnd finds the end of a JSON object starting at start.
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

// parseJSON attempts to parse a JSON string into the target.
func parseJSON(s string, v interface{}) error {
	return json.Unmarshal([]byte(s), v)
}
