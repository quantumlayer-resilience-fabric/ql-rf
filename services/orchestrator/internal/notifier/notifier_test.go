package notifier

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/quantumlayerhq/ql-rf/pkg/config"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/executor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	log := logger.New("debug", "json")
	cfg := config.NotificationConfig{
		SlackEnabled:    true,
		SlackWebhookURL: "https://hooks.slack.com/test",
		SlackChannel:    "#alerts",
	}

	n := New(cfg, log)

	assert.NotNil(t, n)
	assert.Equal(t, cfg.SlackEnabled, n.cfg.SlackEnabled)
	assert.Equal(t, cfg.SlackWebhookURL, n.cfg.SlackWebhookURL)
}

func TestNotify_Webhook(t *testing.T) {
	// Create test server
	var receivedEvent Event
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		err := json.NewDecoder(r.Body).Decode(&receivedEvent)
		require.NoError(t, err)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	log := logger.New("debug", "json")
	cfg := config.NotificationConfig{
		WebhookEnabled: true,
		WebhookURL:     server.URL,
	}

	n := New(cfg, log)
	ctx := context.Background()

	event := Event{
		Type:        EventTaskPendingApproval,
		TaskID:      "task-123",
		TaskType:    "drift_remediation",
		Environment: "production",
		RiskLevel:   "high",
		Summary:     "Fix drift on prod servers",
	}

	err := n.Notify(ctx, event)
	require.NoError(t, err)

	assert.Equal(t, EventTaskPendingApproval, receivedEvent.Type)
	assert.Equal(t, "task-123", receivedEvent.TaskID)
	assert.Equal(t, "drift_remediation", receivedEvent.TaskType)
}

func TestNotify_WebhookFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	log := logger.New("debug", "json")
	cfg := config.NotificationConfig{
		WebhookEnabled: true,
		WebhookURL:     server.URL,
	}

	n := New(cfg, log)
	ctx := context.Background()

	err := n.Notify(ctx, Event{
		Type:   EventTaskApproved,
		TaskID: "task-456",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "webhook")
}

func TestNotifyTaskPendingApproval(t *testing.T) {
	var received Event
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	log := logger.New("debug", "json")
	cfg := config.NotificationConfig{
		WebhookEnabled: true,
		WebhookURL:     server.URL,
	}

	n := New(cfg, log)
	ctx := context.Background()

	err := n.NotifyTaskPendingApproval(ctx,
		"task-789",
		"patch_rollout",
		"production",
		"critical",
		"Apply security patches",
	)

	require.NoError(t, err)
	assert.Equal(t, EventTaskPendingApproval, received.Type)
	assert.Equal(t, "task-789", received.TaskID)
	assert.Equal(t, "patch_rollout", received.TaskType)
	assert.Equal(t, "production", received.Environment)
	assert.Equal(t, "critical", received.RiskLevel)
	assert.Equal(t, "Apply security patches", received.Summary)
}

func TestNotifyTaskApproved(t *testing.T) {
	var received Event
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	log := logger.New("debug", "json")
	cfg := config.NotificationConfig{
		WebhookEnabled: true,
		WebhookURL:     server.URL,
	}

	n := New(cfg, log)
	ctx := context.Background()

	err := n.NotifyTaskApproved(ctx, "task-approved", "admin@example.com")

	require.NoError(t, err)
	assert.Equal(t, EventTaskApproved, received.Type)
	assert.Equal(t, "task-approved", received.TaskID)
	assert.Equal(t, "admin@example.com", received.UserID)
}

func TestNotifyTaskRejected(t *testing.T) {
	var received Event
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	log := logger.New("debug", "json")
	cfg := config.NotificationConfig{
		WebhookEnabled: true,
		WebhookURL:     server.URL,
	}

	n := New(cfg, log)
	ctx := context.Background()

	err := n.NotifyTaskRejected(ctx, "task-rejected", "admin@example.com", "Risk too high")

	require.NoError(t, err)
	assert.Equal(t, EventTaskRejected, received.Type)
	assert.Equal(t, "task-rejected", received.TaskID)
	assert.Equal(t, "admin@example.com", received.UserID)
	assert.Equal(t, "Risk too high", received.Summary)
}

