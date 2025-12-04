//go:build integration

// Package integration contains end-to-end integration tests for QL-RF services.
// These tests require a running Docker environment with all dependencies.
// Run with: go test -tags=integration -v ./tests/integration/...
package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quantumlayerhq/ql-rf/pkg/config"
	"github.com/quantumlayerhq/ql-rf/pkg/database"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/agents"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/handlers"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/llm"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/meta"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/tools"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/validation"
)

// TestEnvironment holds the test environment configuration.
type TestEnvironment struct {
	DB        *database.DB
	Handler   *handlers.Handler
	Server    *httptest.Server
	Config    *config.Config
	Logger    *logger.Logger
	OrgID     string
	UserID    string
}

// setupTestEnvironment creates a test environment with all dependencies.
func setupTestEnvironment(t *testing.T) *TestEnvironment {
	t.Helper()

	// Load test configuration
	cfg := &config.Config{
		Env: "test",
		Database: config.DatabaseConfig{
			Host:     getEnvOrDefault("TEST_DB_HOST", "localhost"),
			Port:     5432,
			User:     getEnvOrDefault("TEST_DB_USER", "postgres"),
			Password: getEnvOrDefault("TEST_DB_PASSWORD", "postgres"),
			Database: getEnvOrDefault("TEST_DB_NAME", "qlrf_test"),
		},
		Orchestrator: config.OrchestratorConfig{
			DevMode: true, // Enable dev mode for testing
		},
	}

	log := logger.New("error", "text")

	// Connect to test database
	db, err := database.New(context.Background(), cfg.Database)
	if err != nil {
		t.Skipf("Skipping integration test: database not available: %v", err)
	}

	// Initialize components with mock LLM for testing
	mockLLM := llm.NewMock()
	toolRegistry := tools.NewRegistry(nil, log)
	agentRegistry := agents.NewRegistry(mockLLM, toolRegistry, log)
	validator := validation.NewPipeline(cfg, log)
	metaEngine := meta.NewEngine(mockLLM, agentRegistry, cfg, log)

	// Create handler
	handler := handlers.New(handlers.Config{
		DB:            db,
		Config:        cfg,
		Logger:        log,
		LLMClient:     mockLLM,
		MetaEngine:    metaEngine,
		AgentRegistry: agentRegistry,
		ToolRegistry:  toolRegistry,
		Validator:     validator,
		BuildInfo: handlers.BuildInfo{
			Version:   "test",
			BuildTime: time.Now().String(),
			GitCommit: "test-commit",
		},
	})

	// Create test server
	server := httptest.NewServer(handler.Router())

	return &TestEnvironment{
		DB:      db,
		Handler: handler,
		Server:  server,
		Config:  cfg,
		Logger:  log,
		OrgID:   "00000000-0000-0000-0000-000000000001",
		UserID:  "test-user",
	}
}

// teardownTestEnvironment cleans up the test environment.
func (env *TestEnvironment) teardown() {
	if env.Server != nil {
		env.Server.Close()
	}
	if env.DB != nil {
		env.DB.Close()
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

// =============================================================================
// Health Check Tests
// =============================================================================

func TestHealthEndpoint(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.teardown()

	resp, err := http.Get(env.Server.URL + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)

	assert.Equal(t, "healthy", body["status"])
	assert.Equal(t, "orchestrator", body["service"])
}

func TestReadyEndpoint(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.teardown()

	resp, err := http.Get(env.Server.URL + "/ready")
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should be OK or ServiceUnavailable depending on database
	assert.Contains(t, []int{http.StatusOK, http.StatusServiceUnavailable}, resp.StatusCode)

	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)

	assert.Contains(t, body, "checks")
}

// =============================================================================
// Task Execution Tests
// =============================================================================

func TestExecuteTask(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.teardown()

	tests := []struct {
		name           string
		request        map[string]interface{}
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "valid drift remediation request",
			request: map[string]interface{}{
				"intent": "Fix drift on production web servers",
				"org_id": env.OrgID,
				"environment": "production",
				"context": map[string]interface{}{
					"fleet_size": 100,
					"drift_score": 85.5,
				},
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				assert.NotEmpty(t, body["task_id"])
				assert.Contains(t, body, "task_spec")
				assert.Contains(t, body, "requires_hitl")
			},
		},
		{
			name: "missing intent returns error",
			request: map[string]interface{}{
				"org_id": env.OrgID,
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				assert.Contains(t, body, "error")
			},
		},
		{
			name: "missing org_id returns error",
			request: map[string]interface{}{
				"intent": "Some intent",
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				assert.Contains(t, body, "error")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reqBody, err := json.Marshal(tc.request)
			require.NoError(t, err)

			resp, err := http.Post(
				env.Server.URL+"/api/v1/ai/execute",
				"application/json",
				bytes.NewReader(reqBody),
			)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tc.expectedStatus, resp.StatusCode)

			var body map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&body)
			require.NoError(t, err)

			tc.checkResponse(t, body)
		})
	}
}

// =============================================================================
// Task Listing Tests
// =============================================================================

func TestListTasks(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.teardown()

	resp, err := http.Get(env.Server.URL + "/api/v1/ai/tasks")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)

	assert.Contains(t, body, "tasks")
	assert.Contains(t, body, "total")
}

