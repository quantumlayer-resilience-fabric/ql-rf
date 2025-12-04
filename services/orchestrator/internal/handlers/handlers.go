// Package handlers provides HTTP handlers for the AI orchestrator service.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/quantumlayerhq/ql-rf/pkg/config"
	"github.com/quantumlayerhq/ql-rf/pkg/database"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/pkg/models"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/agents"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/executor"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/llm"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/meta"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/middleware"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/notifier"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/temporal/worker"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/temporal/workflows"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/tools"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/validation"
)

// Config holds the handler configuration.
type Config struct {
	DB             *database.DB
	Config         *config.Config
	Logger         *logger.Logger
	LLMClient      llm.Client
	MetaEngine     *meta.Engine
	AgentRegistry  *agents.Registry
	ToolRegistry   *tools.Registry
	Validator      *validation.Pipeline
	Executor       *executor.Engine
	Notifier       *notifier.Notifier
	TemporalWorker *worker.Worker // Optional: Temporal worker for async execution
	BuildInfo      BuildInfo
}

// BuildInfo contains build information.
type BuildInfo struct {
	Version   string
	BuildTime string
	GitCommit string
}

// Handler provides HTTP handlers for the orchestrator.
type Handler struct {
	db             *database.DB
	cfg            *config.Config
	log            *logger.Logger
	llm            llm.Client
	meta           *meta.Engine
	agents         *agents.Registry
	tools          *tools.Registry
	validator      *validation.Pipeline
	executor       *executor.Engine
	notifier       *notifier.Notifier
	temporalWorker *worker.Worker
	buildInfo      BuildInfo
}

// New creates a new Handler.
func New(cfg Config) *Handler {
	return &Handler{
		db:             cfg.DB,
		cfg:            cfg.Config,
		log:            cfg.Logger.WithComponent("handlers"),
		llm:            cfg.LLMClient,
		meta:           cfg.MetaEngine,
		agents:         cfg.AgentRegistry,
		tools:          cfg.ToolRegistry,
		validator:      cfg.Validator,
		executor:       cfg.Executor,
		notifier:       cfg.Notifier,
		temporalWorker: cfg.TemporalWorker,
		buildInfo:      cfg.BuildInfo,
	}
}

// Router returns the HTTP router.
func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(h.loggingMiddleware)
	r.Use(chimw.Timeout(60 * time.Second))

	// CORS for frontend
	r.Use(h.corsMiddleware)

	// Health routes (no auth required)
	r.Get("/health", h.healthCheck)
	r.Get("/ready", h.readyCheck)

	// Auth middleware config
	authCfg := middleware.AuthConfig{
		ClerkPublishableKey: h.cfg.Clerk.PublishableKey,
		DevMode:             h.cfg.Orchestrator.DevMode,
	}

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// AI execution routes - require auth (optional in dev mode)
		r.Route("/ai", func(r chi.Router) {
			// Use optional auth - allows dev mode without tokens
			r.Use(middleware.OptionalAuth(authCfg, h.log))

			// Read-only routes - available to all authenticated users
			r.Get("/tasks", h.listTasks)
			r.Get("/tasks/{taskID}", h.getTask)
			r.Get("/tasks/{taskID}/executions", h.listExecutions)
			r.Get("/executions/{executionID}", h.getExecution)
			r.Get("/agents", h.listAgents)
			r.Get("/tools", h.listTools)

			// Task execution - requires execute:ai-tasks permission
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequirePermission(models.PermExecuteAITasks))
				r.Post("/execute", h.executeTask)
			})

			// Task approval/rejection - requires approve:ai-tasks permission
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequirePermission(models.PermApproveAITasks))
				r.Post("/tasks/{taskID}/approve", h.approveTask)
				r.Post("/tasks/{taskID}/reject", h.rejectTask)
				r.Post("/tasks/{taskID}/modify", h.modifyTask)
				r.Post("/tasks/{taskID}/cancel", h.cancelTask)
			})

			// Execution control - requires execute:ai-tasks permission
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequirePermission(models.PermExecuteAITasks))
				r.Post("/executions/{executionID}/pause", h.pauseExecution)
				r.Post("/executions/{executionID}/resume", h.resumeExecution)
				r.Post("/executions/{executionID}/cancel", h.cancelExecution)
			})
		})
	})

	return r
}

// corsMiddleware adds CORS headers for frontend access.
func (h *Handler) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow requests from frontend
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}

		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-Request-ID")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "300")
		w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID")

		// Handle preflight OPTIONS requests immediately
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware logs HTTP requests.
func (h *Handler) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		h.log.Debug("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"duration", time.Since(start),
			"request_id", chimw.GetReqID(r.Context()),
		)
	})
}

// =============================================================================
// Health Endpoints
// =============================================================================

func (h *Handler) healthCheck(w http.ResponseWriter, r *http.Request) {
	h.respond(w, http.StatusOK, map[string]interface{}{
		"status":    "healthy",
		"service":   "orchestrator",
		"version":   h.buildInfo.Version,
		"timestamp": time.Now().UTC(),
	})
}

