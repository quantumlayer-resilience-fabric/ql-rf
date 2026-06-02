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
	"strconv"
	"strings"
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
	"github.com/quantumlayerhq/ql-rf/services/orchestrator/internal/vulnmatch"
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

	// PR #19 / CONN-001: register real AWS tools when credentials are
	// available, with optional fallback to a deterministic mock client for
	// dev/CI. The mock-client path is gated by RF_CONNECTORS_AWS_FALLBACK_TO_MOCK
	// and emits a loud WARN log when active so production never silently
	// serves fake data.
	registerAWSCloudTools(ctx, cfg, toolRegistry, log)

	// PR #26 / CONN-006: same pattern for Azure. Real client validated
	// via a cheap list-VMs page; falls back to mock when credentials are
	// absent and fallback_to_mock=true.
	registerAzureCloudTools(ctx, cfg, toolRegistry, log)

	// PR #20 / CONN-002: register the state-change SSM patch tool in
	// dry-run mode. Neither the real nor mock SSM client makes any network
	// call (the SDK isn't even imported in this PR's code), so this is
	// safe to register unconditionally. Live mode lands as a follow-up
	// PR with explicit env opt-in.
	registerSSMStateChangeTools(cfg, toolRegistry, log)

	// PR #21 / CONN-003: register the LIVE SSM patch tool when explicitly
	// opted in via RF_CONNECTORS_AWS_ALLOW_LIVE_PATCH=true. Refuses to
	// register (and may exit boot loudly) if the configuration is
	// incoherent — see registerSSMLiveTools for the exact gates.
	registerSSMLiveTools(ctx, cfg, toolRegistry, log)

	// PR #27 / CONN-007: register the Azure Run Command dry-run tool.
	// Both real and mock clients here make ZERO network calls — the
	// state-change Azure SDK constructor is excluded from this file by
	// a structural Go test (no_azure_runcommand_sdk_import_test.go).
	// Registration is unconditional and safe.
	registerAzureRunCommandDryRunTools(cfg, toolRegistry, log)

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

	// Start the vulnerability matcher (opt-in background scanner).
	// Core logic lives in vulnmatch.Service.ScanAndAlert (a RunOnce entrypoint);
	// this goroutine is only a ticker loop around it, so the same logic can later
	// be driven by a dedicated worker or a Temporal cron without changes.
	if vulnScanEnabled(cfg.Env) {
		interval := vulnScanInterval(log)
		vulnSvc := vulnmatch.NewService(db.Pool, log.Logger)
		log.Info("starting vulnerability scanner", "interval", interval.String())
		go func() {
			runScan := func() {
				scanCtx, scanCancel := context.WithTimeout(ctx, 10*time.Minute)
				defer scanCancel()
				if _, err := vulnSvc.ScanAndAlert(scanCtx); err != nil {
					log.Error("vulnerability scan failed", "error", err)
				}
			}
			runScan() // initial pass on startup
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					runScan()
				}
			}
		}()
	} else {
		log.Info("vulnerability scanner disabled (set RF_VULN_SCAN_ENABLED=true to enable)")
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

// vulnScanEnabled reports whether the background vulnerability scanner should run.
// It is opt-in: enabled by default only in development; elsewhere it must be
// explicitly enabled via RF_VULN_SCAN_ENABLED=true. An invalid value falls back
// to the development default.
func vulnScanEnabled(env string) bool {
	v := strings.TrimSpace(os.Getenv("RF_VULN_SCAN_ENABLED"))
	if v == "" {
		return env == "development"
	}
	enabled, err := strconv.ParseBool(v)
	if err != nil {
		return env == "development"
	}
	return enabled
}

// vulnScanInterval returns the scan interval from RF_VULN_SCAN_INTERVAL, falling
// back to 15m (with a warning) when unset, invalid, or non-positive.
func vulnScanInterval(log *logger.Logger) time.Duration {
	const def = 15 * time.Minute
	v := strings.TrimSpace(os.Getenv("RF_VULN_SCAN_INTERVAL"))
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		log.Warn("invalid RF_VULN_SCAN_INTERVAL; falling back to default",
			"value", v, "default", def.String())
		return def
	}
	return d
}

// registerAWSCloudTools is the PR #19 / CONN-001 boot hook for the orchestrator
// tool layer's real cloud surface. It tries the real AWS client first
// (validates creds via STS GetCallerIdentity). If that fails AND fallback is
// enabled, it falls back to a deterministic mock client with a loud WARN
// log. If neither path works, no AWS tools are registered and the
// orchestrator boots normally — query_aws_instances simply doesn't exist
// that day.
//
// Three operational modes:
//   - Real client succeeds → tool registered, info log "aws tools: real client initialized".
//   - Real client fails, fallback_to_mock=true → mock registered, loud WARN.
//   - Real client fails, fallback_to_mock=false → nothing registered, info log.
func registerAWSCloudTools(
	ctx context.Context,
	cfg *config.Config,
	reg *tools.Registry,
	log *logger.Logger,
) {
	awsCfg := cfg.Connectors.AWS

	bootCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	realClient, realErr := tools.NewRealAWSClient(bootCtx, awsCfg, log)
	if realErr == nil {
		log.Info("aws tools: real client initialized",
			"region", awsCfg.Region,
			"assume_role", awsCfg.AssumeRoleARN != "",
		)
		reg.RegisterCloudTools(realClient)
		return
	}

	if !awsCfg.FallbackToMock {
		log.Info("aws tools: real client unavailable and fallback_to_mock=false, not registering aws tools",
			"hint", "set RF_CONNECTORS_AWS_FALLBACK_TO_MOCK=true for dev/CI",
			"error", realErr.Error(),
		)
		return
	}

	log.Warn("aws tools: MOCK CLIENT ACTIVE — instances returned are FAKE. DO NOT USE IN PRODUCTION.",
		"fallback_reason", realErr.Error(),
	)
	reg.RegisterCloudTools(tools.NewMockAWSClient())
}