func TestNotifyExecutionStarted(t *testing.T) {
	var received Event
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	log := logger.New("debug", "json")
	cfg := config.NotificationConfig{
		WebhookEnabled: true,
		WebhookURL:     server.URL,
	}

	n := New(cfg, log)
	ctx := context.Background()

	exec := &executor.Execution{
		ID:          "exec-123",
		TaskID:      "task-123",
		Status:      executor.StatusRunning,
		TotalPhases: 3,
		StartedAt:   time.Now(),
	}

	err := n.NotifyExecutionStarted(ctx, exec)

	require.NoError(t, err)
	assert.Equal(t, EventExecutionStarted, received.Type)
	assert.Equal(t, "task-123", received.TaskID)
	assert.NotNil(t, received.Execution)
}

func TestNotifyExecutionCompleted(t *testing.T) {
	var received Event
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	log := logger.New("debug", "json")
	cfg := config.NotificationConfig{
		WebhookEnabled: true,
		WebhookURL:     server.URL,
	}

	n := New(cfg, log)
	ctx := context.Background()

	now := time.Now()
	exec := &executor.Execution{
		ID:          "exec-456",
		TaskID:      "task-456",
		Status:      executor.StatusCompleted,
		TotalPhases: 2,
		StartedAt:   now.Add(-time.Hour),
		CompletedAt: &now,
	}

	err := n.NotifyExecutionCompleted(ctx, exec)

	require.NoError(t, err)
	assert.Equal(t, EventExecutionCompleted, received.Type)
}

func TestNotifyExecutionFailed(t *testing.T) {
	var received Event
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	log := logger.New("debug", "json")
	cfg := config.NotificationConfig{
		WebhookEnabled: true,
		WebhookURL:     server.URL,
	}

	n := New(cfg, log)
	ctx := context.Background()

	exec := &executor.Execution{
		ID:     "exec-failed",
		TaskID: "task-failed",
		Status: executor.StatusFailed,
		Error:  "Connection timeout",
	}

	err := n.NotifyExecutionCompleted(ctx, exec)

	require.NoError(t, err)
	assert.Equal(t, EventExecutionFailed, received.Type)
}

func TestNotifyPhaseStarted(t *testing.T) {
	var received Event
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	log := logger.New("debug", "json")
	cfg := config.NotificationConfig{
		WebhookEnabled: true,
		WebhookURL:     server.URL,
	}

	n := New(cfg, log)
	ctx := context.Background()

	exec := &executor.Execution{
		ID:     "exec-phase",
		TaskID: "task-phase",
		Status: executor.StatusRunning,
	}

	now := time.Now()
	phase := &executor.PhaseExecution{
		Name:      "Canary",
		Status:    executor.PhaseStatusRunning,
		StartedAt: &now,
	}

	err := n.NotifyPhaseStarted(ctx, exec, phase)

	require.NoError(t, err)
	assert.Equal(t, EventPhaseStarted, received.Type)
	assert.NotNil(t, received.Phase)
}

func TestNotifyPhaseCompleted(t *testing.T) {
	var received Event
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	log := logger.New("debug", "json")
	cfg := config.NotificationConfig{
		WebhookEnabled: true,
		WebhookURL:     server.URL,
	}

	n := New(cfg, log)
	ctx := context.Background()

	exec := &executor.Execution{
		ID:     "exec-phase-done",
		TaskID: "task-phase-done",
		Status: executor.StatusRunning,
	}

	now := time.Now()
	phase := &executor.PhaseExecution{
		Name:        "Wave 1",
		Status:      executor.PhaseStatusCompleted,
		StartedAt:   &now,
		CompletedAt: &now,
	}

	err := n.NotifyPhaseCompleted(ctx, exec, phase)

	require.NoError(t, err)
	assert.Equal(t, EventPhaseCompleted, received.Type)
}