func TestListTasksByState(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.teardown()

	states := []string{"planned", "approved", "executing", "completed", "failed"}

	for _, state := range states {
		t.Run("state_"+state, func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/api/v1/ai/tasks?state=%s", env.Server.URL, state))
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var body map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&body)
			require.NoError(t, err)

			assert.Contains(t, body, "tasks")
		})
	}
}

// =============================================================================
// Task Approval Workflow Tests
// =============================================================================

func TestTaskApprovalWorkflow(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.teardown()

	// Step 1: Create a task
	createReq := map[string]interface{}{
		"intent": "Remediate drift on test servers",
		"org_id": env.OrgID,
		"environment": "staging",
	}
	createBody, _ := json.Marshal(createReq)

	resp, err := http.Post(
		env.Server.URL+"/api/v1/ai/execute",
		"application/json",
		bytes.NewReader(createBody),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Skipf("Task creation failed with status %d - skipping workflow test", resp.StatusCode)
	}

	var createResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&createResp)
	require.NoError(t, err)

	taskID, ok := createResp["task_id"].(string)
	require.True(t, ok, "task_id should be present")
	require.NotEmpty(t, taskID)

	// Step 2: Get task details
	resp, err = http.Get(env.Server.URL + "/api/v1/ai/tasks/" + taskID)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Step 3: Approve the task
	approveReq := map[string]interface{}{
		"reason": "Approved for testing",
	}
	approveBody, _ := json.Marshal(approveReq)

	resp, err = http.Post(
		env.Server.URL+"/api/v1/ai/tasks/"+taskID+"/approve",
		"application/json",
		bytes.NewReader(approveBody),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var approveResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&approveResp)
	require.NoError(t, err)

	assert.Equal(t, "approved", approveResp["status"])
	assert.Equal(t, taskID, approveResp["task_id"])
}

func TestTaskRejectionWorkflow(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.teardown()

	// Step 1: Create a task
	createReq := map[string]interface{}{
		"intent": "High risk production change",
		"org_id": env.OrgID,
		"environment": "production",
	}
	createBody, _ := json.Marshal(createReq)

	resp, err := http.Post(
		env.Server.URL+"/api/v1/ai/execute",
		"application/json",
		bytes.NewReader(createBody),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Skipf("Task creation failed with status %d - skipping workflow test", resp.StatusCode)
	}

	var createResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&createResp)
	require.NoError(t, err)

	taskID, ok := createResp["task_id"].(string)
	require.True(t, ok, "task_id should be present")

	// Step 2: Reject the task
	rejectReq := map[string]interface{}{
		"reason": "Too risky for production",
	}
	rejectBody, _ := json.Marshal(rejectReq)

	resp, err = http.Post(
		env.Server.URL+"/api/v1/ai/tasks/"+taskID+"/reject",
		"application/json",
		bytes.NewReader(rejectBody),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var rejectResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&rejectResp)
	require.NoError(t, err)

	assert.Equal(t, "rejected", rejectResp["status"])
	assert.Equal(t, taskID, rejectResp["task_id"])
}

// =============================================================================
// Agent and Tool Metadata Tests
// =============================================================================

func TestListAgents(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.teardown()

	resp, err := http.Get(env.Server.URL + "/api/v1/ai/agents")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)

	assert.Contains(t, body, "agents")
	assert.Contains(t, body, "total")

	agents, ok := body["agents"].([]interface{})
	require.True(t, ok)

	// Verify known agents are present
	agentNames := make([]string, 0)
	for _, a := range agents {
		agent := a.(map[string]interface{})
		if name, ok := agent["name"].(string); ok {
			agentNames = append(agentNames, name)
		}
	}

	expectedAgents := []string{"drift", "patch", "compliance", "incident"}
	for _, expected := range expectedAgents {
		found := false
		for _, actual := range agentNames {
			if actual == expected || actual == expected+"_agent" {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected agent %s not found", expected)
	}
}

func TestListTools(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.teardown()

	resp, err := http.Get(env.Server.URL + "/api/v1/ai/tools")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)

	assert.Contains(t, body, "tools")
	assert.Contains(t, body, "total")
}

// =============================================================================
// CORS Tests
// =============================================================================

func TestCORSHeaders(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.teardown()

	req, err := http.NewRequest("OPTIONS", env.Server.URL+"/api/v1/ai/tasks", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Check CORS headers
	assert.NotEmpty(t, resp.Header.Get("Access-Control-Allow-Origin"))
	assert.NotEmpty(t, resp.Header.Get("Access-Control-Allow-Methods"))
	assert.NotEmpty(t, resp.Header.Get("Access-Control-Allow-Headers"))
}

// =============================================================================
// Error Handling Tests
// =============================================================================

func TestNotFoundTask(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.teardown()

	resp, err := http.Get(env.Server.URL + "/api/v1/ai/tasks/nonexistent-task-id")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)

	assert.Contains(t, body, "error")
}

func TestInvalidJSON(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.teardown()

	resp, err := http.Post(
		env.Server.URL+"/api/v1/ai/execute",
		"application/json",
		bytes.NewReader([]byte("invalid json")),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