// registerAzureCloudTools is the PR #26 / CONN-006 boot hook for the
// Azure read-only tool surface. Mirrors registerAWSCloudTools exactly:
//
//   - Real client succeeds → tool registered, info log.
//   - Real client fails, fallback_to_mock=true → mock registered, loud WARN.
//   - Real client fails, fallback_to_mock=false → nothing registered, info log.
//
// The real client validates credentials at construction by listing one
// page of VMs (the cheapest Resource Manager call that proves auth +
// reachability). PR #27/#28 add state-change Azure tools through their
// own boot hooks (registerAzureRunCommandTools / *Live), preserving the
// same SDK-isolation pattern used for SSM in PR #20/#21.
func registerAzureCloudTools(
	ctx context.Context,
	cfg *config.Config,
	reg *tools.Registry,
	log *logger.Logger,
) {
	azCfg := cfg.Connectors.Azure

	bootCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	realClient, realErr := tools.NewRealAzureClient(bootCtx, azCfg, log)
	if realErr == nil {
		log.Info("azure tools: real client initialized",
			"subscription_id", azCfg.SubscriptionID,
			"tenant_id", azCfg.TenantID,
		)
		reg.RegisterAzureCloudTools(realClient)
		return
	}

	if !azCfg.FallbackToMock {
		log.Info("azure tools: real client unavailable and fallback_to_mock=false, not registering azure tools",
			"hint", "set RF_CONNECTORS_AZURE_FALLBACK_TO_MOCK=true for dev/CI",
			"error", realErr.Error(),
		)
		return
	}

	log.Warn("azure tools: MOCK CLIENT ACTIVE — VMs returned are FAKE. DO NOT USE IN PRODUCTION.",
		"fallback_reason", realErr.Error(),
	)
	reg.RegisterAzureCloudTools(tools.NewMockAzureClient())
}

// registerSSMStateChangeTools is the PR #20 / CONN-002 boot hook for the
// state-change cloud surface. Registers ssm_send_patch_command in dry-run
// mode. Both the "real" and "mock" SSM clients in PR #20 make ZERO network
// calls — the SDK isn't imported in this PR's code, enforced by a depguard
// rule in .golangci.yml. So registration is unconditional and safe.
//
// The mock client is more aggressive (returns a fixed two-instance plan
// regardless of input); the real client validates instance IDs strictly.
// CI uses mock (via fallback_to_mock); dev with creds uses real.
//
// PR #21 will introduce a separate `live_ssm_client.go` that DOES import
// the SDK and exposes a Send method, gated by env opt-in + per-instance
// whitelist + two-approver workflow.
func registerSSMStateChangeTools(
	cfg *config.Config,
	reg *tools.Registry,
	log *logger.Logger,
) {
	var ssmClient tools.SSMClient
	if cfg.Connectors.AWS.FallbackToMock {
		log.Warn("ssm tools: MOCK CLIENT ACTIVE — patch plans use fixed i-mock-* instance IDs. DO NOT USE IN PRODUCTION.",
			"reason", "RF_CONNECTORS_AWS_FALLBACK_TO_MOCK=true",
		)
		ssmClient = tools.NewMockSSMClient()
	} else {
		ssmClient = tools.NewRealSSMClient(log)
		log.Info("ssm tools: real client initialized (dry-run only in PR #20)")
	}
	reg.RegisterStateChangeTools(ssmClient)
}

// registerAzureRunCommandDryRunTools is the PR #27 / CONN-007 boot hook
// for the Azure state-change cloud surface. Registers `azure_run_command`
// in dry-run mode. Both the "real" and "mock" Azure Run Command clients
// in PR #27 make ZERO network calls — the state-change SDK constructor
// is excluded from this package by name-based static check (see
// no_azure_runcommand_sdk_import_test.go). So registration is
// unconditional and safe.
//
// The mock client is the more aggressive variant (returns a fixed mock-vm
// regardless of input); the real client validates resource group + VM
// naming rules strictly. CI uses mock (via fallback_to_mock); dev with
// creds uses real.
//
// PR #28 will introduce `live_azure_runcommand_client.go` that DOES call
// the state-change SDK and exposes a Send method, gated by env opt-in +
// per-VM whitelist + two-approver workflow.
func registerAzureRunCommandDryRunTools(
	cfg *config.Config,
	reg *tools.Registry,
	log *logger.Logger,
) {
	var azClient tools.AzureRunCommandClient
	if cfg.Connectors.Azure.FallbackToMock {
		log.Warn("azure run-command tools: MOCK CLIENT ACTIVE — plans use fixed mock-vm-* VM names. DO NOT USE IN PRODUCTION.",
			"reason", "RF_CONNECTORS_AZURE_FALLBACK_TO_MOCK=true",
		)
		azClient = tools.NewMockAzureRunCommandClient()
	} else {
		azClient = tools.NewRealAzureRunCommandClient(log)
		log.Info("azure run-command tools: real client initialized (dry-run only in PR #27)")
	}
	reg.RegisterAzureRunCommandDryRunTools(azClient)
}