func TestNotifyPhaseFailed(t *testing.T) {
	var received Event
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	log := logger.New("debug", "json")
	cfg := config.NotificationConfig{
		WebhookEnabled: true,
		WebhookURL:     server.URL,
	}

	n := New(cfg, log)
	ctx := context.Background()

	exec := &executor.Execution{
		ID:     "exec-phase-fail",
		TaskID: "task-phase-fail",
		Status: executor.StatusFailed,
	}

	phase := &executor.PhaseExecution{
		Name:   "Wave 2",
		Status: executor.PhaseStatusFailed,
		Error:  "Health check failed",
	}

	err := n.NotifyPhaseCompleted(ctx, exec, phase)

	require.NoError(t, err)
	assert.Equal(t, EventPhaseFailed, received.Type)
}

func TestBuildSlackMessage(t *testing.T) {
	log := logger.New("debug", "json")
	cfg := config.NotificationConfig{
		SlackChannel: "#alerts",
	}

	n := New(cfg, log)

	tests := []struct {
		name      string
		event     Event
		wantTitle string
		wantColor string
	}{
		{
			name: "pending approval",
			event: Event{
				Type:        EventTaskPendingApproval,
				TaskID:      "task-123",
				TaskType:    "drift_remediation",
				Environment: "production",
				RiskLevel:   "high",
				Summary:     "Fix drift",
			},
			wantTitle: "Task Awaiting Approval",
			wantColor: "#FFA500",
		},
		{
			name: "approved",
			event: Event{
				Type:   EventTaskApproved,
				TaskID: "task-456",
				UserID: "admin",
			},
			wantTitle: "Task Approved",
			wantColor: "#36A64F",
		},
		{
			name: "rejected",
			event: Event{
				Type:    EventTaskRejected,
				TaskID:  "task-789",
				UserID:  "admin",
				Summary: "Too risky",
			},
			wantTitle: "Task Rejected",
			wantColor: "#FF0000",
		},
		{
			name: "execution started",
			event: Event{
				Type:   EventExecutionStarted,
				TaskID: "task-exec",
				Execution: &executor.Execution{
					TotalPhases: 3,
				},
			},
			wantTitle: "Execution Started",
			wantColor: "#36A64F",
		},
		{
			name: "execution completed",
			event: Event{
				Type:   EventExecutionCompleted,
				TaskID: "task-done",
			},
			wantTitle: "Execution Completed",
			wantColor: "#36A64F",
		},
		{
			name: "execution failed",
			event: Event{
				Type:   EventExecutionFailed,
				TaskID: "task-fail",
				Execution: &executor.Execution{
					Error: "Timeout",
				},
			},
			wantTitle: "Execution Failed",
			wantColor: "#FF0000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := n.buildSlackMessage(tt.event)

			assert.Equal(t, "#alerts", msg["channel"])
			assert.Equal(t, "QL-RF AI Orchestrator", msg["username"])

			attachments := msg["attachments"].([]map[string]interface{})
			assert.Len(t, attachments, 1)

			attachment := attachments[0]
			assert.Equal(t, tt.wantColor, attachment["color"])
			assert.Contains(t, attachment["title"], tt.wantTitle)
		})
	}
}

func TestEventType_Values(t *testing.T) {
	assert.Equal(t, EventType("task_pending_approval"), EventTaskPendingApproval)
	assert.Equal(t, EventType("task_approved"), EventTaskApproved)
	assert.Equal(t, EventType("task_rejected"), EventTaskRejected)
	assert.Equal(t, EventType("execution_started"), EventExecutionStarted)
	assert.Equal(t, EventType("execution_completed"), EventExecutionCompleted)
	assert.Equal(t, EventType("execution_failed"), EventExecutionFailed)
	assert.Equal(t, EventType("phase_started"), EventPhaseStarted)
	assert.Equal(t, EventType("phase_completed"), EventPhaseCompleted)
	assert.Equal(t, EventType("phase_failed"), EventPhaseFailed)
}

func TestNotify_NoChannelsEnabled(t *testing.T) {
	log := logger.New("debug", "json")
	cfg := config.NotificationConfig{
		SlackEnabled:   false,
		EmailEnabled:   false,
		WebhookEnabled: false,
	}

	n := New(cfg, log)
	ctx := context.Background()

	err := n.Notify(ctx, Event{
		Type:   EventTaskApproved,
		TaskID: "task-123",
	})

	// Should succeed with no errors since no channels are enabled
	assert.NoError(t, err)
}