func (h *Handler) readyCheck(w http.ResponseWriter, r *http.Request) {
	// Check dependencies
	checks := map[string]string{
		"database": "ok",
		"llm":      "ok",
	}

	// Check database
	if err := h.db.Health(r.Context()); err != nil {
		checks["database"] = "error: " + err.Error()
	}

	// Check LLM
	if h.llm == nil {
		checks["llm"] = "not configured"
	}

	// Determine overall status
	status := http.StatusOK
	for _, v := range checks {
		if v != "ok" && v != "not configured" {
			status = http.StatusServiceUnavailable
			break
		}
	}

	h.respond(w, status, map[string]interface{}{
		"status": map[string]bool{"ready": status == http.StatusOK},
		"checks": checks,
	})
}

// =============================================================================
// Task Execution Endpoints
// =============================================================================

// ExecuteTaskRequest is the request body for task execution.
type ExecuteTaskRequest struct {
	Intent      string                 `json:"intent"`
	OrgID       string                 `json:"org_id"`
	Environment string                 `json:"environment,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
}

// ExecuteTaskResponse is the response for task execution.
type ExecuteTaskResponse struct {
	TaskID       string                   `json:"task_id"`
	Status       string                   `json:"status"`
	TaskSpec     *agents.TaskSpec         `json:"task_spec"`
	AgentResult  *agents.AgentResult      `json:"agent_result,omitempty"`
	QualityScore *validation.QualityScore `json:"quality_score,omitempty"`
	RequiresHITL bool                     `json:"requires_hitl"`
	Message      string                   `json:"message"`
}

func (h *Handler) executeTask(w http.ResponseWriter, r *http.Request) {
	var req ExecuteTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	// Validate required fields
	if req.Intent == "" {
		h.respondError(w, http.StatusBadRequest, "intent is required", nil)
		return
	}
	if req.OrgID == "" {
		h.respondError(w, http.StatusBadRequest, "org_id is required", nil)
		return
	}

	// Get user/org from auth middleware context
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		userID = "dev-user" // Fallback for development
	}

	// Override org_id from auth context if not provided in request
	orgID := middleware.GetOrgID(r.Context())
	if orgID != "" && req.OrgID == "" {
		req.OrgID = orgID
	}

	ctx := r.Context()

	// Process the intent through the meta-engine
	taskSpec, err := h.meta.ProcessRequest(ctx, &meta.IntentRequest{
		UserIntent:  req.Intent,
		OrgID:       req.OrgID,
		UserID:      userID,
		Environment: req.Environment,
		Context:     req.Context,
	})
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to process intent", err)
		return
	}

	// Validate the task spec
	validationResult, err := h.validator.ValidatePlan(ctx, taskSpec, taskSpec.Environment)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "validation failed", err)
		return
	}

	if !validationResult.Valid {
		h.respond(w, http.StatusBadRequest, map[string]interface{}{
			"error":      "task validation failed",
			"task_spec":  taskSpec,
			"validation": validationResult,
		})
		return
	}

	// Get the appropriate agent
	agentList := h.agents.GetForTask(taskSpec.TaskType)
	if len(agentList) == 0 {
		h.respondError(w, http.StatusBadRequest, "no agent available for task type", nil)
		return
	}

	agent := agentList[0] // Use the first matching agent

	// Execute the agent
	result, err := agent.Execute(ctx, taskSpec)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "agent execution failed", err)
		return
	}

	// Store the task in the database
	if err := h.storeTask(ctx, taskSpec, result); err != nil {
		h.log.Error("failed to store task", "error", err)
		// Continue anyway - task is still valid
	}

	// Compute quality score for the artifact
	qualityScore := h.validator.ComputeQualityScore(ctx, &validation.QualityScoreRequest{
		ArtifactType:     string(taskSpec.TaskType),
		ArtifactID:       taskSpec.ID,
		ArtifactVersion:  "1.0",
		Data:             result.Plan,
		Environment:      taskSpec.Environment,
		ValidationResult: validationResult,
	})

	// If HITL is required and Temporal is available, start a workflow
	if taskSpec.HITLRequired && h.temporalWorker != nil {
		// Convert plan to map if possible
		var planMap map[string]interface{}
		if p, ok := result.Plan.(map[string]interface{}); ok {
			planMap = p
		}

		workflowInput := workflows.TaskWorkflowInput{
			TaskID:      taskSpec.ID,
			TaskType:    string(taskSpec.TaskType),
			OrgID:       req.OrgID,
			UserID:      userID,
			Environment: taskSpec.Environment,
			Goal:        taskSpec.Goal,
			RiskLevel:   string(taskSpec.RiskLevel),
			Plan:        planMap,
			Context:     req.Context,
		}

		runID, err := h.temporalWorker.StartTaskWorkflow(ctx, workflowInput)
		if err != nil {
			h.log.Error("failed to start workflow", "error", err)
			// Continue without workflow - task can still be approved via API
		} else {
			h.log.Info("started task workflow",
				"task_id", taskSpec.ID,
				"run_id", runID,
			)
		}
	}

	// Send notification for pending approval
	if taskSpec.HITLRequired && h.notifier != nil {
		go func() {
			notifyCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := h.notifier.NotifyTaskPendingApproval(
				notifyCtx,
				taskSpec.ID,
				string(taskSpec.TaskType),
				taskSpec.Environment,
				string(taskSpec.RiskLevel),
				result.Summary,
			); err != nil {
				h.log.Error("failed to send approval notification", "error", err)
			}
		}()
	}

	// Return the result
	h.respond(w, http.StatusOK, ExecuteTaskResponse{
		TaskID:       taskSpec.ID,
		Status:       string(result.Status),
		TaskSpec:     taskSpec,
		AgentResult:  result,
		QualityScore: qualityScore,
		RequiresHITL: taskSpec.HITLRequired,
		Message:      result.Summary,
	})
}

// TaskResponse represents a task with its plan for API responses.
type TaskResponse struct {
	ID           string                 `json:"id"`
	OrgID        string                 `json:"org_id"`
	UserIntent   string                 `json:"user_intent"`
	TaskSpec     map[string]interface{} `json:"task_spec,omitempty"`
	State        string                 `json:"state"`
	Source       string                 `json:"source"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	Plan         *PlanResponse          `json:"plan,omitempty"`
	RiskLevel    string                 `json:"risk_level,omitempty"`
	TaskType     string                 `json:"task_type,omitempty"`
	HITLRequired bool                   `json:"hitl_required"`
}