// registerSSMLiveTools is the PR #21 / CONN-003 boot hook for the LIVE
// state-change cloud surface. The first code path in the orchestrator
// that can fire a real cloud-mutating API call.
//
// Four gates, evaluated in order:
//
//  1. AllowLivePatch off (default): silent no-op. Production deployments
//     that don't want live mode never see anything from this function.
//  2. AllowLivePatch on + FallbackToMock on: REFUSE TO START. The
//     combination is incoherent and the only safe response is to fail
//     loudly at boot rather than serve mixed real/mock behavior.
//  3. AllowLivePatch on + whitelist empty: REFUSE TO START. Live mode
//     without targets is meaningless and probably indicates a missing
//     env var.
//  4. All gates pass: emit a loud WARN log with the whitelist contents
//     and a "real cloud mutations possible" string, then register the
//     live tool with either the real SDK client (default) or a
//     deterministic mock (LivePatchClientMode="mock", for local smoke
//     and integration tests where we want to exercise the full live-mode
//     boot path without firing real AWS calls).
//
// The OPA tool_authorization policy adds the runtime gate (two distinct
// approvers required) at invocation time.
func registerSSMLiveTools(
	ctx context.Context,
	cfg *config.Config,
	reg *tools.Registry,
	log *logger.Logger,
) {
	awsCfg := cfg.Connectors.AWS
	if !awsCfg.AllowLivePatch {
		// Silent no-op. The dry-run tool from PR #20 stays the only
		// state-change SSM surface in this deployment.
		return
	}

	if awsCfg.FallbackToMock {
		log.Error("REFUSING TO START: RF_CONNECTORS_AWS_ALLOW_LIVE_PATCH=true and RF_CONNECTORS_AWS_FALLBACK_TO_MOCK=true are incoherent. Pick one.",
			"hint", "Production live mode requires real AWS credentials, not the mock client.",
		)
		os.Exit(1)
	}

	// Pull whitelist from env (preferred — keeps secrets out of config
	// files) with config-derived list as the documented fallback.
	whitelist := tools.LoadInstanceWhitelistFromEnv()
	if len(whitelist) == 0 {
		whitelist = awsCfg.LivePatchWhitelistInstanceIDs
	}
	if len(whitelist) == 0 {
		log.Error("REFUSING TO START: RF_CONNECTORS_AWS_ALLOW_LIVE_PATCH=true but no instances on the whitelist.",
			"hint", "Set RF_CONNECTORS_AWS_LIVE_PATCH_WHITELIST_INSTANCE_IDS=i-xxx,i-yyy or connectors.aws.live_patch_whitelist_instance_ids in config.",
		)
		os.Exit(1)
	}

	// LOUD WARN: by the time we reach here, real cloud mutations are
	// possible. Log every relevant boot-state field so post-incident
	// triage has the configuration verbatim.
	log.Warn("LIVE SSM MODE ENABLED — real cloud mutations possible after two-approver workflow",
		"client_mode", awsCfg.LivePatchClientMode,
		"whitelist_count", len(whitelist),
		"whitelist", whitelist,
		"region", awsCfg.Region,
		"assume_role_arn", awsCfg.AssumeRoleARN,
	)

	// Build the live client. "mock" mode short-circuits the SDK so local
	// smoke + integration tests exercise the full boot path without
	// touching AWS. Production MUST run with client_mode="real".
	var liveClient tools.LiveSSMClient
	if awsCfg.LivePatchClientMode == "mock" {
		log.Warn("ssm live tools: MOCK CLIENT ACTIVE — SendCommand returns cmd-mock-* without calling AWS. DO NOT USE IN PRODUCTION.")
		liveClient = tools.NewMockLiveSSMClient(whitelist)
	} else {
		realLive, err := tools.NewLiveSSMClient(ctx, awsCfg, whitelist, log)
		if err != nil {
			log.Error("REFUSING TO START: cannot construct live SSM client",
				"error", err.Error(),
				"hint", "Check AWS credentials, region, and assume_role_arn.",
			)
			os.Exit(1)
		}
		liveClient = realLive
	}

	// The dry client is reused from PR #20 for plan construction; the
	// live client takes over for the actual SendCommand call.
	reg.RegisterLiveStateChangeTools(tools.NewRealSSMClient(log), liveClient)
}
