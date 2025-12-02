// Package main is the entry point for the drift service.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"

	"github.com/quantumlayerhq/ql-rf/pkg/config"
	"github.com/quantumlayerhq/ql-rf/pkg/database"
	"github.com/quantumlayerhq/ql-rf/pkg/kafka"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/pkg/models"
	"github.com/quantumlayerhq/ql-rf/services/drift/internal/engine"
)

// Build information (set via ldflags).
var (
	version   = "dev"
	buildTime = "unknown"
	gitCommit = "unknown"
)

func main() {
	if err := run(); err != nil {
		slog.Error("application error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize logger
	log := logger.New(cfg.LogLevel, "json")
	log = log.WithService("drift")

	log.Info("starting drift service",
		"version", version,
		"build_time", buildTime,
		"git_commit", gitCommit,
		"env", cfg.Env,
	)

	// Create context that listens for shutdown signals
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect to database
	db, err := database.New(ctx, cfg.Database)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()
	log.Info("connected to database")

	// Create Kafka producer
	producer, err := kafka.NewProducer(cfg.Kafka)
	if err != nil {
		return fmt.Errorf("failed to create Kafka producer: %w", err)
	}
	defer producer.Close()
	log.Info("connected to Kafka producer")

	// Create Kafka consumer
	consumer, err := kafka.NewConsumer(cfg.Kafka)
	if err != nil {
		return fmt.Errorf("failed to create Kafka consumer: %w", err)
	}
	defer consumer.Close()
	log.Info("connected to Kafka consumer")

	// Create drift engine
	driftConfig := models.DriftConfig{
		WarningThreshold:  cfg.Drift.ThresholdWarning,
		CriticalThreshold: cfg.Drift.ThresholdCritical,
		MaxOffenders:      10,
	}
	driftEngine := engine.New(db, log, driftConfig)

	// Start Kafka consumer for asset events
	go func() {
		topics := []string{cfg.Kafka.Topics.AssetDiscovered}
		log.Info("starting Kafka consumer", "topics", topics)

		err := consumer.Subscribe(ctx, topics, func(ctx context.Context, msg kafka.Message) error {
			log.Debug("received message",
				"topic", msg.Topic,
				"key", msg.Key,
			)

			// Process asset discovered event
			// This could trigger incremental drift calculation
			return nil
		})
		if err != nil && ctx.Err() == nil {
			log.Error("Kafka consumer error", "error", err)
		}
	}()

	// Create and start scheduler for periodic drift calculation
	scheduler := cron.New(cron.WithSeconds())

	interval := cfg.Drift.CalculationInterval
	cronSpec := fmt.Sprintf("@every %s", interval)

	_, err = scheduler.AddFunc(cronSpec, func() {
		runDriftCalculation(ctx, driftEngine, db, producer, cfg, log)
	})
	if err != nil {
		return fmt.Errorf("failed to schedule drift calculation: %w", err)
	}

	scheduler.Start()
	log.Info("scheduler started", "interval", interval)

	// Run initial calculation
	go runDriftCalculation(ctx, driftEngine, db, producer, cfg, log)

	// Wait for shutdown signal
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	<-shutdown
	log.Info("shutdown signal received")

	// Stop scheduler
	scheduler.Stop()

	log.Info("drift service shutdown complete")
	return nil
}

func runDriftCalculation(
	ctx context.Context,
	driftEngine *engine.Engine,
	db *database.DB,
	producer *kafka.Producer,
	cfg *config.Config,
	log *logger.Logger,
) {
	log.Info("starting drift calculation")
	startTime := time.Now()

	// Get all organizations (in production, query from database)
	// For now, use a placeholder org ID
	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	// Calculate drift summary
	summary, err := driftEngine.CalculateSummary(ctx, orgID)
	if err != nil {
		log.Error("failed to calculate drift summary", "error", err)
		return
	}

	log.Info("drift calculation completed",
		"org_id", orgID,
		"total_assets", summary.TotalAssets,
		"compliant_assets", summary.CompliantAssets,
		"coverage_pct", fmt.Sprintf("%.2f", summary.CoveragePct),
		"status", summary.Status,
	)

	// Store drift report in database
	// TODO: Insert into drift_reports table

	// Check if drift exceeds thresholds and publish event
	if summary.Status == models.DriftStatusWarning || summary.Status == models.DriftStatusCritical {
		event := kafka.Event{
			ID:        uuid.New().String(),
			Type:      "drift.detected",
			Source:    "drift-service",
			Timestamp: time.Now(),
			Data: models.DriftDetectedEvent{
				Report: models.DriftReport{
					ID:              uuid.New(),
					OrgID:           orgID,
					TotalAssets:     summary.TotalAssets,
					CompliantAssets: summary.CompliantAssets,
					CoveragePct:     summary.CoveragePct,
					Status:          summary.Status,
					CalculatedAt:    summary.CalculatedAt,
				},
				TopOffenders: summary.TopOffenders,
				Timestamp:    time.Now(),
			},
		}

		if err := producer.PublishEvent(ctx, cfg.Kafka.Topics.DriftDetected, event); err != nil {
			log.Error("failed to publish drift event", "error", err)
		} else {
			log.Info("drift event published",
				"status", summary.Status,
				"top_offenders", len(summary.TopOffenders),
			)
		}
	}

	duration := time.Since(startTime)
	log.Info("drift calculation cycle completed", "duration", duration.String())
}