// PlanResponse represents a plan for API responses.
type PlanResponse struct {
	ID              string                 `json:"id"`
	Type            string                 `json:"type"`
	Payload         map[string]interface{} `json:"payload"`
	State           string                 `json:"state"`
	ApprovedBy      *string                `json:"approved_by,omitempty"`
	ApprovedAt      *time.Time             `json:"approved_at,omitempty"`
	RejectionReason *string                `json:"rejection_reason,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
}

func (h *Handler) listTasks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Query parameters - prefer auth context org_id
	orgID := r.URL.Query().Get("org_id")
	if orgID == "" {
		orgID = middleware.GetOrgID(ctx)
	}
	state := r.URL.Query().Get("state")
	limit := 50 // Default limit

	// Build query
	query := `
		SELECT
			t.id, t.org_id, t.user_intent, t.task_spec, t.state, t.source, t.created_at, t.updated_at,
			p.id as plan_id, p.type as plan_type, p.payload as plan_payload, p.state as plan_state,
			p.approved_by, p.approved_at, p.rejection_reason, p.created_at as plan_created_at
		FROM ai_tasks t
		LEFT JOIN ai_plans p ON p.task_id = t.id
		WHERE 1=1
	`
	args := []interface{}{}
	argNum := 1

	if orgID != "" {
		query += fmt.Sprintf(" AND t.org_id = $%d", argNum)
		args = append(args, orgID)
		argNum++
	}

	if state != "" {
		query += fmt.Sprintf(" AND t.state = $%d", argNum)
		args = append(args, state)
		argNum++
	}

	// Also filter by plan state if looking for pending approvals
	if state == "planned" {
		query += " AND (p.state = 'awaiting_approval' OR p.state = 'draft')"
	}

	query += fmt.Sprintf(" ORDER BY t.created_at DESC LIMIT %d", limit)

	rows, err := h.db.Pool.Query(ctx, query, args...)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to query tasks", err)
		return
	}
	defer rows.Close()

	tasks := []TaskResponse{}
	for rows.Next() {
		var task TaskResponse
		var taskSpecJSON, planPayloadJSON []byte
		var planID, planType, planState *string
		var approvedBy, rejectionReason *string
		var approvedAt, planCreatedAt *time.Time

		err := rows.Scan(
			&task.ID, &task.OrgID, &task.UserIntent, &taskSpecJSON, &task.State, &task.Source, &task.CreatedAt, &task.UpdatedAt,
			&planID, &planType, &planPayloadJSON, &planState,
			&approvedBy, &approvedAt, &rejectionReason, &planCreatedAt,
		)
		if err != nil {
			h.log.Error("failed to scan task row", "error", err)
			continue
		}

		// Parse task spec JSON
		if taskSpecJSON != nil {
			var taskSpec map[string]interface{}
			if err := json.Unmarshal(taskSpecJSON, &taskSpec); err == nil {
				task.TaskSpec = taskSpec
				// Extract common fields for convenience
				if rt, ok := taskSpec["risk_level"].(string); ok {
					task.RiskLevel = rt
				}
				if tt, ok := taskSpec["task_type"].(string); ok {
					task.TaskType = tt
				}
				if hitl, ok := taskSpec["hitl_required"].(bool); ok {
					task.HITLRequired = hitl
				}
			}
		}

		// Add plan if exists
		if planID != nil {
			task.Plan = &PlanResponse{
				ID:              *planID,
				Type:            *planType,
				State:           *planState,
				ApprovedBy:      approvedBy,
				ApprovedAt:      approvedAt,
				RejectionReason: rejectionReason,
				CreatedAt:       *planCreatedAt,
			}
			if planPayloadJSON != nil {
				var payload map[string]interface{}
				if err := json.Unmarshal(planPayloadJSON, &payload); err == nil {
					task.Plan.Payload = payload
				}
			}
		}

		tasks = append(tasks, task)
	}

	h.respond(w, http.StatusOK, map[string]interface{}{
		"tasks": tasks,
		"total": len(tasks),
	})
}

func (h *Handler) getTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	taskID := chi.URLParam(r, "taskID")

	query := `
		SELECT
			t.id, t.org_id, t.user_intent, t.task_spec, t.state, t.source, t.created_at, t.updated_at,
			p.id as plan_id, p.type as plan_type, p.payload as plan_payload, p.state as plan_state,
			p.approved_by, p.approved_at, p.rejection_reason, p.created_at as plan_created_at
		FROM ai_tasks t
		LEFT JOIN ai_plans p ON p.task_id = t.id
		WHERE t.id = $1
	`

	var task TaskResponse
	var taskSpecJSON, planPayloadJSON []byte
	var planID, planType, planState *string
	var approvedBy, rejectionReason *string
	var approvedAt, planCreatedAt *time.Time

	err := h.db.Pool.QueryRow(ctx, query, taskID).Scan(
		&task.ID, &task.OrgID, &task.UserIntent, &taskSpecJSON, &task.State, &task.Source, &task.CreatedAt, &task.UpdatedAt,
		&planID, &planType, &planPayloadJSON, &planState,
		&approvedBy, &approvedAt, &rejectionReason, &planCreatedAt,
	)
	if err != nil {
		h.respondError(w, http.StatusNotFound, "task not found", err)
		return
	}

	// Parse task spec JSON
	if taskSpecJSON != nil {
		var taskSpec map[string]interface{}
		if err := json.Unmarshal(taskSpecJSON, &taskSpec); err == nil {
			task.TaskSpec = taskSpec
			if rt, ok := taskSpec["risk_level"].(string); ok {
				task.RiskLevel = rt
			}
			if tt, ok := taskSpec["task_type"].(string); ok {
				task.TaskType = tt
			}
			if hitl, ok := taskSpec["hitl_required"].(bool); ok {
				task.HITLRequired = hitl
			}
		}
	}

	// Add plan if exists
	if planID != nil {
		task.Plan = &PlanResponse{
			ID:              *planID,
			Type:            *planType,
			State:           *planState,
			ApprovedBy:      approvedBy,
			ApprovedAt:      approvedAt,
			RejectionReason: rejectionReason,
			CreatedAt:       *planCreatedAt,
		}
		if planPayloadJSON != nil {
			var payload map[string]interface{}
			if err := json.Unmarshal(planPayloadJSON, &payload); err == nil {
				task.Plan.Payload = payload
			}
		}
	}

	h.respond(w, http.StatusOK, task)
}

// ApprovalRequest is the request body for task approval.
type ApprovalRequest struct {
	Reason        string                 `json:"reason,omitempty"`
	Modifications map[string]interface{} `json:"modifications,omitempty"`
}

func (h *Handler) approveTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")

	var req ApprovalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
		h.respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	// Get user ID from auth middleware context
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		userID = "anonymous"
	}

	ctx := r.Context()
	now := time.Now().UTC()

	// Check if task exists first
	var taskExists bool
	err := h.db.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM ai_tasks WHERE id = $1)`, taskID).Scan(&taskExists)
	if err != nil || !taskExists {
		h.respondError(w, http.StatusNotFound, "task not found", nil)
		return
	}

	h.log.Info("task approved",
		"task_id", taskID,
		"user_id", userID,
		"reason", req.Reason,
	)

	// Signal the Temporal workflow if available
	if h.temporalWorker != nil {
		approval := workflows.ApprovalSignal{
			Action:     "approve",
			ApprovedBy: userID,
			Reason:     req.Reason,
		}
		if err := h.temporalWorker.SignalApproval(ctx, taskID, approval); err != nil {
			h.log.Error("failed to signal workflow", "error", err)
			// Fall back to direct database update
		}
	}

	// Update task state in database
	_, err = h.db.Pool.Exec(ctx, `
		UPDATE ai_tasks
		SET state = 'approved', updated_at = $1
		WHERE id = $2
	`, now, taskID)
	if err != nil {
		h.log.Warn("failed to update task in database", "error", err)
	}

	// Update plan state to approved
	_, err = h.db.Pool.Exec(ctx, `
		UPDATE ai_plans
		SET state = 'approved', approved_by = $1, approved_at = $2, updated_at = $2
		WHERE task_id = $3
	`, userID, now, taskID)
	if err != nil {
		h.log.Warn("failed to update plan in database", "error", err)
	}

	// Start execution if executor is available
	var executionID string
	if h.executor != nil {
		execution, err := h.startExecution(ctx, taskID, userID)
		if err != nil {
			h.log.Error("failed to start execution", "error", err)
			// Don't fail the approval, just log the error
		} else {
			executionID = execution.ID
			h.log.Info("execution started",
				"task_id", taskID,
				"execution_id", execution.ID,
			)

			// Update task state to executing
			_, _ = h.db.Pool.Exec(ctx, `
				UPDATE ai_tasks SET state = 'executing', updated_at = $1 WHERE id = $2
			`, time.Now().UTC(), taskID)
		}
	}

	// Send approval notification
	if h.notifier != nil {
		go func() {
			notifyCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := h.notifier.NotifyTaskApproved(notifyCtx, taskID, userID); err != nil {
				h.log.Error("failed to send approval notification", "error", err)
			}
		}()
	}

	response := map[string]interface{}{
		"task_id":     taskID,
		"status":      "approved",
		"message":     "Task approved and queued for execution",
		"approved_by": userID,
		"approved_at": now,
	}
	if executionID != "" {
		response["execution_id"] = executionID
	}

	h.respond(w, http.StatusOK, response)
}

