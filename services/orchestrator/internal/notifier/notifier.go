// Package notifier provides notification capabilities for task and execution events.
package notifier

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	"github.com/quantumlayerhq/ql-rf/pkg/config"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/executor"
)

// Notifier sends notifications for task events.
type Notifier struct {
	cfg    config.NotificationConfig
	log    *logger.Logger
	client *http.Client
}

// New creates a new Notifier.
func New(cfg config.NotificationConfig, log *logger.Logger) *Notifier {
	return &Notifier{
		cfg: cfg,
		log: log.WithComponent("notifier"),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// EventType represents the type of notification event.
type EventType string

const (
	EventTaskPendingApproval EventType = "task_pending_approval"
	EventTaskApproved        EventType = "task_approved"
	EventTaskRejected        EventType = "task_rejected"
	EventExecutionStarted    EventType = "execution_started"
	EventExecutionCompleted  EventType = "execution_completed"
	EventExecutionFailed     EventType = "execution_failed"
	EventPhaseStarted        EventType = "phase_started"
	EventPhaseCompleted      EventType = "phase_completed"
	EventPhaseFailed         EventType = "phase_failed"
)

// Event represents a notification event.
type Event struct {
	Type        EventType              `json:"type"`
	TaskID      string                 `json:"task_id"`
	TaskType    string                 `json:"task_type,omitempty"`
	Environment string                 `json:"environment,omitempty"`
	RiskLevel   string                 `json:"risk_level,omitempty"`
	Summary     string                 `json:"summary,omitempty"`
	UserID      string                 `json:"user_id,omitempty"`
	Execution   *executor.Execution    `json:"execution,omitempty"`
	Phase       *executor.PhaseExecution `json:"phase,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Notify sends a notification for an event.
func (n *Notifier) Notify(ctx context.Context, event Event) error {
	event.Timestamp = time.Now()

	var errs []string

	// Send to Slack
	if n.cfg.SlackEnabled {
		if err := n.sendSlack(ctx, event); err != nil {
			n.log.Error("failed to send Slack notification", "error", err)
			errs = append(errs, fmt.Sprintf("slack: %v", err))
		}
	}

	// Send email
	if n.cfg.EmailEnabled {
		if err := n.sendEmail(ctx, event); err != nil {
			n.log.Error("failed to send email notification", "error", err)
			errs = append(errs, fmt.Sprintf("email: %v", err))
		}
	}

	// Send webhook
	if n.cfg.WebhookEnabled {
		if err := n.sendWebhook(ctx, event); err != nil {
			n.log.Error("failed to send webhook notification", "error", err)
			errs = append(errs, fmt.Sprintf("webhook: %v", err))
		}
	}

	// Send to Microsoft Teams
	if n.cfg.TeamsEnabled {
		if err := n.sendTeams(ctx, event); err != nil {
			n.log.Error("failed to send Teams notification", "error", err)
			errs = append(errs, fmt.Sprintf("teams: %v", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("notification errors: %s", strings.Join(errs, "; "))
	}

	return nil
}

// NotifyTaskPendingApproval sends notification when a task is pending approval.
func (n *Notifier) NotifyTaskPendingApproval(ctx context.Context, taskID, taskType, environment, riskLevel, summary string) error {
	return n.Notify(ctx, Event{
		Type:        EventTaskPendingApproval,
		TaskID:      taskID,
		TaskType:    taskType,
		Environment: environment,
		RiskLevel:   riskLevel,
		Summary:     summary,
	})
}

// NotifyTaskApproved sends notification when a task is approved.
func (n *Notifier) NotifyTaskApproved(ctx context.Context, taskID, userID string) error {
	return n.Notify(ctx, Event{
		Type:   EventTaskApproved,
		TaskID: taskID,
		UserID: userID,
	})
}

// NotifyTaskRejected sends notification when a task is rejected.
func (n *Notifier) NotifyTaskRejected(ctx context.Context, taskID, userID, reason string) error {
	return n.Notify(ctx, Event{
		Type:    EventTaskRejected,
		TaskID:  taskID,
		UserID:  userID,
		Summary: reason,
	})
}

// NotifyExecutionStarted sends notification when execution starts.
func (n *Notifier) NotifyExecutionStarted(ctx context.Context, exec *executor.Execution) error {
	return n.Notify(ctx, Event{
		Type:      EventExecutionStarted,
		TaskID:    exec.TaskID,
		Execution: exec,
	})
}

// NotifyExecutionCompleted sends notification when execution completes.
func (n *Notifier) NotifyExecutionCompleted(ctx context.Context, exec *executor.Execution) error {
	eventType := EventExecutionCompleted
	if exec.Status == executor.StatusFailed || exec.Status == executor.StatusRolledBack {
		eventType = EventExecutionFailed
	}
	return n.Notify(ctx, Event{
		Type:      eventType,
		TaskID:    exec.TaskID,
		Execution: exec,
	})
}

// NotifyPhaseStarted sends notification when a phase starts.
func (n *Notifier) NotifyPhaseStarted(ctx context.Context, exec *executor.Execution, phase *executor.PhaseExecution) error {
	return n.Notify(ctx, Event{
		Type:      EventPhaseStarted,
		TaskID:    exec.TaskID,
		Execution: exec,
		Phase:     phase,
	})
}

// NotifyPhaseCompleted sends notification when a phase completes.
func (n *Notifier) NotifyPhaseCompleted(ctx context.Context, exec *executor.Execution, phase *executor.PhaseExecution) error {
	eventType := EventPhaseCompleted
	if phase.Status == executor.PhaseStatusFailed {
		eventType = EventPhaseFailed
	}
	return n.Notify(ctx, Event{
		Type:      eventType,
		TaskID:    exec.TaskID,
		Execution: exec,
		Phase:     phase,
	})
}

// sendSlack sends a notification to Slack.
func (n *Notifier) sendSlack(ctx context.Context, event Event) error {
	if n.cfg.SlackWebhookURL == "" {
		return fmt.Errorf("slack webhook URL not configured")
	}

	// Build Slack message
	message := n.buildSlackMessage(event)

	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", n.cfg.SlackWebhookURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack returned status %d", resp.StatusCode)
	}

	n.log.Debug("sent Slack notification", "event", event.Type, "task_id", event.TaskID)
	return nil
}

// buildSlackMessage builds a Slack message for an event.
func (n *Notifier) buildSlackMessage(event Event) map[string]interface{} {
	var color, title, text string
	var emoji string

	switch event.Type {
	case EventTaskPendingApproval:
		color = "#FFA500" // Orange
		emoji = ":warning:"
		title = "Task Awaiting Approval"
		text = fmt.Sprintf("*%s* task requires approval\n*Environment:* %s\n*Risk Level:* %s\n*Summary:* %s",
			event.TaskType, event.Environment, event.RiskLevel, event.Summary)

	case EventTaskApproved:
		color = "#36A64F" // Green
		emoji = ":white_check_mark:"
		title = "Task Approved"
		text = fmt.Sprintf("Task `%s` was approved by %s", event.TaskID[:8], event.UserID)

	case EventTaskRejected:
		color = "#FF0000" // Red
		emoji = ":x:"
		title = "Task Rejected"
		text = fmt.Sprintf("Task `%s` was rejected by %s\n*Reason:* %s", event.TaskID[:8], event.UserID, event.Summary)

	case EventExecutionStarted:
		color = "#36A64F" // Green
		emoji = ":rocket:"
		title = "Execution Started"
		text = fmt.Sprintf("Execution started for task `%s`\n*Phases:* %d", event.TaskID[:8], event.Execution.TotalPhases)

	case EventExecutionCompleted:
		color = "#36A64F" // Green
		emoji = ":tada:"
		title = "Execution Completed"
		text = fmt.Sprintf("Task `%s` execution completed successfully", event.TaskID[:8])

	case EventExecutionFailed:
		color = "#FF0000" // Red
		emoji = ":rotating_light:"
		title = "Execution Failed"
		errMsg := ""
		if event.Execution != nil {
			errMsg = event.Execution.Error
		}
		text = fmt.Sprintf("Task `%s` execution failed\n*Error:* %s", event.TaskID[:8], errMsg)

	case EventPhaseStarted:
		color = "#439FE0" // Blue
		emoji = ":arrow_forward:"
		phaseName := ""
		if event.Phase != nil {
			phaseName = event.Phase.Name
		}
		title = "Phase Started"
		text = fmt.Sprintf("Phase *%s* started for task `%s`", phaseName, event.TaskID[:8])

	case EventPhaseCompleted:
		color = "#36A64F" // Green
		emoji = ":ballot_box_with_check:"
		phaseName := ""
		if event.Phase != nil {
			phaseName = event.Phase.Name
		}
		title = "Phase Completed"
		text = fmt.Sprintf("Phase *%s* completed for task `%s`", phaseName, event.TaskID[:8])

	case EventPhaseFailed:
		color = "#FF0000" // Red
		emoji = ":warning:"
		phaseName := ""
		phaseError := ""
		if event.Phase != nil {
			phaseName = event.Phase.Name
			phaseError = event.Phase.Error
		}
		title = "Phase Failed"
		text = fmt.Sprintf("Phase *%s* failed for task `%s`\n*Error:* %s", phaseName, event.TaskID[:8], phaseError)

	default:
		color = "#808080" // Gray
		emoji = ":bell:"
		title = "Notification"
		text = fmt.Sprintf("Event: %s for task `%s`", event.Type, event.TaskID[:8])
	}

	return map[string]interface{}{
		"channel":  n.cfg.SlackChannel,
		"username": "QL-RF AI Orchestrator",
		"icon_emoji": ":robot_face:",
		"attachments": []map[string]interface{}{
			{
				"color":      color,
				"title":      emoji + " " + title,
				"text":       text,
				"footer":     "QL-RF Orchestrator",
				"footer_icon": "https://platform.slack-edge.com/img/default_application_icon.png",
				"ts":         event.Timestamp.Unix(),
				"mrkdwn_in":  []string{"text"},
			},
		},
	}
}

// sendEmail sends an email notification.
func (n *Notifier) sendEmail(ctx context.Context, event Event) error {
	if n.cfg.SMTPHost == "" || len(n.cfg.EmailRecipients) == 0 {
		return fmt.Errorf("email not configured")
	}

	subject, body := n.buildEmailContent(event)

	// Build message
	msg := fmt.Sprintf("From: %s\r\n", n.cfg.EmailFrom)
	msg += fmt.Sprintf("To: %s\r\n", strings.Join(n.cfg.EmailRecipients, ","))
	msg += fmt.Sprintf("Subject: %s\r\n", subject)
	msg += "MIME-Version: 1.0\r\n"
	msg += "Content-Type: text/html; charset=\"UTF-8\"\r\n"
	msg += "\r\n"
	msg += body

	// Send email
	auth := smtp.PlainAuth("", n.cfg.SMTPUser, n.cfg.SMTPPassword, n.cfg.SMTPHost)
	addr := fmt.Sprintf("%s:%d", n.cfg.SMTPHost, n.cfg.SMTPPort)

	err := smtp.SendMail(addr, auth, n.cfg.EmailFrom, n.cfg.EmailRecipients, []byte(msg))
	if err != nil {
		return err
	}

	n.log.Debug("sent email notification", "event", event.Type, "task_id", event.TaskID)
	return nil
}

// buildEmailContent builds email subject and body for an event.
func (n *Notifier) buildEmailContent(event Event) (subject, body string) {
	// Use configured base URL or default to localhost for dev
	baseURL := n.cfg.AppBaseURL
	if baseURL == "" {
		baseURL = "http://localhost:3000"
	}

	switch event.Type {
	case EventTaskPendingApproval:
		subject = fmt.Sprintf("[QL-RF] Task Awaiting Approval: %s", event.TaskType)
		body = fmt.Sprintf(`
<html>
<body>
<h2>Task Awaiting Approval</h2>
<p><strong>Task ID:</strong> %s</p>
<p><strong>Type:</strong> %s</p>
<p><strong>Environment:</strong> %s</p>
<p><strong>Risk Level:</strong> %s</p>
<p><strong>Summary:</strong> %s</p>
<p><a href="%s/ai/tasks/%s">View Task</a></p>
</body>
</html>
`, event.TaskID, event.TaskType, event.Environment, event.RiskLevel, event.Summary, baseURL, event.TaskID)

	case EventTaskApproved:
		subject = fmt.Sprintf("[QL-RF] Task Approved: %s", event.TaskID[:8])
		body = fmt.Sprintf(`
<html>
<body>
<h2>Task Approved</h2>
<p><strong>Task ID:</strong> %s</p>
<p><strong>Approved by:</strong> %s</p>
<p>Execution will begin shortly.</p>
</body>
</html>
`, event.TaskID, event.UserID)

	case EventTaskRejected:
		subject = fmt.Sprintf("[QL-RF] Task Rejected: %s", event.TaskID[:8])
		body = fmt.Sprintf(`
<html>
<body>
<h2>Task Rejected</h2>
<p><strong>Task ID:</strong> %s</p>
<p><strong>Rejected by:</strong> %s</p>
<p><strong>Reason:</strong> %s</p>
</body>
</html>
`, event.TaskID, event.UserID, event.Summary)

	case EventExecutionCompleted:
		subject = fmt.Sprintf("[QL-RF] Execution Completed: %s", event.TaskID[:8])
		body = fmt.Sprintf(`
<html>
<body>
<h2>Execution Completed Successfully</h2>
<p><strong>Task ID:</strong> %s</p>
<p><strong>Status:</strong> %s</p>
</body>
</html>
`, event.TaskID, event.Execution.Status)

	case EventExecutionFailed:
		errMsg := ""
		if event.Execution != nil {
			errMsg = event.Execution.Error
		}
		subject = fmt.Sprintf("[QL-RF] Execution Failed: %s", event.TaskID[:8])
		body = fmt.Sprintf(`
<html>
<body>
<h2 style="color: red;">Execution Failed</h2>
<p><strong>Task ID:</strong> %s</p>
<p><strong>Error:</strong> %s</p>
<p><a href="%s/ai/tasks/%s">View Details</a></p>
</body>
</html>
`, event.TaskID, errMsg, baseURL, event.TaskID)

	default:
		subject = fmt.Sprintf("[QL-RF] Notification: %s", event.Type)
		body = fmt.Sprintf(`
<html>
<body>
<h2>Notification</h2>
<p><strong>Event:</strong> %s</p>
<p><strong>Task ID:</strong> %s</p>
<p><strong>Time:</strong> %s</p>
</body>
</html>
`, event.Type, event.TaskID, event.Timestamp.Format(time.RFC3339))
	}

	return subject, body
}

// sendWebhook sends a webhook notification.
func (n *Notifier) sendWebhook(ctx context.Context, event Event) error {
	if n.cfg.WebhookURL == "" {
		return fmt.Errorf("webhook URL not configured")
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", n.cfg.WebhookURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-QL-Event", string(event.Type))

	if n.cfg.WebhookSecret != "" {
		// Compute HMAC-SHA256 signature
		signature := n.computeHMAC(payload)
		req.Header.Set("X-QL-Signature", signature)
	}

	resp, err := n.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	n.log.Debug("sent webhook notification", "event", event.Type, "task_id", event.TaskID)
	return nil
}

// computeHMAC computes an HMAC-SHA256 signature for webhook payloads.
func (n *Notifier) computeHMAC(payload []byte) string {
	h := hmac.New(sha256.New, []byte(n.cfg.WebhookSecret))
	h.Write(payload)
	return "sha256=" + hex.EncodeToString(h.Sum(nil))
}

// sendTeams sends a notification to Microsoft Teams via webhook.
func (n *Notifier) sendTeams(ctx context.Context, event Event) error {
	if n.cfg.TeamsWebhookURL == "" {
		return fmt.Errorf("Teams webhook URL not configured")
	}

	// Build Teams Adaptive Card message
	message := n.buildTeamsMessage(event)

	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", n.cfg.TeamsWebhookURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Teams webhooks return 200 on success
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Teams webhook returned status %d", resp.StatusCode)
	}

	n.log.Debug("sent Teams notification", "event", event.Type, "task_id", event.TaskID)
	return nil
}

// buildTeamsMessage builds a Microsoft Teams Adaptive Card message for an event.
func (n *Notifier) buildTeamsMessage(event Event) map[string]interface{} {
	var title, text string
	var iconURL string

	// Base URL for links
	baseURL := n.cfg.AppBaseURL
	if baseURL == "" {
		baseURL = "http://localhost:3000"
	}

	switch event.Type {
	case EventTaskPendingApproval:
		iconURL = "https://adaptivecards.io/content/pending.png"
		title = "⚠️ Task Awaiting Approval"
		text = fmt.Sprintf("**%s** task requires approval\n\n**Environment:** %s\n\n**Risk Level:** %s\n\n**Summary:** %s",
			event.TaskType, event.Environment, event.RiskLevel, event.Summary)

	case EventTaskApproved:
		iconURL = "https://adaptivecards.io/content/check.png"
		title = "✅ Task Approved"
		text = fmt.Sprintf("Task `%s` was approved by **%s**\n\nExecution will begin shortly.", event.TaskID[:8], event.UserID)

	case EventTaskRejected:
		iconURL = "https://adaptivecards.io/content/error.png"
		title = "❌ Task Rejected"
		text = fmt.Sprintf("Task `%s` was rejected by **%s**\n\n**Reason:** %s", event.TaskID[:8], event.UserID, event.Summary)

	case EventExecutionStarted:
		iconURL = "https://adaptivecards.io/content/rocket.png"
		title = "🚀 Execution Started"
		text = fmt.Sprintf("Execution started for task `%s`\n\n**Total Phases:** %d", event.TaskID[:8], event.Execution.TotalPhases)

	case EventExecutionCompleted:
		iconURL = "https://adaptivecards.io/content/check.png"
		title = "🎉 Execution Completed"
		text = fmt.Sprintf("Task `%s` execution completed successfully!", event.TaskID[:8])

	case EventExecutionFailed:
		iconURL = "https://adaptivecards.io/content/error.png"
		title = "🚨 Execution Failed"
		errMsg := ""
		if event.Execution != nil {
			errMsg = event.Execution.Error
		}
		text = fmt.Sprintf("Task `%s` execution failed!\n\n**Error:** %s", event.TaskID[:8], errMsg)

	case EventPhaseStarted:
		iconURL = "https://adaptivecards.io/content/pending.png"
		phaseName := ""
		if event.Phase != nil {
			phaseName = event.Phase.Name
		}
		title = "▶️ Phase Started"
		text = fmt.Sprintf("Phase **%s** started for task `%s`", phaseName, event.TaskID[:8])

	case EventPhaseCompleted:
		iconURL = "https://adaptivecards.io/content/check.png"
		phaseName := ""
		if event.Phase != nil {
			phaseName = event.Phase.Name
		}
		title = "☑️ Phase Completed"
		text = fmt.Sprintf("Phase **%s** completed for task `%s`", phaseName, event.TaskID[:8])

	case EventPhaseFailed:
		iconURL = "https://adaptivecards.io/content/error.png"
		phaseName := ""
		phaseError := ""
		if event.Phase != nil {
			phaseName = event.Phase.Name
			phaseError = event.Phase.Error
		}
		title = "⚠️ Phase Failed"
		text = fmt.Sprintf("Phase **%s** failed for task `%s`\n\n**Error:** %s", phaseName, event.TaskID[:8], phaseError)

	default:
		iconURL = "https://adaptivecards.io/content/notification.png"
		title = "🔔 Notification"
		text = fmt.Sprintf("Event: %s for task `%s`", event.Type, event.TaskID[:8])
	}

	// Build task URL
	taskURL := fmt.Sprintf("%s/ai/tasks/%s", baseURL, event.TaskID)

	// Build Microsoft Teams Adaptive Card payload
	// Using the newer Adaptive Card format for better rendering
	return map[string]interface{}{
		"type": "message",
		"attachments": []map[string]interface{}{
			{
				"contentType": "application/vnd.microsoft.card.adaptive",
				"contentUrl":  nil,
				"content": map[string]interface{}{
					"$schema": "http://adaptivecards.io/schemas/adaptive-card.json",
					"type":    "AdaptiveCard",
					"version": "1.4",
					"msteams": map[string]interface{}{
						"width": "Full",
					},
					"body": []map[string]interface{}{
						{
							"type":   "Container",
							"style":  "emphasis",
							"bleed":  true,
							"items": []map[string]interface{}{
								{
									"type":   "ColumnSet",
									"columns": []map[string]interface{}{
										{
											"type":  "Column",
											"width": "auto",
											"items": []map[string]interface{}{
												{
													"type": "Image",
													"url":  iconURL,
													"size": "Small",
												},
											},
										},
										{
											"type":  "Column",
											"width": "stretch",
											"items": []map[string]interface{}{
												{
													"type":   "TextBlock",
													"text":   title,
													"weight": "Bolder",
													"size":   "Medium",
													"wrap":   true,
												},
												{
													"type":     "TextBlock",
													"text":     fmt.Sprintf("Task ID: %s", event.TaskID[:8]),
													"isSubtle": true,
													"spacing":  "None",
												},
											},
										},
									},
								},
							},
						},
						{
							"type":    "TextBlock",
							"text":    text,
							"wrap":    true,
							"spacing": "Medium",
						},
						{
							"type":     "TextBlock",
							"text":     fmt.Sprintf("_Sent at %s_", event.Timestamp.Format("2006-01-02 15:04:05 MST")),
							"isSubtle": true,
							"wrap":     true,
							"size":     "Small",
							"spacing":  "Medium",
						},
					},
					"actions": []map[string]interface{}{
						{
							"type":  "Action.OpenUrl",
							"title": "View Task Details",
							"url":   taskURL,
						},
					},
				},
			},
		},
	}
}

// TeamsMessageCard builds a legacy MessageCard format for older Teams webhook connectors.
// This is kept as a fallback option.
func (n *Notifier) buildTeamsMessageCard(event Event) map[string]interface{} {
	var themeColor, title, text string

	switch event.Type {
	case EventTaskPendingApproval:
		themeColor = "FFA500"
		title = "Task Awaiting Approval"
		text = fmt.Sprintf("**%s** task requires approval<br>**Environment:** %s<br>**Risk Level:** %s<br>**Summary:** %s",
			event.TaskType, event.Environment, event.RiskLevel, event.Summary)
	case EventTaskApproved:
		themeColor = "36A64F"
		title = "Task Approved"
		text = fmt.Sprintf("Task %s was approved by %s", event.TaskID[:8], event.UserID)
	case EventTaskRejected:
		themeColor = "FF0000"
		title = "Task Rejected"
		text = fmt.Sprintf("Task %s was rejected by %s<br>**Reason:** %s", event.TaskID[:8], event.UserID, event.Summary)
	case EventExecutionStarted:
		themeColor = "36A64F"
		title = "Execution Started"
		text = fmt.Sprintf("Execution started for task %s", event.TaskID[:8])
	case EventExecutionCompleted:
		themeColor = "36A64F"
		title = "Execution Completed"
		text = fmt.Sprintf("Task %s execution completed successfully", event.TaskID[:8])
	case EventExecutionFailed:
		themeColor = "FF0000"
		title = "Execution Failed"
		errMsg := ""
		if event.Execution != nil {
			errMsg = event.Execution.Error
		}
		text = fmt.Sprintf("Task %s execution failed<br>**Error:** %s", event.TaskID[:8], errMsg)
	default:
		themeColor = "808080"
		title = "Notification"
		text = fmt.Sprintf("Event: %s for task %s", event.Type, event.TaskID[:8])
	}

	baseURL := n.cfg.AppBaseURL
	if baseURL == "" {
		baseURL = "http://localhost:3000"
	}

	// Legacy Office 365 Connector Card format
	return map[string]interface{}{
		"@type":      "MessageCard",
		"@context":   "http://schema.org/extensions",
		"themeColor": themeColor,
		"summary":    title,
		"sections": []map[string]interface{}{
			{
				"activityTitle":    title,
				"activitySubtitle": fmt.Sprintf("Task ID: %s", event.TaskID[:8]),
				"activityImage":    "https://adaptivecards.io/content/notification.png",
				"text":             text,
				"markdown":         true,
			},
		},
		"potentialAction": []map[string]interface{}{
			{
				"@type": "OpenUri",
				"name":  "View Task",
				"targets": []map[string]interface{}{
					{
						"os":  "default",
						"uri": fmt.Sprintf("%s/ai/tasks/%s", baseURL, event.TaskID),
					},
				},
			},
		},
	}
}
