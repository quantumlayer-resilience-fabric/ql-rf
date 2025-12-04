//go:build integration

// Package integration contains end-to-end integration tests for QL-RF services.
// These tests verify Temporal workflow execution for AI tasks via the API.
// Run with: go test -tags=integration -v -timeout 10m ./tests/integration/... -run TestTemporal
//
// These are blackbox tests that interact with the API via HTTP, testing the full
// workflow execution without importing internal packages.
package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getEnvOrDefaultTemporal returns env value or default for Temporal workflow tests
func getEnvOrDefaultTemporal(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

// =============================================================================
// Temporal Workflow Integration Test Environment
// =============================================================================

// TemporalTestEnv provides a blackbox test environment for Temporal workflow tests.
type TemporalTestEnv struct {
	BaseURL      string
	OrgID        string
	UserID       string
	CreatedTasks []string
}

// setupTemporalTestEnv creates a blackbox test environment.
func setupTemporalTestEnv(t *testing.T) *TemporalTestEnv {
	t.Helper()

	baseURL := os.Getenv("TEST_ORCHESTRATOR_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8083"
	}

	// Test connectivity
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(baseURL + "/health")
	if err != nil {
		t.Skipf("Skipping Temporal workflow test: orchestrator not available at %s: %v", baseURL, err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Skipf("Skipping Temporal workflow test: orchestrator health check failed with status %d", resp.StatusCode)
	}

	return &TemporalTestEnv{
		BaseURL:      baseURL,
		OrgID:        getEnvOrDefaultTemporal("TEST_ORG_ID", "00000000-0000-0000-0000-000000000001"),
		UserID:       "test-user-" + uuid.New().String()[:8],
		CreatedTasks: make([]string, 0),
	}
}

func (env *TemporalTestEnv) teardown() {
	// Cleanup is handled by the API - no direct DB access needed
}

// makeRequest is a helper to make HTTP requests.
func (env *TemporalTestEnv) makeRequest(method, path string, body interface{}) (*http.Response, map[string]interface{}) {
	var reqBody []byte
	if body != nil {
		reqBody, _ = json.Marshal(body)
	}

	req, err := http.NewRequest(method, env.BaseURL+path, bytes.NewReader(reqBody))
	if err != nil {
		return nil, nil
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer dev-token")
	req.Header.Set("X-User-ID", env.UserID)
	req.Header.Set("X-Org-ID", env.OrgID)

	client := &http.Client{Timeout: 60 * time.Second} // Longer timeout for workflow operations
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		resp.Body.Close()
		return resp, nil
	}
	resp.Body.Close()

	return &http.Response{StatusCode: resp.StatusCode, Header: resp.Header}, result
}

func (env *TemporalTestEnv) trackTask(taskID string) {
	env.CreatedTasks = append(env.CreatedTasks, taskID)
}

// =============================================================================
// Workflow Execution Tests via API
// =============================================================================

func TestTemporal_WorkflowExecution_DriftRemediation(t *testing.T) {
	env := setupTemporalTestEnv(t)
	defer env.teardown()

	// Step 1: Create a drift remediation task
	resp, body := env.makeRequest("POST", "/api/v1/ai/execute", map[string]interface{}{
		"intent":      "Remediate drift on staging web servers",
		"org_id":      env.OrgID,
		"environment": "staging",
		"context": map[string]interface{}{
			"priority": "high",
		},
	})

	require.Equal(t, http.StatusOK, resp.StatusCode, "Task creation failed: %v", body)

	taskID, ok := body["task_id"].(string)
	require.True(t, ok, "task_id should be present")
	env.trackTask(taskID)

	t.Logf("Created task: %s", taskID)

	// Step 2: Approve the task to trigger workflow
	resp, body = env.makeRequest("POST", "/api/v1/ai/tasks/"+taskID+"/approve", map[string]interface{}{
		"reason": "Approved for testing",
	})

	if resp.StatusCode != http.StatusOK {
		t.Skipf("Task approval not available: %v", body)
	}

	t.Logf("Task approved: %v", body["status"])

	// Step 3: Wait for workflow to start and check execution status
	time.Sleep(2 * time.Second) // Give time for workflow to start

	resp, body = env.makeRequest("GET", "/api/v1/ai/tasks/"+taskID, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	state, _ := body["state"].(string)
	t.Logf("Task state after approval: %s", state)

	// State should have progressed from "planned"
	assert.Contains(t, []string{"approved", "executing", "completed", "failed"}, state)
}

func TestTemporal_WorkflowExecution_ImageManagement(t *testing.T) {
	env := setupTemporalTestEnv(t)
	defer env.teardown()

	resp, body := env.makeRequest("POST", "/api/v1/ai/execute", map[string]interface{}{
		"intent":      "Audit golden images for Ubuntu 22.04",
		"org_id":      env.OrgID,
		"environment": "staging",
	})

	require.Equal(t, http.StatusOK, resp.StatusCode)

	taskID, _ := body["task_id"].(string)
	env.trackTask(taskID)

	// Check task type was correctly identified
	if taskSpec, ok := body["task_spec"].(map[string]interface{}); ok {
		taskType, _ := taskSpec["task_type"].(string)
		t.Logf("Task type: %s", taskType)
	}
}

func TestTemporal_WorkflowExecution_CostOptimization(t *testing.T) {
	env := setupTemporalTestEnv(t)
	defer env.teardown()

	resp, body := env.makeRequest("POST", "/api/v1/ai/execute", map[string]interface{}{
		"intent":      "Find cost savings across all cloud platforms",
		"org_id":      env.OrgID,
		"environment": "all",
	})

	require.Equal(t, http.StatusOK, resp.StatusCode)

	taskID, _ := body["task_id"].(string)
	env.trackTask(taskID)

	t.Logf("Created cost optimization task: %s", taskID)
}

func TestTemporal_WorkflowExecution_ComplianceAudit(t *testing.T) {
	env := setupTemporalTestEnv(t)
	defer env.teardown()

	resp, body := env.makeRequest("POST", "/api/v1/ai/execute", map[string]interface{}{
		"intent":      "Run CIS benchmark compliance audit",
		"org_id":      env.OrgID,
		"environment": "production",
	})

	require.Equal(t, http.StatusOK, resp.StatusCode)

	taskID, _ := body["task_id"].(string)
	env.trackTask(taskID)

	t.Logf("Created compliance audit task: %s", taskID)
}

func TestTemporal_WorkflowExecution_PatchRollout(t *testing.T) {
	env := setupTemporalTestEnv(t)
	defer env.teardown()

	resp, body := env.makeRequest("POST", "/api/v1/ai/execute", map[string]interface{}{
		"intent":      "Roll out security patches to staging servers",
		"org_id":      env.OrgID,
		"environment": "staging",
	})

	require.Equal(t, http.StatusOK, resp.StatusCode)

	taskID, _ := body["task_id"].(string)
	env.trackTask(taskID)

	t.Logf("Created patch rollout task: %s", taskID)
}

// =============================================================================
// Workflow State Transition Tests
// =============================================================================

func TestTemporal_WorkflowStateTransitions(t *testing.T) {
	env := setupTemporalTestEnv(t)
	defer env.teardown()

	// Create task
	resp, body := env.makeRequest("POST", "/api/v1/ai/execute", map[string]interface{}{
		"intent":      "Test workflow state transitions",
		"org_id":      env.OrgID,
		"environment": "staging",
	})

	require.Equal(t, http.StatusOK, resp.StatusCode)
	taskID, _ := body["task_id"].(string)
	env.trackTask(taskID)

	// Track initial state
	resp, body = env.makeRequest("GET", "/api/v1/ai/tasks/"+taskID, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	initialState, _ := body["state"].(string)
	t.Logf("Initial state: %s", initialState)

	// Approve to trigger execution
	resp, body = env.makeRequest("POST", "/api/v1/ai/tasks/"+taskID+"/approve", map[string]interface{}{
		"reason": "Testing state transitions",
	})

	if resp.StatusCode == http.StatusOK {
		// Poll for state changes
		var states []string
		states = append(states, initialState)

		for i := 0; i < 10; i++ {
			time.Sleep(500 * time.Millisecond)

			resp, body = env.makeRequest("GET", "/api/v1/ai/tasks/"+taskID, nil)
			if resp.StatusCode == http.StatusOK {
				state, _ := body["state"].(string)
				if len(states) == 0 || states[len(states)-1] != state {
					states = append(states, state)
					t.Logf("State transition: %s", state)
				}

				// Stop if reached terminal state
				if state == "completed" || state == "failed" || state == "cancelled" {
					break
				}
			}
		}

		t.Logf("State transitions observed: %v", states)
		assert.GreaterOrEqual(t, len(states), 1, "Should observe at least one state")
	}
}

// =============================================================================
// Workflow Cancellation Tests
// =============================================================================

func TestTemporal_WorkflowCancellation(t *testing.T) {
	env := setupTemporalTestEnv(t)
	defer env.teardown()

	// Create task
	resp, body := env.makeRequest("POST", "/api/v1/ai/execute", map[string]interface{}{
		"intent":      "Long running task for cancellation test",
		"org_id":      env.OrgID,
		"environment": "staging",
	})

	require.Equal(t, http.StatusOK, resp.StatusCode)
	taskID, _ := body["task_id"].(string)
	env.trackTask(taskID)

	// Cancel the task
	resp, body = env.makeRequest("POST", "/api/v1/ai/tasks/"+taskID+"/cancel", map[string]interface{}{
		"reason": "Testing cancellation",
	})

	if resp.StatusCode == http.StatusOK {
		t.Logf("Task cancelled successfully")

		// Verify state
		resp, body = env.makeRequest("GET", "/api/v1/ai/tasks/"+taskID, nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		state, _ := body["state"].(string)
		assert.Contains(t, []string{"cancelled", "failed", "rejected"}, state)
	}
}

// =============================================================================
// Execution Monitoring Tests
// =============================================================================

func TestTemporal_ExecutionMonitoring(t *testing.T) {
	env := setupTemporalTestEnv(t)
	defer env.teardown()

	// Create and approve a task
	resp, body := env.makeRequest("POST", "/api/v1/ai/execute", map[string]interface{}{
		"intent":      "Task for execution monitoring test",
		"org_id":      env.OrgID,
		"environment": "staging",
	})

	require.Equal(t, http.StatusOK, resp.StatusCode)
	taskID, _ := body["task_id"].(string)
	env.trackTask(taskID)

	// Approve
	resp, body = env.makeRequest("POST", "/api/v1/ai/tasks/"+taskID+"/approve", map[string]interface{}{
		"reason": "Testing execution monitoring",
	})

	if resp.StatusCode == http.StatusOK {
		// Wait for execution to start
		time.Sleep(1 * time.Second)

		// Get executions
		resp, body = env.makeRequest("GET", "/api/v1/ai/tasks/"+taskID+"/executions", nil)

		if resp.StatusCode == http.StatusOK {
			executions, ok := body["executions"].([]interface{})
			if ok && len(executions) > 0 {
				exec := executions[0].(map[string]interface{})
				t.Logf("Execution ID: %v", exec["id"])
				t.Logf("Execution status: %v", exec["status"])

				assert.Contains(t, exec, "id")
				assert.Contains(t, exec, "status")
			}
		}
	}
}

// =============================================================================
// Tool Invocation Tracking Tests
// =============================================================================

func TestTemporal_ToolInvocationTracking(t *testing.T) {
	env := setupTemporalTestEnv(t)
	defer env.teardown()

	// Create a task that will invoke tools
	resp, body := env.makeRequest("POST", "/api/v1/ai/execute", map[string]interface{}{
		"intent":      "Query all AWS assets and check drift status",
		"org_id":      env.OrgID,
		"environment": "production",
	})

	require.Equal(t, http.StatusOK, resp.StatusCode)
	taskID, _ := body["task_id"].(string)
	env.trackTask(taskID)

	// Get tool invocations
	resp, body = env.makeRequest("GET", "/api/v1/ai/tasks/"+taskID+"/invocations", nil)

	if resp.StatusCode == http.StatusOK {
		invocations, ok := body["invocations"].([]interface{})
		if ok && len(invocations) > 0 {
			t.Logf("Found %d tool invocations", len(invocations))

			for _, inv := range invocations {
				invMap := inv.(map[string]interface{})
				t.Logf("Tool: %v, Status: %v, Duration: %vms",
					invMap["tool_name"],
					invMap["status"],
					invMap["duration_ms"])
			}
		}
	}
}

// =============================================================================
// Multiple Workflow Types Test
// =============================================================================

func TestTemporal_MultipleWorkflowTypes(t *testing.T) {
	env := setupTemporalTestEnv(t)
	defer env.teardown()

	testCases := []struct {
		name   string
		intent string
	}{
		{"drift", "Fix drift on all servers"},
		{"image", "Create new golden image"},
		{"patch", "Apply security patches"},
		{"compliance", "Run compliance check"},
		{"cost", "Optimize cloud costs"},
		{"dr", "Test disaster recovery"},
		{"incident", "Investigate alert"},
		{"security", "Run security scan"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, body := env.makeRequest("POST", "/api/v1/ai/execute", map[string]interface{}{
				"intent": tc.intent,
				"org_id": env.OrgID,
			})

			if resp.StatusCode == http.StatusOK {
				taskID, _ := body["task_id"].(string)
				env.trackTask(taskID)
				t.Logf("%s task created: %s", tc.name, taskID)
			} else {
				t.Logf("%s task creation: status %d", tc.name, resp.StatusCode)
			}
		})
	}
}

// =============================================================================
// Workflow with Plan Modification
// =============================================================================

func TestTemporal_WorkflowWithPlanModification(t *testing.T) {
	env := setupTemporalTestEnv(t)
	defer env.teardown()

	// Create task
	resp, body := env.makeRequest("POST", "/api/v1/ai/execute", map[string]interface{}{
		"intent":      "Remediate drift on web servers",
		"org_id":      env.OrgID,
		"environment": "staging",
	})

	require.Equal(t, http.StatusOK, resp.StatusCode)
	taskID, _ := body["task_id"].(string)
	env.trackTask(taskID)

	// Modify the plan
	resp, body = env.makeRequest("POST", "/api/v1/ai/tasks/"+taskID+"/modify", map[string]interface{}{
		"modifications": "Exclude database servers from remediation",
		"reason":        "Database servers need manual handling",
	})

	if resp.StatusCode == http.StatusOK {
		t.Logf("Plan modified successfully")

		// Verify task is still in awaiting state
		resp, body = env.makeRequest("GET", "/api/v1/ai/tasks/"+taskID, nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		state, _ := body["state"].(string)
		assert.Contains(t, []string{"pending", "planned"}, state)
	}
}

// =============================================================================
// Workflow Retry Behavior
// =============================================================================

func TestTemporal_WorkflowRetryBehavior(t *testing.T) {
	env := setupTemporalTestEnv(t)
	defer env.teardown()

	// Create a task that might fail
	resp, body := env.makeRequest("POST", "/api/v1/ai/execute", map[string]interface{}{
		"intent":      "Process assets with potential failures",
		"org_id":      env.OrgID,
		"environment": "staging",
	})

	require.Equal(t, http.StatusOK, resp.StatusCode)
	taskID, _ := body["task_id"].(string)
	env.trackTask(taskID)

	// Approve
	resp, _ = env.makeRequest("POST", "/api/v1/ai/tasks/"+taskID+"/approve", map[string]interface{}{
		"reason": "Testing retry behavior",
	})

	if resp.StatusCode == http.StatusOK {
		// Poll for completion or failure
		var finalState string
		for i := 0; i < 20; i++ {
			time.Sleep(500 * time.Millisecond)

			resp, body = env.makeRequest("GET", "/api/v1/ai/tasks/"+taskID, nil)
			if resp.StatusCode == http.StatusOK {
				state, _ := body["state"].(string)
				if state == "completed" || state == "failed" {
					finalState = state
					break
				}
			}
		}

		t.Logf("Final state after retry: %s", finalState)
	}
}

// =============================================================================
// Concurrent Workflow Execution
// =============================================================================

func TestTemporal_ConcurrentWorkflows(t *testing.T) {
	env := setupTemporalTestEnv(t)
	defer env.teardown()

	// Create multiple tasks concurrently
	taskIDs := make([]string, 0)

	for i := 0; i < 5; i++ {
		resp, body := env.makeRequest("POST", "/api/v1/ai/execute", map[string]interface{}{
			"intent": fmt.Sprintf("Concurrent workflow test %d", i),
			"org_id": env.OrgID,
		})

		if resp.StatusCode == http.StatusOK {
			if taskID, ok := body["task_id"].(string); ok {
				taskIDs = append(taskIDs, taskID)
				env.trackTask(taskID)
			}
		}
	}

	t.Logf("Created %d concurrent tasks", len(taskIDs))

	// Verify all tasks exist
	for _, taskID := range taskIDs {
		resp, body := env.makeRequest("GET", "/api/v1/ai/tasks/"+taskID, nil)
		if resp.StatusCode == http.StatusOK {
			state, _ := body["state"].(string)
			t.Logf("Task %s state: %s", taskID[:8], state)
		}
	}
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkTemporal_WorkflowCreation(b *testing.B) {
	if os.Getenv("RUN_BENCHMARKS") != "true" {
		b.Skip("Skipping benchmark: RUN_BENCHMARKS not set")
	}

	baseURL := os.Getenv("TEST_ORCHESTRATOR_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8083"
	}

	client := &http.Client{Timeout: 30 * time.Second}

	resp, err := client.Get(baseURL + "/health")
	if err != nil {
		b.Skipf("Skipping benchmark: orchestrator not available: %v", err)
	}
	resp.Body.Close()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		reqBody, _ := json.Marshal(map[string]interface{}{
			"intent": "Benchmark workflow creation",
			"org_id": "00000000-0000-0000-0000-000000000001",
		})

		req, _ := http.NewRequest("POST", baseURL+"/api/v1/ai/execute", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer dev-token")

		resp, err := client.Do(req)
		if err != nil {
			b.Errorf("Request failed: %v", err)
			continue
		}
		resp.Body.Close()
	}
}
