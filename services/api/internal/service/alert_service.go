package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// AlertService handles alert business logic.
type AlertService struct {
	repo AlertRepository
}

// NewAlertService creates a new AlertService.
func NewAlertService(repo AlertRepository) *AlertService {
	return &AlertService{repo: repo}
}

// GetAlertInput contains input for getting an alert.
type GetAlertInput struct {
	ID    uuid.UUID
	OrgID uuid.UUID
}

// GetAlert retrieves an alert by ID with authorization check.
func (s *AlertService) GetAlert(ctx context.Context, input GetAlertInput) (*Alert, error) {
	alert, err := s.repo.GetAlert(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("get alert: %w", err)
	}

	// Authorization: verify org ownership
	if alert.OrgID != input.OrgID {
		return nil, ErrNotFound
	}

	return alert, nil
}

// ListAlertsInput contains input for listing alerts.
type ListAlertsInput struct {
	OrgID    uuid.UUID
	Severity *string
	Status   *string
	Source   *string
	SiteID   *uuid.UUID
	Page     int
	PageSize int
}

// ListAlertsOutput contains output for listing alerts.
type ListAlertsOutput struct {
	Alerts     []Alert `json:"alerts"`
	Total      int64   `json:"total"`
	Page       int     `json:"page"`
	PageSize   int     `json:"pageSize"`
	TotalPages int     `json:"totalPages"`
}

// ListAlerts retrieves a paginated list of alerts.
func (s *AlertService) ListAlerts(ctx context.Context, input ListAlertsInput) (*ListAlertsOutput, error) {
	// Apply defaults
	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 || input.PageSize > 100 {
		input.PageSize = 20
	}

	offset := int32((input.Page - 1) * input.PageSize)

	alerts, err := s.repo.ListAlerts(ctx, ListAlertsParams{
		OrgID:    input.OrgID,
		Severity: input.Severity,
		Status:   input.Status,
		Source:   input.Source,
		SiteID:   input.SiteID,
		Limit:    int32(input.PageSize),
		Offset:   offset,
	})
	if err != nil {
		return nil, fmt.Errorf("list alerts: %w", err)
	}

	total, err := s.repo.CountAlertsByOrg(ctx, input.OrgID)
	if err != nil {
		return nil, fmt.Errorf("count alerts: %w", err)
	}

	totalPages := int(total) / input.PageSize
	if int(total)%input.PageSize > 0 {
		totalPages++
	}

	return &ListAlertsOutput{
		Alerts:     alerts,
		Total:      total,
		Page:       input.Page,
		PageSize:   input.PageSize,
		TotalPages: totalPages,
	}, nil
}

// AcknowledgeAlertInput contains input for acknowledging an alert.
type AcknowledgeAlertInput struct {
	ID     uuid.UUID
	OrgID  uuid.UUID
	UserID uuid.UUID
}

// AcknowledgeAlert marks an alert as acknowledged.
func (s *AlertService) AcknowledgeAlert(ctx context.Context, input AcknowledgeAlertInput) error {
	// Verify ownership
	alert, err := s.repo.GetAlert(ctx, input.ID)
	if err != nil {
		return fmt.Errorf("get alert: %w", err)
	}
	if alert.OrgID != input.OrgID {
		return ErrNotFound
	}

	if err := s.repo.UpdateAlertStatus(ctx, input.ID, "acknowledged", &input.UserID); err != nil {
		return fmt.Errorf("acknowledge alert: %w", err)
	}

	return nil
}

// ResolveAlertInput contains input for resolving an alert.
type ResolveAlertInput struct {
	ID     uuid.UUID
	OrgID  uuid.UUID
	UserID uuid.UUID
}

// ResolveAlert marks an alert as resolved.
func (s *AlertService) ResolveAlert(ctx context.Context, input ResolveAlertInput) error {
	// Verify ownership
	alert, err := s.repo.GetAlert(ctx, input.ID)
	if err != nil {
		return fmt.Errorf("get alert: %w", err)
	}
	if alert.OrgID != input.OrgID {
		return ErrNotFound
	}

	if err := s.repo.UpdateAlertStatus(ctx, input.ID, "resolved", &input.UserID); err != nil {
		return fmt.Errorf("resolve alert: %w", err)
	}

	return nil
}

// GetAlertSummaryInput contains input for getting alert summary.
type GetAlertSummaryInput struct {
	OrgID uuid.UUID
}

// AlertSummary contains summary statistics for alerts.
type AlertSummary struct {
	Total        int64        `json:"total"`
	BySeverity   []AlertCount `json:"bySeverity"`
	OpenCount    int64        `json:"openCount"`
	AckedCount   int64        `json:"ackedCount"`
	ResolvedCount int64       `json:"resolvedCount"`
}

// GetAlertSummary retrieves alert summary statistics.
func (s *AlertService) GetAlertSummary(ctx context.Context, input GetAlertSummaryInput) (*AlertSummary, error) {
	total, err := s.repo.CountAlertsByOrg(ctx, input.OrgID)
	if err != nil {
		return nil, fmt.Errorf("count alerts: %w", err)
	}

	bySeverity, err := s.repo.CountAlertsBySeverity(ctx, input.OrgID)
	if err != nil {
		return nil, fmt.Errorf("count alerts by severity: %w", err)
	}

	return &AlertSummary{
		Total:      total,
		BySeverity: bySeverity,
	}, nil
}
