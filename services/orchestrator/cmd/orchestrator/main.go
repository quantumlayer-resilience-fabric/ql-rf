// Package main is the entry point for the AI Orchestrator service.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/quantumlayerhq/ql-rf/pkg/config"
	"github.com/quantumlayerhq/ql-rf/pkg/database"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/pkg/models"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/agents"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/executor"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/handlers"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/llm"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/meta"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/notifier"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/temporal/worker"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/tools"
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/validation"
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

	// Check if orchestrator is enabled
	if !cfg.Orchestrator.Enabled {
		slog.Info("AI orchestrator is disabled, exiting")
		return nil
	}

	// Initialize logger
	log := logger.New(cfg.LogLevel, "json")
	log = log.WithService("orchestrator")

	log.Info("starting AI Orchestrator service",
		"version", version,
		"build_time", buildTime,
		"git_commit", gitCommit,
		"env", cfg.Env,
		"llm_provider", cfg.LLM.Provider,
		"llm_model", cfg.LLM.Model,
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

	// Initialize LLM client
	llmClient, err := llm.NewClient(cfg.LLM, log)
	if err != nil {
		return fmt.Errorf("failed to create LLM client: %w", err)
	}
	log.Info("initialized LLM client", "provider", cfg.LLM.Provider)

	// Initialize validation pipeline
	validator, err := validation.NewPipeline(cfg.OPA, log)
	if err != nil {
		return fmt.Errorf("failed to create validation pipeline: %w", err)
	}
	log.Info("initialized validation pipeline", "opa_enabled", cfg.OPA.Enabled)

	// Initialize tool registry
	toolRegistry := tools.NewRegistry(db.Pool, log)
	log.Info("initialized tool registry", "tools", toolRegistry.ListTools())

	// Initialize agent registry
	agentRegistry := agents.NewRegistry(llmClient, toolRegistry, validator, log)
	log.Info("initialized agent registry", "agents", agentRegistry.ListAgents())

	// Initialize meta-prompt engine
	metaEngine := meta.NewEngine(llmClient, agentRegistry, log)
	log.Info("initialized meta-prompt engine")

	// Initialize Temporal worker (optional - depends on Temporal server availability)
	var temporalWorker *worker.Worker
	temporalWorker, err = worker.New(worker.Config{
		Temporal:      cfg.Temporal,
		DB:            db.Pool,
		Logger:        log,
		AgentRegistry: agentRegistry,
		ToolRegistry:  toolRegistry,
	})
	if err != nil {
		log.Warn("failed to create Temporal worker, running without workflows", "error", err)
		temporalWorker = nil
	} else {
		log.Info("initialized Temporal worker",
			"task_queue", cfg.Temporal.TaskQueue,
			"namespace", cfg.Temporal.Namespace,
		)
	}

	// Initialize execution engine
	execEngine := executor.NewEngine(db, toolRegistry, log)
	log.Info("initialized execution engine")

	// Initialize notifier
	notify := notifier.New(cfg.Notifications, log)
	log.Info("initialized notifier",
		"slack_enabled", cfg.Notifications.SlackEnabled,
		"email_enabled", cfg.Notifications.EmailEnabled,
		"webhook_enabled", cfg.Notifications.WebhookEnabled,
	)

	// Timeout for notification and database operations in callbacks
	const callbackTimeout = 30 * time.Second

	// Set execution callbacks for notifications and task state updates
	execEngine.SetCallbacks(
		// On phase start
		func(exec *executor.Execution, phase *executor.PhaseExecution) {
			callbackCtx, cancel := context.WithTimeout(ctx, callbackTimeout)
			defer cancel()
			if err := notify.NotifyPhaseStarted(callbackCtx, exec, phase); err != nil {
				log.Error("failed to send phase start notification", "error", err)
			}
		},
		// On phase complete
		func(exec *executor.Execution, phase *executor.PhaseExecution) {
			callbackCtx, cancel := context.WithTimeout(ctx, callbackTimeout)
			defer cancel()
			if err := notify.NotifyPhaseCompleted(callbackCtx, exec, phase); err != nil {
				log.Error("failed to send phase complete notification", "error", err)
			}
		},
		// On execution done
		func(exec *executor.Execution) {
			callbackCtx, cancel := context.WithTimeout(ctx, callbackTimeout)
			defer cancel()
			if err := notify.NotifyExecutionCompleted(callbackCtx, exec); err != nil {
				log.Error("failed to send execution complete notification", "error", err)
			}
			// Update task state in database
			taskState := "completed"
			if exec.Status == executor.StatusFailed {
				taskState = "failed"
			} else if exec.Status == executor.StatusRolledBack {
				taskState = "rolled_back"
			}
			dbCtx, dbCancel := context.WithTimeout(ctx, 10*time.Second)
			defer dbCancel()
			_, err := db.Pool.Exec(dbCtx,
				`UPDATE ai_tasks SET state = $1, updated_at = NOW() WHERE id = $2`,
				taskState, exec.TaskID,
			)
			if err != nil {
				log.Error("failed to update task state", "error", err, "task_id", exec.TaskID)
			}
		},
	)

	// Create HTTP handler
	handler := handlers.New(handlers.Config{
		DB:             db,
		Config:         cfg,
		Logger:         log,
		LLMClient:      llmClient,
		MetaEngine:     metaEngine,
		AgentRegistry:  agentRegistry,
		ToolRegistry:   toolRegistry,
		Validator:      validator,
		Executor:       execEngine,
		Notifier:       notify,
		TemporalWorker: temporalWorker,
		BuildInfo: handlers.BuildInfo{
			Version:   version,
			BuildTime: buildTime,
			GitCommit: gitCommit,
		},
	})

	// Create HTTP server
	server := &http.Server{
		Addr:         cfg.Orchestrator.Address(),
		Handler:      handler.Router(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second, // Longer for LLM responses
		IdleTimeout:  120 * time.Second,
	}

	// Start server in goroutine
	serverErrors := make(chan error, 1)
	go func() {
		log.Info("starting HTTP server", "addr", server.Addr)
		serverErrors <- server.ListenAndServe()
	}()

	// Initialize and register platform clients for real cloud operations
	// AWS Platform Client
	awsRegion := cfg.Connectors.AWS.Region
	if awsRegion == "" {
		awsRegion = "us-east-1"
	}
	awsClient := executor.NewAWSPlatformClient(executor.AWSClientConfig{
		Region:        awsRegion,
		AssumeRoleARN: cfg.Connectors.AWS.AssumeRoleARN,
	}, log)

	// Azure Platform Client
	azureClient := executor.NewAzurePlatformClient(executor.AzureClientConfig{
		SubscriptionID: cfg.Connectors.Azure.SubscriptionID,
		ResourceGroup:  "", // Will use per-instance resource group from instance ID
		TenantID:       cfg.Connectors.Azure.TenantID,
		ClientID:       cfg.Connectors.Azure.ClientID,
		ClientSecret:   cfg.Connectors.Azure.ClientSecret,
	}, log)

	// GCP Platform Client
	gcpClient := executor.NewGCPPlatformClient(executor.GCPClientConfig{
		ProjectID:       cfg.Connectors.GCP.ProjectID,
		CredentialsFile: cfg.Connectors.GCP.CredentialsFile,
	}, log)

	// Kubernetes Platform Client
	k8sClient := executor.NewKubernetesPlatformClient(executor.KubernetesClientConfig{
		KubeConfig: cfg.Connectors.K8s.Kubeconfig,
		Context:    cfg.Connectors.K8s.Context,
	}, log)

	// vSphere Platform Client
	vsphereClient := executor.NewVSpherePlatformClient(executor.VSphereConfig{
		URL:              cfg.Connectors.VSphere.URL,
		Username:         cfg.Connectors.VSphere.User,
		Password:         cfg.Connectors.VSphere.Password,
		Insecure:         cfg.Connectors.VSphere.Insecure,
		Datacenter:       "", // Use default datacenter
		GuestUsername:    "", // Set from per-operation context or env vars
		GuestPassword:    "", // Set from per-operation context or env vars
		ConnectTimeout:   30 * time.Second,
		OperationTimeout: 10 * time.Minute,
	}, log)

	// Connect enabled platform clients
	for _, connector := range cfg.Connectors.Enabled {
		switch connector {
		case "aws":
			if err := awsClient.Connect(ctx); err != nil {
				log.Warn("failed to connect AWS platform client", "error", err)
			} else {
				log.Info("connected AWS platform client for patch operations", "region", awsRegion)
				execEngine.RegisterPlatformClient("aws", awsClient)
				if temporalWorker != nil {
					temporalWorker.RegisterPlatformClient(models.PlatformAWS, awsClient)
				}
			}

		case "azure":
			if err := azureClient.Connect(ctx); err != nil {
				log.Warn("failed to connect Azure platform client", "error", err)
			} else {
				log.Info("connected Azure platform client for patch operations",
					"subscription_id", cfg.Connectors.Azure.SubscriptionID)
				execEngine.RegisterPlatformClient("azure", azureClient)
				if temporalWorker != nil {
					temporalWorker.RegisterPlatformClient(models.PlatformAzure, azureClient)
				}
			}

		case "gcp":
			if err := gcpClient.Connect(ctx); err != nil {
				log.Warn("failed to connect GCP platform client", "error", err)
			} else {
				log.Info("connected GCP platform client for patch operations",
					"project_id", cfg.Connectors.GCP.ProjectID)
				execEngine.RegisterPlatformClient("gcp", gcpClient)
				if temporalWorker != nil {
					temporalWorker.RegisterPlatformClient(models.PlatformGCP, gcpClient)
				}
			}

		case "k8s":
			if err := k8sClient.Connect(ctx); err != nil {
				log.Warn("failed to connect Kubernetes platform client", "error", err)
			} else {
				log.Info("connected Kubernetes platform client for rolling updates")
				execEngine.RegisterPlatformClient("k8s", k8sClient)
				if temporalWorker != nil {
					temporalWorker.RegisterPlatformClient(models.PlatformK8s, k8sClient)
				}
			}

		case "vsphere":
			if err := vsphereClient.Connect(ctx); err != nil {
				log.Warn("failed to connect vSphere platform client", "error", err)
			} else {
				log.Info("connected vSphere platform client for VM patching operations",
					"url", cfg.Connectors.VSphere.URL)
				execEngine.RegisterPlatformClient("vsphere", vsphereClient)
				if temporalWorker != nil {
					temporalWorker.RegisterPlatformClient(models.PlatformVSphere, vsphereClient)
				}
			}
		}
	}

	// Start Temporal worker if available
	if temporalWorker != nil {
		go func() {
			if err := temporalWorker.Start(); err != nil {
				log.Error("Temporal worker error", "error", err)
			}
		}()
	}

	// Wait for shutdown signal
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server error: %w", err)
		}
	case sig := <-shutdown:
		log.Info("shutdown signal received", "signal", sig.String())

		// Create shutdown context with timeout
		shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 10*time.Second)
		defer shutdownCancel()

		// Stop Temporal worker first
		if temporalWorker != nil {
			log.Info("stopping Temporal worker")
			temporalWorker.Stop()
		}

		// Attempt graceful shutdown
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Error("graceful shutdown failed", "error", err)
			if err := server.Close(); err != nil {
				return fmt.Errorf("forced shutdown error: %w", err)
			}
		}

		log.Info("server shutdown complete")
	}

	return nil
}
