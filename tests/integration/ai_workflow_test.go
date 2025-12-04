//go:build integration

// Package integration contains end-to-end integration tests for QL-RF services.
// These tests verify the complete AI workflow from task creation to execution.
// Run with: go test -tags=integration -v -timeout 10m ./tests/integration/... -run TestAI
//
// These are blackbox tests that interact with the API via HTTP, testing the full
// workflow without importing internal packages.
package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getEnvOrDefaultAI returns env value or default for AI workflow tests
func getEnvOrDefaultAI(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

// =============================================================================
// AI Workflow Test Environment
// =============================================================================

// AIWorkflowTestEnv holds the test environment for AI workflow tests.
// This is a blackbox test environment that tests via HTTP endpoints.
type AIWorkflowTestEnv struct {
	BaseURL      string
	OrgID        string
	UserID       string
	SecondUserID string // For dual approval tests
	CreatedTasks []string
	mu           sync.Mutex
}

// setupAIWorkflowTestEnv creates a blackbox test environment.
// It requires a running orchestrator service at the configured URL.
func setupAIWorkflowTestEnv(t *testing.T) *AIWorkflowTestEnv {
	t.Helper()

	// Get orchestrator URL from environment or use default
	baseURL := os.Getenv("TEST_ORCHESTRATOR_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8083"
	}

	// Test connectivity
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(baseURL + "/health")
	if err != nil {
		t.Skipf("Skipping AI workflow test: orchestrator not available at %s: %v", baseURL, err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Skipf("Skipping AI workflow test: orchestrator health check failed with status %d", resp.StatusCode)
	}

	env := &AIWorkflowTestEnv{
		BaseURL:      baseURL,
		OrgID:        getEnvOrDefaultAI("TEST_ORG_ID", "00000000-0000-0000-0000-000000000001"),
		UserID:       "test-user-" + uuid.New().String()[:8],
		SecondUserID: "test-user-" + uuid.New().String()[:8],
		CreatedTasks: make([]string, 0),
	}

	return env
}


// trackTask adds a task ID to the cleanup list.
func (env *AIWorkflowTestEnv) trackTask(taskID string) {
	env.mu.Lock()
	defer env.mu.Unlock()
	env.CreatedTasks = append(env.CreatedTasks, taskID)
}

// teardown cleans up the test environment.
// Note: In blackbox tests, we don't clean up via DB. Tasks are left for manual cleanup or
// will be cleaned up by subsequent test runs.
func (env *AIWorkflowTestEnv) teardown() {
	// No cleanup needed - blackbox tests don't have DB access
}

// makeRequest is a helper to make HTTP requests.
func (env *AIWorkflowTestEnv) makeRequest(method, path string, body interface{}) (*http.Response, map[string]interface{}) {
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

	client := &http.Client{Timeout: 120 * time.Second} // LLM calls can take a while
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

// =============================================================================
// Task Lifecycle Tests
// =============================================================================

func TestAIWorkflow_FullLifecycle(t *testing.T) {
	env := setupAIWorkflowTestEnv(t)
	defer env.teardown()

	t.Run("CreateTask_DriftRemediation", func(t *testing.T) {
		resp, body := env.makeRequest("POST", "/api/v1/ai/execute", map[string]interface{}{
			"intent":      "Fix configuration drift on production web servers",
			"org_id":      env.OrgID,
			"environment": "production",
			"context": map[string]interface{}{
				"priority":   "high",
				"fleet_size": 50,
			},
		})

		require.NotNil(t, resp, "Request failed - response is nil (timeout or connection error)")
		require.Equal(t, http.StatusOK, resp.StatusCode, "Expected 200 OK, got %d: %v", resp.StatusCode, body)

		taskID, ok := body["task_id"].(string)
		require.True(t, ok, "task_id should be present in response")
		require.NotEmpty(t, taskID)
		env.trackTask(taskID)

		// Verify task structure
		assert.Contains(t, body, "task_spec")
		assert.Contains(t, body, "requires_hitl")

		taskSpec, ok := body["task_spec"].(map[string]interface{})
		require.True(t, ok)
		assert.Contains(t, taskSpec, "task_type")
		assert.Contains(t, taskSpec, "risk_level")
	})

	t.Run("CreateTask_ImageManagement", func(t *testing.T) {
		resp, body := env.makeRequest("POST", "/api/v1/ai/execute", map[string]interface{}{
			"intent":      "Audit and update golden images for Ubuntu 22.04",
			"org_id":      env.OrgID,
			"environment": "staging",
		})

		require.Equal(t, http.StatusOK, resp.StatusCode)

		taskID, ok := body["task_id"].(string)
		require.True(t, ok)
		env.trackTask(taskID)
	})

	t.Run("CreateTask_CostOptimization", func(t *testing.T) {
		resp, body := env.makeRequest("POST", "/api/v1/ai/execute", map[string]interface{}{
			"intent":      "Identify cost savings opportunities across all cloud platforms",
			"org_id":      env.OrgID,
			"environment": "all",
		})

		require.Equal(t, http.StatusOK, resp.StatusCode)

		taskID, ok := body["task_id"].(string)
		require.True(t, ok)
		env.trackTask(taskID)
	})
}

func TestAIWorkflow_TaskStateTransitions(t *testing.T) {
	env := setupAIWorkflowTestEnv(t)
	defer env.teardown()

	// Step 1: Create a task
	resp, body := env.makeRequest("POST", "/api/v1/ai/execute", map[string]interface{}{
		"intent":      "Patch all staging servers to latest security updates",
		"org_id":      env.OrgID,
		"environment": "staging",
	})

	require.Equal(t, http.StatusOK, resp.StatusCode)
	taskID := body["task_id"].(string)
	env.trackTask(taskID)

	// Step 2: Verify task is in "planned" state
	resp, body = env.makeRequest("GET", "/api/v1/ai/tasks/"+taskID, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	state, _ := body["state"].(string)
	assert.Contains(t, []string{"pending", "planned"}, state)

	// Step 3: Approve the task
	resp, body = env.makeRequest("POST", "/api/v1/ai/tasks/"+taskID+"/approve", map[string]interface{}{
		"reason": "Approved for staging deployment",
	})

	if resp.StatusCode == http.StatusOK {
		assert.Equal(t, "approved", body["status"])

		// Step 4: Verify state changed
		resp, body = env.makeRequest("GET", "/api/v1/ai/tasks/"+taskID, nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		state, _ := body["state"].(string)
		assert.Contains(t, []string{"approved", "executing", "completed"}, state)
	}
}

func TestAIWorkflow_TaskRejection(t *testing.T) {
	env := setupAIWorkflowTestEnv(t)
	defer env.teardown()

	// Create a high-risk task
	resp, body := env.makeRequest("POST", "/api/v1/ai/execute", map[string]interface{}{
		"intent":      "Delete all unused resources in production",
		"org_id":      env.OrgID,
		"environment": "production",
	})

	require.Equal(t, http.StatusOK, resp.StatusCode)
	taskID := body["task_id"].(string)
	env.trackTask(taskID)

	// Reject the task
	resp, body = env.makeRequest("POST", "/api/v1/ai/tasks/"+taskID+"/reject", map[string]interface{}{
		"reason": "Too risky for production - requires manual review",
	})

	if resp.StatusCode == http.StatusOK {
		assert.Equal(t, "rejected", body["status"])

		// Verify rejection reason is recorded
		resp, body = env.makeRequest("GET", "/api/v1/ai/tasks/"+taskID, nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "rejected", body["state"])
	}
}

// =============================================================================
// Multi-Agent Scenario Tests
// =============================================================================

func TestAIWorkflow_MultipleAgentTypes(t *testing.T) {
	env := setupAIWorkflowTestEnv(t)
	defer env.teardown()

	testCases := []struct {
		name           string
		intent         string
		expectedAgent  string
		expectedType   string
	}{
		{
			name:          "drift_agent",
			intent:        "Show me all drifted assets in AWS",
			expectedAgent: "drift",
			expectedType:  "drift",
		},
		{
			name:          "image_agent",
			intent:        "Create a new golden image for web servers",
			expectedAgent: "image",
			expectedType:  "image",
		},
		{
			name:          "patch_agent",
			intent:        "Roll out security patches to all servers",
			expectedAgent: "patch",
			expectedType:  "patch",
		},
		{
			name:          "compliance_agent",
			intent:        "Run CIS benchmark compliance check",
			expectedAgent: "compliance",
			expectedType:  "compliance",
		},
		{
			name:          "cost_agent",
			intent:        "Analyze cloud spending and find savings",
			expectedAgent: "cost",
			expectedType:  "cost",
		},
		{
			name:          "dr_agent",
			intent:        "Test disaster recovery failover",
			expectedAgent: "dr",
			expectedType:  "dr",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, body := env.makeRequest("POST", "/api/v1/ai/execute", map[string]interface{}{
				"intent": tc.intent,
				"org_id": env.OrgID,
			})

			require.Equal(t, http.StatusOK, resp.StatusCode, "Failed for %s: %v", tc.name, body)

			taskID, ok := body["task_id"].(string)
			require.True(t, ok)
			env.trackTask(taskID)

			// Verify agent routing
			if taskSpec, ok := body["task_spec"].(map[string]interface{}); ok {
				taskType, _ := taskSpec["task_type"].(string)
				assert.Contains(t, taskType, tc.expectedType,
					"Expected task type containing %q, got %q", tc.expectedType, taskType)
			}
		})
	}
}

func TestAIWorkflow_ConcurrentTasks(t *testing.T) {
	env := setupAIWorkflowTestEnv(t)
	defer env.teardown()

	// Launch multiple tasks concurrently
	var wg sync.WaitGroup
	results := make(chan struct {
		taskID string
		err    error
	}, 5)

	intents := []string{
		"Fix drift on web servers",
		"Update golden images",
		"Check compliance status",
		"Optimize cloud costs",
		"Test DR failover",
	}

	for _, intent := range intents {
		wg.Add(1)
		go func(intent string) {
			defer wg.Done()

			resp, body := env.makeRequest("POST", "/api/v1/ai/execute", map[string]interface{}{
				"intent": intent,
				"org_id": env.OrgID,
			})

			if resp.StatusCode == http.StatusOK {
				if taskID, ok := body["task_id"].(string); ok {
					env.trackTask(taskID)
					results <- struct {
						taskID string
						err    error
					}{taskID: taskID, err: nil}
					return
				}
			}
			results <- struct {
				taskID string
				err    error
			}{err: fmt.Errorf("failed to create task: %v", body)}
		}(intent)
	}

	wg.Wait()
	close(results)

	// Verify all tasks were created
	successCount := 0
	for result := range results {
		if result.err == nil {
			successCount++
		}
	}

	assert.GreaterOrEqual(t, successCount, 3, "Expected at least 3 concurrent tasks to succeed")
}

// =============================================================================
// Tool Invocation Tests
// =============================================================================

func TestAIWorkflow_ToolInvocationAudit(t *testing.T) {
	env := setupAIWorkflowTestEnv(t)
	defer env.teardown()

	// Create and approve a task to trigger tool invocations
	resp, body := env.makeRequest("POST", "/api/v1/ai/execute", map[string]interface{}{
		"intent": "Query all AWS assets and check their drift status",
		"org_id": env.OrgID,
	})

	require.Equal(t, http.StatusOK, resp.StatusCode)
	taskID := body["task_id"].(string)
	env.trackTask(taskID)

	// Get tool invocations for the task
	resp, body = env.makeRequest("GET", "/api/v1/ai/tasks/"+taskID+"/invocations", nil)

	if resp.StatusCode == http.StatusOK {
		invocations, ok := body["invocations"].([]interface{})
		require.True(t, ok)

		// Verify invocation structure
		for _, inv := range invocations {
			invMap := inv.(map[string]interface{})
			assert.Contains(t, invMap, "tool_name")
			assert.Contains(t, invMap, "invoked_at")
			assert.Contains(t, invMap, "duration_ms")
			assert.Contains(t, invMap, "status")
		}
	}
}

// =============================================================================
// Dual Approval Tests
// =============================================================================

func TestAIWorkflow_DualApproval(t *testing.T) {
	env := setupAIWorkflowTestEnv(t)
	defer env.teardown()

	// Create a high-risk production task (should require dual approval)
	resp, body := env.makeRequest("POST", "/api/v1/ai/execute", map[string]interface{}{
		"intent":      "Deploy critical security patches to all production databases",
		"org_id":      env.OrgID,
		"environment": "production",
		"context": map[string]interface{}{
			"risk_level": "critical",
		},
	})

	require.Equal(t, http.StatusOK, resp.StatusCode)
	taskID := body["task_id"].(string)
	env.trackTask(taskID)

	// Check if dual approval is required
	resp, body = env.makeRequest("GET", "/api/v1/ai/tasks/"+taskID, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// First approval
	resp, body = env.makeRequest("POST", "/api/v1/ai/tasks/"+taskID+"/approve", map[string]interface{}{
		"reason":  "First approval - verified plan",
		"user_id": env.UserID,
	})

	if resp.StatusCode == http.StatusOK {
		// Check approval status
		approvalStatus, ok := body["approval_status"].(map[string]interface{})
		if ok {
			received, _ := approvalStatus["approvals_received"].(float64)
			required, _ := approvalStatus["approvals_required"].(float64)

			if required > 1 {
				// Need second approval
				assert.Less(t, received, required, "Should need more approvals")

				// Second approval with different user
				req, _ := http.NewRequest("POST", env.BaseURL+"/api/v1/ai/tasks/"+taskID+"/approve",
					bytes.NewReader([]byte(`{"reason": "Second approval - ready for execution"}`)))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer dev-token")
				req.Header.Set("X-User-ID", env.SecondUserID)
				req.Header.Set("X-Org-ID", env.OrgID)

				client := &http.Client{Timeout: 120 * time.Second} // LLM calls can take a while
				resp2, err := client.Do(req)
				if err != nil {
					t.Logf("Second approval request failed: %v", err)
				} else {
					defer resp2.Body.Close()

					if resp2.StatusCode == http.StatusOK {
						// Verify task proceeds to execution
						resp, body = env.makeRequest("GET", "/api/v1/ai/tasks/"+taskID, nil)
						state, _ := body["state"].(string)
						assert.Contains(t, []string{"approved", "executing", "completed"}, state)
					}
				}
			}
		}
	}
}

// =============================================================================
// Plan Modification Tests
// =============================================================================

func TestAIWorkflow_PlanModification(t *testing.T) {
	env := setupAIWorkflowTestEnv(t)
	defer env.teardown()

	// Create a task
	resp, body := env.makeRequest("POST", "/api/v1/ai/execute", map[string]interface{}{
		"intent": "Remediate drift on development servers",
		"org_id": env.OrgID,
	})

	require.Equal(t, http.StatusOK, resp.StatusCode)
	taskID := body["task_id"].(string)
	env.trackTask(taskID)

	// Modify the plan
	resp, body = env.makeRequest("POST", "/api/v1/ai/tasks/"+taskID+"/modify", map[string]interface{}{
		"modifications": "Exclude server dev-db-01 from remediation",
		"reason":        "Database server requires manual handling",
	})

	if resp.StatusCode == http.StatusOK {
		assert.Contains(t, body, "modified_plan")

		// Verify task state
		resp, body = env.makeRequest("GET", "/api/v1/ai/tasks/"+taskID, nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		// Should still be in planned state waiting for approval
		state, _ := body["state"].(string)
		assert.Contains(t, []string{"pending", "planned"}, state)
	}
}

// =============================================================================
// Quality Score Tests
// =============================================================================

func TestAIWorkflow_QualityScore(t *testing.T) {
	env := setupAIWorkflowTestEnv(t)
	defer env.teardown()

	// Create tasks with different complexity
	resp, body := env.makeRequest("POST", "/api/v1/ai/execute", map[string]interface{}{
		"intent": "Run comprehensive compliance audit across all environments",
		"org_id": env.OrgID,
	})

	require.Equal(t, http.StatusOK, resp.StatusCode)
	taskID := body["task_id"].(string)
	env.trackTask(taskID)

	// Get task details with quality score
	resp, body = env.makeRequest("GET", "/api/v1/ai/tasks/"+taskID, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Check for quality score if present
	if plan, ok := body["plan"].(map[string]interface{}); ok {
		if qualityScore, ok := plan["quality_score"].(map[string]interface{}); ok {
			// Verify quality score structure
			assert.Contains(t, qualityScore, "overall")
			assert.Contains(t, qualityScore, "completeness")
			assert.Contains(t, qualityScore, "safety")
			assert.Contains(t, qualityScore, "efficiency")

			// Overall score should be between 0 and 100
			overall, _ := qualityScore["overall"].(float64)
			assert.GreaterOrEqual(t, overall, float64(0))
			assert.LessOrEqual(t, overall, float64(100))
		}
	}
}

// =============================================================================
// Error Handling Tests
// =============================================================================

func TestAIWorkflow_ErrorHandling(t *testing.T) {
	env := setupAIWorkflowTestEnv(t)
	defer env.teardown()

	t.Run("InvalidTaskID", func(t *testing.T) {
		resp, body := env.makeRequest("GET", "/api/v1/ai/tasks/invalid-uuid", nil)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		assert.Contains(t, body, "error")
	})

	t.Run("MissingIntent", func(t *testing.T) {
		resp, body := env.makeRequest("POST", "/api/v1/ai/execute", map[string]interface{}{
			"org_id": env.OrgID,
		})
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		assert.Contains(t, body, "error")
	})

	t.Run("MissingOrgID", func(t *testing.T) {
		resp, body := env.makeRequest("POST", "/api/v1/ai/execute", map[string]interface{}{
			"intent": "Some intent",
		})
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		assert.Contains(t, body, "error")
	})

	t.Run("ApproveNonexistentTask", func(t *testing.T) {
		resp, _ := env.makeRequest("POST", "/api/v1/ai/tasks/00000000-0000-0000-0000-000000000000/approve", map[string]interface{}{
			"reason": "test",
		})
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("DoubleApproval", func(t *testing.T) {
		// Create and approve a task
		resp, body := env.makeRequest("POST", "/api/v1/ai/execute", map[string]interface{}{
			"intent": "Simple drift check",
			"org_id": env.OrgID,
		})
		require.Equal(t, http.StatusOK, resp.StatusCode)
		taskID := body["task_id"].(string)
		env.trackTask(taskID)

		// First approval
		resp, _ = env.makeRequest("POST", "/api/v1/ai/tasks/"+taskID+"/approve", map[string]interface{}{
			"reason": "First approval",
		})

		if resp.StatusCode == http.StatusOK {
			// Try to approve again (may be rejected or idempotent)
			resp, body = env.makeRequest("POST", "/api/v1/ai/tasks/"+taskID+"/approve", map[string]interface{}{
				"reason": "Duplicate approval",
			})

			// Should either succeed (idempotent) or return error
			assert.Contains(t, []int{http.StatusOK, http.StatusBadRequest, http.StatusConflict}, resp.StatusCode)
		}
	})
}

// =============================================================================
// Task Listing and Filtering Tests
// =============================================================================

func TestAIWorkflow_TaskListing(t *testing.T) {
	env := setupAIWorkflowTestEnv(t)
	defer env.teardown()

	// Create multiple tasks
	intents := []string{
		"Check drift status",
		"Update images",
		"Run compliance audit",
	}

	for _, intent := range intents {
		resp, body := env.makeRequest("POST", "/api/v1/ai/execute", map[string]interface{}{
			"intent": intent,
			"org_id": env.OrgID,
		})
		if resp.StatusCode == http.StatusOK {
			if taskID, ok := body["task_id"].(string); ok {
				env.trackTask(taskID)
			}
		}
	}

	t.Run("ListAllTasks", func(t *testing.T) {
		resp, body := env.makeRequest("GET", "/api/v1/ai/tasks", nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		tasks, ok := body["tasks"].([]interface{})
		require.True(t, ok)
		assert.GreaterOrEqual(t, len(tasks), 1)

		assert.Contains(t, body, "total")
	})

	t.Run("ListByState", func(t *testing.T) {
		states := []string{"pending", "planned", "approved", "executing", "completed", "failed", "rejected"}

		for _, state := range states {
			resp, body := env.makeRequest("GET", fmt.Sprintf("/api/v1/ai/tasks?state=%s", state), nil)
			require.Equal(t, http.StatusOK, resp.StatusCode)

			tasks, ok := body["tasks"].([]interface{})
			require.True(t, ok)

			// All returned tasks should have the requested state
			for _, task := range tasks {
				taskMap := task.(map[string]interface{})
				taskState, _ := taskMap["state"].(string)
				assert.Equal(t, state, taskState)
			}
		}
	})

	t.Run("ListWithPagination", func(t *testing.T) {
		resp, body := env.makeRequest("GET", "/api/v1/ai/tasks?limit=2&offset=0", nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		tasks, ok := body["tasks"].([]interface{})
		require.True(t, ok)
		assert.LessOrEqual(t, len(tasks), 2)
	})
}

// =============================================================================
// Agent and Tool Metadata Tests
// =============================================================================

func TestAIWorkflow_AgentMetadata(t *testing.T) {
	env := setupAIWorkflowTestEnv(t)
	defer env.teardown()

	resp, body := env.makeRequest("GET", "/api/v1/ai/agents", nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	agents, ok := body["agents"].([]interface{})
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(agents), 4, "Expected at least 4 agents")

	// Verify agent structure
	for _, a := range agents {
		agent := a.(map[string]interface{})
		assert.Contains(t, agent, "name")
		assert.Contains(t, agent, "description")
		assert.Contains(t, agent, "capabilities")
	}

	// Check for specific agents
	agentNames := make([]string, 0)
	for _, a := range agents {
		agent := a.(map[string]interface{})
		if name, ok := agent["name"].(string); ok {
			agentNames = append(agentNames, name)
		}
	}

	expectedAgents := []string{"drift", "image", "patch", "compliance"}
	for _, expected := range expectedAgents {
		found := false
		for _, actual := range agentNames {
			if actual == expected || actual == expected+"_agent" {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected agent %s not found in %v", expected, agentNames)
	}
}

func TestAIWorkflow_ToolMetadata(t *testing.T) {
	env := setupAIWorkflowTestEnv(t)
	defer env.teardown()

	resp, body := env.makeRequest("GET", "/api/v1/ai/tools", nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	tools, ok := body["tools"].([]interface{})
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(tools), 10, "Expected at least 10 tools")

	// Verify tool structure
	for _, toolItem := range tools {
		tool := toolItem.(map[string]interface{})
		assert.Contains(t, tool, "name")
		assert.Contains(t, tool, "description")
		assert.Contains(t, tool, "category")
	}

	// Check for critical tools
	toolNames := make([]string, 0)
	for _, toolItem := range tools {
		tool := toolItem.(map[string]interface{})
		if name, ok := tool["name"].(string); ok {
			toolNames = append(toolNames, name)
		}
	}

	criticalTools := []string{"query_assets", "get_drift_status", "analyze_drift"}
	for _, expected := range criticalTools {
		found := false
		for _, actual := range toolNames {
			if actual == expected {
				found = true
				break
			}
		}
		assert.True(t, found, "Critical tool %s not found", expected)
	}
}

// =============================================================================
// Usage Statistics Tests
// =============================================================================

func TestAIWorkflow_UsageStats(t *testing.T) {
	env := setupAIWorkflowTestEnv(t)
	defer env.teardown()

	// Create some tasks to generate usage
	for i := 0; i < 3; i++ {
		resp, body := env.makeRequest("POST", "/api/v1/ai/execute", map[string]interface{}{
			"intent": fmt.Sprintf("Task %d for usage stats", i),
			"org_id": env.OrgID,
		})
		if resp.StatusCode == http.StatusOK {
			if taskID, ok := body["task_id"].(string); ok {
				env.trackTask(taskID)
			}
		}
	}

	// Get usage stats
	resp, body := env.makeRequest("GET", "/api/v1/ai/usage", nil)

	if resp.StatusCode == http.StatusOK {
		// Verify usage stats structure
		assert.Contains(t, body, "total_tasks")
		assert.Contains(t, body, "total_tokens")

		// Token counts should be non-negative
		if tokens, ok := body["total_tokens"].(float64); ok {
			assert.GreaterOrEqual(t, tokens, float64(0))
		}
	}
}

// =============================================================================
// Settings Tests
// =============================================================================

func TestAIWorkflow_Settings(t *testing.T) {
	env := setupAIWorkflowTestEnv(t)
	defer env.teardown()

	t.Run("GetSettings", func(t *testing.T) {
		resp, body := env.makeRequest("GET", "/api/v1/ai/settings", nil)

		if resp.StatusCode == http.StatusOK {
			assert.Contains(t, body, "autonomy_mode")
			assert.Contains(t, body, "hitl_required")
		}
	})

	t.Run("UpdateSettings", func(t *testing.T) {
		resp, body := env.makeRequest("PUT", "/api/v1/ai/settings", map[string]interface{}{
			"autonomy_mode":       "supervised",
			"hitl_required":       true,
			"max_tokens_per_task": 10000,
		})

		if resp.StatusCode == http.StatusOK {
			assert.Equal(t, "supervised", body["autonomy_mode"])
			assert.Equal(t, true, body["hitl_required"])
		}
	})
}

// =============================================================================
// Execution Monitoring Tests
// =============================================================================

func TestAIWorkflow_ExecutionMonitoring(t *testing.T) {
	env := setupAIWorkflowTestEnv(t)
	defer env.teardown()

	// Create and approve a task
	resp, body := env.makeRequest("POST", "/api/v1/ai/execute", map[string]interface{}{
		"intent":      "Remediate drift on staging servers",
		"org_id":      env.OrgID,
		"environment": "staging",
	})

	require.Equal(t, http.StatusOK, resp.StatusCode)
	taskID := body["task_id"].(string)
	env.trackTask(taskID)

	// Approve
	resp, _ = env.makeRequest("POST", "/api/v1/ai/tasks/"+taskID+"/approve", map[string]interface{}{
		"reason": "Approved for testing",
	})

	if resp.StatusCode == http.StatusOK {
		// Get executions for the task
		resp, body = env.makeRequest("GET", "/api/v1/ai/tasks/"+taskID+"/executions", nil)

		if resp.StatusCode == http.StatusOK {
			executions, ok := body["executions"].([]interface{})
			require.True(t, ok)

			if len(executions) > 0 {
				exec := executions[0].(map[string]interface{})
				assert.Contains(t, exec, "id")
				assert.Contains(t, exec, "status")
				assert.Contains(t, exec, "started_at")
			}
		}
	}
}

// =============================================================================
// Cancellation Tests
// =============================================================================

func TestAIWorkflow_TaskCancellation(t *testing.T) {
	env := setupAIWorkflowTestEnv(t)
	defer env.teardown()

	// Create a task
	resp, body := env.makeRequest("POST", "/api/v1/ai/execute", map[string]interface{}{
		"intent": "Long running compliance audit",
		"org_id": env.OrgID,
	})

	require.Equal(t, http.StatusOK, resp.StatusCode)
	taskID := body["task_id"].(string)
	env.trackTask(taskID)

	// Cancel the task
	resp, body = env.makeRequest("POST", "/api/v1/ai/tasks/"+taskID+"/cancel", map[string]interface{}{
		"reason": "User requested cancellation",
	})

	if resp.StatusCode == http.StatusOK {
		// Verify state
		resp, body = env.makeRequest("GET", "/api/v1/ai/tasks/"+taskID, nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		state, _ := body["state"].(string)
		assert.Contains(t, []string{"cancelled", "failed"}, state)
	}
}

// =============================================================================
// CORS Tests
// =============================================================================

func TestAIWorkflow_CORS(t *testing.T) {
	env := setupAIWorkflowTestEnv(t)
	defer env.teardown()

	// Test preflight request
	req, _ := http.NewRequest("OPTIONS", env.BaseURL+"/api/v1/ai/tasks", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type, Authorization")

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NotEmpty(t, resp.Header.Get("Access-Control-Allow-Origin"))
	assert.NotEmpty(t, resp.Header.Get("Access-Control-Allow-Methods"))
}

// =============================================================================
// Rate Limiting Tests (if implemented)
// =============================================================================

func TestAIWorkflow_RateLimiting(t *testing.T) {
	env := setupAIWorkflowTestEnv(t)
	defer env.teardown()

	// Rapid-fire requests
	var successCount, rateLimitedCount int

	for i := 0; i < 20; i++ {
		resp, body := env.makeRequest("POST", "/api/v1/ai/execute", map[string]interface{}{
			"intent": fmt.Sprintf("Rate limit test %d", i),
			"org_id": env.OrgID,
		})

		if resp.StatusCode == http.StatusOK {
			successCount++
			if taskID, ok := body["task_id"].(string); ok {
				env.trackTask(taskID)
			}
		} else if resp.StatusCode == http.StatusTooManyRequests {
			rateLimitedCount++
		}
	}

	// Should have at least some successes
	assert.Greater(t, successCount, 0, "Expected some requests to succeed")

	t.Logf("Success: %d, Rate limited: %d", successCount, rateLimitedCount)
}

// =============================================================================
// Integration with External Services
// =============================================================================

func TestAIWorkflow_LLMIntegration(t *testing.T) {
	env := setupAIWorkflowTestEnv(t)
	defer env.teardown()

	// Create a task and verify it completes with LLM-generated plan
	resp, body := env.makeRequest("POST", "/api/v1/ai/execute", map[string]interface{}{
		"intent": "Test LLM integration",
		"org_id": env.OrgID,
	})

	require.Equal(t, http.StatusOK, resp.StatusCode)

	taskID, ok := body["task_id"].(string)
	require.True(t, ok)
	env.trackTask(taskID)

	// Verify task was created with a plan
	if taskSpec, ok := body["task_spec"].(map[string]interface{}); ok {
		assert.NotEmpty(t, taskSpec)
	}
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkAIWorkflow_TaskCreation(b *testing.B) {
	// Skip if not running benchmarks explicitly
	if os.Getenv("RUN_BENCHMARKS") != "true" {
		b.Skip("Skipping benchmark: RUN_BENCHMARKS not set")
	}

	// Get orchestrator URL from environment or use default
	baseURL := os.Getenv("TEST_ORCHESTRATOR_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8083"
	}

	// Test connectivity
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(baseURL + "/health")
	if err != nil {
		b.Skipf("Skipping benchmark: orchestrator not available at %s: %v", baseURL, err)
	}
	resp.Body.Close()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		reqBody, _ := json.Marshal(map[string]interface{}{
			"intent": "Benchmark test task",
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
