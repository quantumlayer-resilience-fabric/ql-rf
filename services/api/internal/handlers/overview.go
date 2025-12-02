package handlers

import (
	"net/http"

	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/pkg/models"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/middleware"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/service"
)

// OverviewHandler handles overview/dashboard requests.
type OverviewHandler struct {
	svc *service.OverviewService
	log *logger.Logger
}

// NewOverviewHandler creates a new OverviewHandler.
func NewOverviewHandler(svc *service.OverviewService, log *logger.Logger) *OverviewHandler {
	return &OverviewHandler{
		svc: svc,
		log: log.WithComponent("overview-handler"),
	}
}

// GetMetrics returns dashboard overview metrics.
func (h *OverviewHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	org := middleware.GetOrg(ctx)
	if org == nil {
		http.Error(w, "organization not found", http.StatusUnauthorized)
		return
	}

	// Call service
	metrics, err := h.svc.GetOverviewMetrics(ctx, service.GetOverviewMetricsInput{
		OrgID: org.ID,
	})
	if err != nil {
		h.log.Error("failed to get overview metrics", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Convert to response format
	response := serviceOverviewToModel(metrics)

	writeJSON(w, http.StatusOK, response)
}

// Helper functions to convert between service and model types
func serviceOverviewToModel(m *service.OverviewMetrics) models.OverviewMetrics {
	alerts := make([]models.AlertCount, 0, len(m.Alerts))
	for _, a := range m.Alerts {
		alerts = append(alerts, models.AlertCount{
			Severity: a.Severity,
			Count:    a.Count,
		})
	}

	activities := make([]models.Activity, 0, len(m.RecentActivity))
	for _, a := range m.RecentActivity {
		activities = append(activities, models.Activity{
			ID:        a.ID,
			OrgID:     a.OrgID,
			Type:      a.Type,
			Action:    a.Action,
			Detail:    a.Detail,
			UserID:    a.UserID,
			SiteID:    a.SiteID,
			AssetID:   a.AssetID,
			ImageID:   a.ImageID,
			Timestamp: a.Timestamp,
		})
	}

	platformDist := make([]models.PlatformCount, 0, len(m.PlatformDistribution))
	for _, p := range m.PlatformDistribution {
		platformDist = append(platformDist, models.PlatformCount{
			Platform:   models.Platform(p.Platform),
			Count:      p.Count,
			Percentage: p.Percentage,
		})
	}

	return models.OverviewMetrics{
		FleetSize: models.MetricWithTrend{
			Value: m.FleetSize.Value,
			Trend: models.MetricTrend{
				Direction: m.FleetSize.Trend.Direction,
				Value:     m.FleetSize.Trend.Value,
				Period:    m.FleetSize.Trend.Period,
			},
		},
		DriftScore: models.FloatMetricWithTrend{
			Value: m.DriftScore.Value,
			Trend: models.MetricTrend{
				Direction: m.DriftScore.Trend.Direction,
				Value:     m.DriftScore.Trend.Value,
				Period:    m.DriftScore.Trend.Period,
			},
		},
		Compliance: models.FloatMetricWithTrend{
			Value: m.Compliance.Value,
			Trend: models.MetricTrend{
				Direction: m.Compliance.Trend.Direction,
				Value:     m.Compliance.Trend.Value,
				Period:    m.Compliance.Trend.Period,
			},
		},
		DRReadiness: models.FloatMetricWithTrend{
			Value: m.DRReadiness.Value,
			Trend: models.MetricTrend{
				Direction: m.DRReadiness.Trend.Direction,
				Value:     m.DRReadiness.Trend.Value,
				Period:    m.DRReadiness.Trend.Period,
			},
		},
		PlatformDistribution: platformDist,
		Alerts:               alerts,
		RecentActivity:       activities,
	}
}
