// Package main is the entry point for the connectors service.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/quantumlayerhq/ql-rf/pkg/config"
	"github.com/quantumlayerhq/ql-rf/pkg/database"
	"github.com/quantumlayerhq/ql-rf/pkg/kafka"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/services/connectors/internal/aws"
	"github.com/quantumlayerhq/ql-rf/services/connectors/internal/connector"
	"github.com/quantumlayerhq/ql-rf/services/connectors/internal/repository"
	"github.com/quantumlayerhq/ql-rf/services/connectors/internal/sync"
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
	log = log.WithService("connectors")

	log.Info("starting connectors service",
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
	log.Info("connected to Kafka")

	// Initialize repository and sync service
	repo := repository.New(db.Pool)
	syncSvc := sync.New(repo, producer, cfg.Kafka.Topics.AssetDiscovered, log)

	// Initialize connectors
	connectors := initializeConnectors(cfg, log)
	log.Info("initialized connectors", "count", len(connectors))

	// Create and start scheduler
	scheduler := cron.New(cron.WithSeconds())

	// Schedule asset discovery based on config interval
	interval := cfg.Connectors.SyncInterval
	cronSpec := fmt.Sprintf("@every %s", interval)

	_, err = scheduler.AddFunc(cronSpec, func() {
		runDiscovery(ctx, connectors, syncSvc, cfg, log)
	})
	if err != nil {
		return fmt.Errorf("failed to schedule discovery: %w", err)
	}

	scheduler.Start()
	log.Info("scheduler started", "interval", interval)

	// Run initial discovery
	go runDiscovery(ctx, connectors, syncSvc, cfg, log)

	// Wait for shutdown signal
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	<-shutdown
	log.Info("shutdown signal received")

	// Stop scheduler
	scheduler.Stop()

	// Close all connectors
	for _, c := range connectors {
		if err := c.Close(); err != nil {
			log.Error("failed to close connector",
				"connector", c.Name(),
				"error", err,
			)
		}
	}

	log.Info("connectors service shutdown complete")
	return nil
}

func initializeConnectors(cfg *config.Config, log *logger.Logger) []connector.Connector {
	var connectors []connector.Connector

	for _, name := range cfg.Connectors.Enabled {
		switch name {
		case "aws":
			awsConnector := aws.New(aws.Config{
				Region:        cfg.Connectors.AWS.Region,
				AssumeRoleARN: cfg.Connectors.AWS.AssumeRoleARN,
				Regions:       cfg.Connectors.AWS.Regions,
			}, log)
			connectors = append(connectors, awsConnector)

		case "azure":
			// TODO: Initialize Azure connector
			log.Info("Azure connector not yet implemented")

		case "gcp":
			// TODO: Initialize GCP connector
			log.Info("GCP connector not yet implemented")

		case "vsphere":
			// TODO: Initialize vSphere connector
			log.Info("vSphere connector not yet implemented")

		default:
			log.Warn("unknown connector", "name", name)
		}
	}

	return connectors
}

func runDiscovery(
	ctx context.Context,
	connectors []connector.Connector,
	syncSvc *sync.Service,
	cfg *config.Config,
	log *logger.Logger,
) {
	log.Info("starting asset discovery")
	startTime := time.Now()

	var totalResults []*sync.SyncResult

	for _, c := range connectors {
		// Connect to platform
		if err := c.Connect(ctx); err != nil {
			log.Error("failed to connect",
				"connector", c.Name(),
				"error", err,
			)
			continue
		}

		// Discover assets
		assets, err := c.DiscoverAssets(ctx, cfg.Connectors.OrgID)
		if err != nil {
			log.Error("failed to discover assets",
				"connector", c.Name(),
				"error", err,
			)
			continue
		}

		log.Info("discovered assets",
			"connector", c.Name(),
			"count", len(assets),
		)

		// Sync assets to database
		result, err := syncSvc.SyncAssets(ctx, cfg.Connectors.OrgID, string(c.Platform()), assets)
		if err != nil {
			log.Error("failed to sync assets",
				"connector", c.Name(),
				"error", err,
			)
			continue
		}

		totalResults = append(totalResults, result)
	}

	// Log summary
	var totalNew, totalUpdated, totalRemoved int
	for _, r := range totalResults {
		totalNew += r.AssetsNew
		totalUpdated += r.AssetsUpdated
		totalRemoved += r.AssetsRemoved
	}

	duration := time.Since(startTime)
	log.Info("asset discovery completed",
		"duration", duration.String(),
		"platforms", len(totalResults),
		"new", totalNew,
		"updated", totalUpdated,
		"removed", totalRemoved,
	)
}