func (h *Handler) rejectTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")

	var req ApprovalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
		h.respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		userID = "anonymous"
	}

	ctx := r.Context()

	// Check if task exists first
	var taskExists bool
	err := h.db.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM ai_tasks WHERE id = $1)`, taskID).Scan(&taskExists)
	if err != nil || !taskExists {
		h.respondError(w, http.StatusNotFound, "task not found", nil)
		return
	}

	h.log.Info("task rejected",
		"task_id", taskID,
		"user_id", userID,
		"reason", req.Reason,
	)

	// Signal the Temporal workflow if available
	if h.temporalWorker != nil {
		rejection := workflows.ApprovalSignal{
			Action:     "reject",
			ApprovedBy: userID,
			Reason:     req.Reason,
		}
		if err := h.temporalWorker.SignalApproval(ctx, taskID, rejection); err != nil {
			h.log.Error("failed to signal workflow", "error", err)
		}
	}

	// Update task status in database
	_, err = h.db.Pool.Exec(ctx, `
		UPDATE ai_tasks
		SET state = 'rejected', updated_at = $1
		WHERE id = $2
	`, time.Now().UTC(), taskID)
	if err != nil {
		h.log.Warn("failed to update task in database", "error", err)
	}

	// Update plan with rejection reason
	_, _ = h.db.Pool.Exec(ctx, `
		UPDATE ai_plans
		SET state = 'rejected', rejection_reason = $1, updated_at = $2
		WHERE task_id = $3
	`, req.Reason, time.Now().UTC(), taskID)

	// Send rejection notification
	if h.notifier != nil {
		go func() {
			notifyCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := h.notifier.NotifyTaskRejected(notifyCtx, taskID, userID, req.Reason); err != nil {
				h.log.Error("failed to send rejection notification", "error", err)
			}
		}()
	}

	h.respond(w, http.StatusOK, map[string]interface{}{
		"task_id":     taskID,
		"status":      "rejected",
		"message":     "Task rejected",
		"rejected_by": userID,
		"rejected_at": time.Now().UTC(),
		"reason":      req.Reason,
	})
}

func (h *Handler) modifyTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	ctx := r.Context()

	var req ApprovalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	userID := middleware.GetUserID(ctx)
	if userID == "" {
		userID = "anonymous"
	}

	// Verify task exists and is in a modifiable state
	var currentState string
	err := h.db.Pool.QueryRow(ctx, `
		SELECT state FROM ai_tasks WHERE id = $1
	`, taskID).Scan(&currentState)
	if err != nil {
		h.respondError(w, http.StatusNotFound, "task not found", err)
		return
	}

	// Only allow modification of tasks in planned or rejected state
	if currentState != "planned" && currentState != "rejected" {
		h.respondError(w, http.StatusBadRequest,
			fmt.Sprintf("cannot modify task in state '%s', must be 'planned' or 'rejected'", currentState),
			nil)
		return
	}

	// Get current plan
	var planPayloadJSON []byte
	err = h.db.Pool.QueryRow(ctx, `
		SELECT payload FROM ai_plans WHERE task_id = $1
	`, taskID).Scan(&planPayloadJSON)
	if err != nil {
		h.respondError(w, http.StatusNotFound, "plan not found", err)
		return
	}

	// Parse and merge modifications
	var planPayload map[string]interface{}
	if err := json.Unmarshal(planPayloadJSON, &planPayload); err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to parse plan", err)
		return
	}

	// Apply modifications to plan
	for key, value := range req.Modifications {
		planPayload[key] = value
	}

	// Add modification audit trail
	modifications := planPayload["_modifications"]
	if modifications == nil {
		modifications = []interface{}{}
	}
	modList := modifications.([]interface{})
	modList = append(modList, map[string]interface{}{
		"modified_by": userID,
		"modified_at": time.Now().UTC(),
		"changes":     req.Modifications,
		"reason":      req.Reason,
	})
	planPayload["_modifications"] = modList

	// Save updated plan
	updatedPayloadJSON, err := json.Marshal(planPayload)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to serialize plan", err)
		return
	}

	now := time.Now().UTC()

	// Update plan
	_, err = h.db.Pool.Exec(ctx, `
		UPDATE ai_plans
		SET payload = $1, state = 'awaiting_approval', updated_at = $2
		WHERE task_id = $3
	`, updatedPayloadJSON, now, taskID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to update plan", err)
		return
	}

	// Update task state
	_, err = h.db.Pool.Exec(ctx, `
		UPDATE ai_tasks
		SET state = 'planned', updated_at = $1
		WHERE id = $2
	`, now, taskID)
	if err != nil {
		h.log.Warn("failed to update task state", "error", err)
	}

	h.log.Info("task modified",
		"task_id", taskID,
		"modified_by", userID,
		"modifications", req.Modifications,
	)

	h.respond(w, http.StatusOK, map[string]interface{}{
		"task_id":       taskID,
		"status":        "modified",
		"message":       "Task modifications applied, ready for re-approval",
		"modified_by":   userID,
		"modified_at":   now,
		"modifications": req.Modifications,
	})
}

func (h *Handler) cancelTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	ctx := r.Context()

	var req ApprovalRequest
	// Request body is optional for cancel
	json.NewDecoder(r.Body).Decode(&req)

	userID := middleware.GetUserID(ctx)
	if userID == "" {
		userID = "anonymous"
	}

	// Verify task exists and get current state
	var currentState string
	err := h.db.Pool.QueryRow(ctx, `
		SELECT state FROM ai_tasks WHERE id = $1
	`, taskID).Scan(&currentState)
	if err != nil {
		h.respondError(w, http.StatusNotFound, "task not found", err)
		return
	}

	// Cannot cancel already completed, failed, or cancelled tasks
	if currentState == "completed" || currentState == "failed" || currentState == "cancelled" {
		h.respondError(w, http.StatusBadRequest,
			fmt.Sprintf("cannot cancel task in state '%s'", currentState),
			nil)
		return
	}

	now := time.Now().UTC()

	// If task is executing, try to cancel the execution
	if currentState == "executing" {
		// Check for active execution
		var executionID string
		err := h.db.Pool.QueryRow(ctx, `
			SELECT id FROM ai_executions WHERE task_id = $1 AND status IN ('pending', 'running', 'paused')
			ORDER BY created_at DESC LIMIT 1
		`, taskID).Scan(&executionID)
		if err == nil && h.executor != nil {
			// Cancel the execution
			if err := h.executor.CancelExecution(ctx, executionID); err != nil {
				h.log.Warn("failed to cancel execution", "execution_id", executionID, "error", err)
			}
		}

		// Cancel via Temporal if available
		if h.temporalWorker != nil {
			if err := h.temporalWorker.CancelWorkflow(ctx, taskID); err != nil {
				h.log.Warn("failed to cancel temporal workflow", "task_id", taskID, "error", err)
			}
		}
	}

	// Update task state to cancelled
	_, err = h.db.Pool.Exec(ctx, `
		UPDATE ai_tasks
		SET state = 'cancelled', updated_at = $1
		WHERE id = $2
	`, now, taskID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to update task", err)
		return
	}

	// Update plan state
	_, _ = h.db.Pool.Exec(ctx, `
		UPDATE ai_plans
		SET state = 'cancelled', rejection_reason = $1, updated_at = $2
		WHERE task_id = $3
	`, req.Reason, now, taskID)

	h.log.Info("task cancelled",
		"task_id", taskID,
		"cancelled_by", userID,
		"previous_state", currentState,
		"reason", req.Reason,
	)

	// Send cancellation notification
	if h.notifier != nil {
		go func() {
			notifyCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := h.notifier.NotifyTaskRejected(notifyCtx, taskID, userID, "Task cancelled: "+req.Reason); err != nil {
				h.log.Error("failed to send cancellation notification", "error", err)
			}
		}()
	}

	h.respond(w, http.StatusOK, map[string]interface{}{
		"task_id":        taskID,
		"status":         "cancelled",
		"previous_state": currentState,
		"message":        "Task cancelled",
		"cancelled_by":   userID,
		"cancelled_at":   now,
		"reason":         req.Reason,
	})
}

// =============================================================================
// Metadata Endpoints
// =============================================================================

func (h *Handler) listAgents(w http.ResponseWriter, r *http.Request) {
	agentInfo := h.agents.AgentInfo()

	h.respond(w, http.StatusOK, map[string]interface{}{
		"agents": agentInfo,
		"total":  len(agentInfo),
	})
}

func (h *Handler) listTools(w http.ResponseWriter, r *http.Request) {
	toolInfo := h.tools.ToolInfo()

	h.respond(w, http.StatusOK, map[string]interface{}{
		"tools": toolInfo,
		"total": len(toolInfo),
	})
}

// =============================================================================
// Helper Methods
// =============================================================================

func (h *Handler) respond(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.log.Error("failed to encode response", "error", err)
	}
}

func (h *Handler) respondError(w http.ResponseWriter, status int, message string, err error) {
	h.log.Error(message, "error", err)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	response := map[string]interface{}{
		"error":   message,
		"status":  status,
	}
	if err != nil && h.cfg.Env != "production" {
		response["details"] = err.Error()
	}

	if encErr := json.NewEncoder(w).Encode(response); encErr != nil {
		h.log.Error("failed to encode error response", "error", encErr)
	}
}

func (h *Handler) storeTask(ctx context.Context, spec *agents.TaskSpec, result *agents.AgentResult) error {
	h.log.Debug("storing task",
		"task_id", spec.ID,
		"task_type", spec.TaskType,
		"status", result.Status,
	)

	// Get user UUID - use system user as fallback
	userID := spec.UserID
	if userID == "" || userID == "anonymous" {
		userID = "00000000-0000-0000-0000-000000000001" // System user
	}

	// Verify user UUID is valid
	if _, err := uuid.Parse(userID); err != nil {
		userID = "00000000-0000-0000-0000-000000000001" // Fall back to system user
	}

	// Serialize task spec to JSON
	taskSpecJSON, err := json.Marshal(spec)
	if err != nil {
		return err
	}

	// Determine state based on result status
	state := "created"
	switch result.Status {
	case agents.AgentStatusPendingApproval:
		state = "planned"
	case agents.AgentStatusFailed:
		state = "failed"
	}

	// Insert task into ai_tasks
	query := `
		INSERT INTO ai_tasks (id, org_id, created_by, user_intent, task_spec, state, source, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, 'api', NOW(), NOW())
		ON CONFLICT (id) DO UPDATE SET
			task_spec = EXCLUDED.task_spec,
			state = EXCLUDED.state,
			updated_at = NOW()
	`
	_, err = h.db.Pool.Exec(ctx, query,
		spec.ID,
		spec.OrgID,
		userID,
		spec.UserIntent,
		taskSpecJSON,
		state,
	)
	if err != nil {
		h.log.Error("failed to insert task", "error", err)
		return err
	}

	// Store the plan in ai_plans if result has a plan
	if result.Plan != nil {
		if err := h.storePlan(ctx, spec, result); err != nil {
			h.log.Error("failed to store plan", "error", err)
			return err
		}
	}

	h.log.Info("task stored successfully", "task_id", spec.ID)
	return nil
}

func (h *Handler) storePlan(ctx context.Context, spec *agents.TaskSpec, result *agents.AgentResult) error {
	// Serialize plan to JSON
	planJSON, err := json.Marshal(result.Plan)
	if err != nil {
		return err
	}

	// Map task type to plan type
	planType := "drift_plan" // default
	switch spec.TaskType {
	case agents.TaskTypeDriftRemediation:
		planType = "drift_plan"
	case agents.TaskTypePatchRollout:
		planType = "patch_plan"
	case agents.TaskTypeComplianceAudit:
		planType = "compliance_report"
	case agents.TaskTypeIncidentResponse:
		planType = "incident_analysis"
	case agents.TaskTypeDRDrill:
		planType = "dr_runbook"
	case agents.TaskTypeCostOptimization:
		planType = "cost_optimization_plan"
	case agents.TaskTypeSecurityScan:
		planType = "security_report"
	case agents.TaskTypeImageManagement:
		planType = "image_spec"
	}

	// Determine plan state
	planState := "draft"
	if spec.HITLRequired {
		planState = "awaiting_approval"
	}

	// Insert plan into ai_plans
	// First check if plan already exists for this task
	var existingPlanID string
	checkQuery := `SELECT id FROM ai_plans WHERE task_id = $1 LIMIT 1`
	err = h.db.Pool.QueryRow(ctx, checkQuery, spec.ID).Scan(&existingPlanID)

	if err == nil {
		// Plan exists, update it
		updateQuery := `
			UPDATE ai_plans SET payload = $1, state = $2, updated_at = NOW()
			WHERE task_id = $3
		`
		_, err = h.db.Pool.Exec(ctx, updateQuery, planJSON, planState, spec.ID)
	} else {
		// Plan doesn't exist, insert it
		insertQuery := `
			INSERT INTO ai_plans (task_id, type, payload, state, created_at, updated_at)
			VALUES ($1, $2, $3, $4, NOW(), NOW())
		`
		_, err = h.db.Pool.Exec(ctx, insertQuery,
			spec.ID,
			planType,
			planJSON,
			planState,
		)
	}
	if err != nil {
		return err
	}

	h.log.Debug("plan stored successfully", "task_id", spec.ID, "type", planType)
	return nil
}

// =============================================================================
// Execution Endpoints
// =============================================================================

func (h *Handler) listExecutions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	taskID := chi.URLParam(r, "taskID")

	if h.executor == nil {
		h.respondError(w, http.StatusServiceUnavailable, "executor not available", nil)
		return
	}

	executions, err := h.executor.ListExecutions(ctx, taskID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "failed to list executions", err)
		return
	}

	h.respond(w, http.StatusOK, map[string]interface{}{
		"executions": executions,
		"total":      len(executions),
	})
}

func (h *Handler) getExecution(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	executionID := chi.URLParam(r, "executionID")

	if h.executor == nil {
		h.respondError(w, http.StatusServiceUnavailable, "executor not available", nil)
		return
	}

	execution, err := h.executor.GetExecution(ctx, executionID)
	if err != nil {
		h.respondError(w, http.StatusNotFound, "execution not found", err)
		return
	}

	h.respond(w, http.StatusOK, execution)
}

func (h *Handler) pauseExecution(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	executionID := chi.URLParam(r, "executionID")

	if h.executor == nil {
		h.respondError(w, http.StatusServiceUnavailable, "executor not available", nil)
		return
	}

	if err := h.executor.PauseExecution(ctx, executionID); err != nil {
		h.respondError(w, http.StatusBadRequest, "failed to pause execution", err)
		return
	}

	h.respond(w, http.StatusOK, map[string]interface{}{
		"execution_id": executionID,
		"status":       "paused",
		"message":      "Execution paused",
	})
}

func (h *Handler) resumeExecution(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	executionID := chi.URLParam(r, "executionID")

	if h.executor == nil {
		h.respondError(w, http.StatusServiceUnavailable, "executor not available", nil)
		return
	}

	if err := h.executor.ResumeExecution(ctx, executionID); err != nil {
		h.respondError(w, http.StatusBadRequest, "failed to resume execution", err)
		return
	}

	h.respond(w, http.StatusOK, map[string]interface{}{
		"execution_id": executionID,
		"status":       "running",
		"message":      "Execution resumed",
	})
}

func (h *Handler) cancelExecution(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	executionID := chi.URLParam(r, "executionID")

	if h.executor == nil {
		h.respondError(w, http.StatusServiceUnavailable, "executor not available", nil)
		return
	}

	if err := h.executor.CancelExecution(ctx, executionID); err != nil {
		h.respondError(w, http.StatusBadRequest, "failed to cancel execution", err)
		return
	}

	h.respond(w, http.StatusOK, map[string]interface{}{
		"execution_id": executionID,
		"status":       "cancelled",
		"message":      "Execution cancelled",
	})
}

// startExecution starts execution of an approved plan.
func (h *Handler) startExecution(ctx context.Context, taskID, userID string) (*executor.Execution, error) {
	if h.executor == nil {
		return nil, fmt.Errorf("executor not available")
	}

	// Load task and plan from database
	query := `
		SELECT t.org_id, t.task_spec, p.payload
		FROM ai_tasks t
		JOIN ai_plans p ON p.task_id = t.id
		WHERE t.id = $1 AND p.state = 'approved'
	`

	var orgID string
	var taskSpecJSON, planPayloadJSON []byte
	err := h.db.Pool.QueryRow(ctx, query, taskID).Scan(&orgID, &taskSpecJSON, &planPayloadJSON)
	if err != nil {
		return nil, fmt.Errorf("task or approved plan not found: %w", err)
	}

	// Parse task spec and plan
	var taskSpec map[string]interface{}
	if err := json.Unmarshal(taskSpecJSON, &taskSpec); err != nil {
		return nil, fmt.Errorf("failed to parse task spec: %w", err)
	}

	var planPayload map[string]interface{}
	if err := json.Unmarshal(planPayloadJSON, &planPayload); err != nil {
		return nil, fmt.Errorf("failed to parse plan: %w", err)
	}

	// Extract phases from plan
	phases := []executor.ExecutionPhase{}
	if planPhases, ok := planPayload["phases"].([]interface{}); ok {
		for _, p := range planPhases {
			if phaseMap, ok := p.(map[string]interface{}); ok {
				phase := executor.ExecutionPhase{
					Name: fmt.Sprintf("%v", phaseMap["name"]),
				}
				if wt, ok := phaseMap["wait_time"].(string); ok {
					phase.WaitTime = wt
				}
				if rb, ok := phaseMap["rollback_if"].(string); ok {
					phase.RollbackIf = rb
				}
				if assets, ok := phaseMap["assets"].([]interface{}); ok {
					for _, a := range assets {
						if assetMap, ok := a.(map[string]interface{}); ok {
							phase.Assets = append(phase.Assets, assetMap)
						}
					}
				}
				phases = append(phases, phase)
			}
		}
	}

	// Build execution plan
	taskType := ""
	if tt, ok := taskSpec["task_type"].(string); ok {
		taskType = tt
	}
	environment := "production"
	if env, ok := taskSpec["environment"].(string); ok {
		environment = env
	}

	execPlan := &executor.ExecutionPlan{
		TaskID:      taskID,
		PlanID:      taskID, // Using task ID as plan ID for simplicity
		OrgID:       orgID,
		UserID:      userID,
		TaskType:    taskType,
		Environment: environment,
		Phases:      phases,
		Metadata:    planPayload,
	}

	// Check for rollback strategy
	if rollback, ok := planPayload["rollback"].(map[string]interface{}); ok {
		strategy := "manual"
		if s, ok := rollback["strategy"].(string); ok {
			strategy = s
		}
		execPlan.Rollback = &executor.RollbackPlan{
			Strategy: strategy,
		}
	}

	// Start execution
	return h.executor.Execute(ctx, execPlan)
}
