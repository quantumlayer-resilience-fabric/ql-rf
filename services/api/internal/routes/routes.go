// Package routes configures the HTTP router and middleware.
package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/quantumlayerhq/ql-rf/pkg/compliance"
	"github.com/quantumlayerhq/ql-rf/pkg/config"
	"github.com/quantumlayerhq/ql-rf/pkg/database"
	"github.com/quantumlayerhq/ql-rf/pkg/finops"
	"github.com/quantumlayerhq/ql-rf/pkg/inspec"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
	"github.com/quantumlayerhq/ql-rf/pkg/models"
	"github.com/quantumlayerhq/ql-rf/pkg/multitenancy"
	"github.com/quantumlayerhq/ql-rf/pkg/rbac"
	"github.com/quantumlayerhq/ql-rf/pkg/sbom"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/handlers"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/middleware"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/repository"
	"github.com/quantumlayerhq/ql-rf/services/api/internal/service"
)

// Config holds dependencies for route setup.
type Config struct {
	DB        *database.DB
	Config    *config.Config
	Logger    *logger.Logger
	BuildInfo BuildInfo
}

// BuildInfo contains build information.
type BuildInfo struct {
	Version   string
	BuildTime string
	GitCommit string
}

// New creates a new chi router with all routes and middleware configured.
func New(cfg Config) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(middleware.Logger(cfg.Logger))
	r.Use(middleware.Recoverer(cfg.Logger))
	r.Use(chimiddleware.Compress(5))

	// Rate limiting (enabled by default in production)
	rateLimitCfg := middleware.DefaultRateLimitConfig()
	if cfg.Config.Env == "development" {
		rateLimitCfg.Enabled = false // Disable in dev for easier testing
	}
	r.Use(middleware.RateLimit(rateLimitCfg, cfg.Logger))

	// CORS configuration
	corsOptions := cors.Options{
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}
	if cfg.Config.Env == "development" {
		// In development, allow any localhost port and local network IPs
		corsOptions.AllowedOrigins = []string{
			"http://localhost:*",
			"http://127.0.0.1:*",
			"http://192.168.*:*",
		}
		corsOptions.AllowOriginFunc = func(r *http.Request, origin string) bool {
			// Allow any localhost origin in development
			if len(origin) > 0 && (
				(len(origin) > 17 && origin[:17] == "http://localhost:") ||
				(len(origin) > 18 && origin[:18] == "http://127.0.0.1:") ||
				// Allow local network IPs (192.168.x.x)
				(len(origin) > 15 && origin[:15] == "http://192.168.")) {
				return true
			}
			return false
		}
	} else {
		// In production, only allow specific origins
		corsOptions.AllowedOrigins = []string{
			"http://localhost:3000",
			"https://*.quantumlayer.dev",
		}
	}
	r.Use(cors.Handler(corsOptions))

	// Initialize repository adapters (implement service interfaces)
	imageRepo := repository.NewImageRepositoryAdapter(cfg.DB.Pool)
	assetRepo := repository.NewAssetRepositoryAdapter(cfg.DB.Pool)
	driftRepo := repository.NewDriftRepositoryAdapter(cfg.DB.Pool)
	siteRepo := repository.NewSiteRepositoryAdapter(cfg.DB.Pool)
	alertRepo := repository.NewAlertRepositoryAdapter(cfg.DB.Pool)
	activityRepo := repository.NewActivityRepositoryAdapter(cfg.DB.Pool)
	drPairRepo := repository.NewDRPairRepositoryAdapter(cfg.DB.Pool)

	// Initialize service layer
	imageSvc := service.NewImageService(imageRepo)
	assetSvc := service.NewAssetService(assetRepo)
	driftSvc := service.NewDriftService(driftRepo, assetRepo, imageRepo)
	siteSvc := service.NewSiteService(siteRepo)
	alertSvc := service.NewAlertService(alertRepo)
	overviewSvc := service.NewOverviewService(assetRepo, driftRepo, siteRepo, alertRepo, activityRepo)
	complianceSvc := service.NewComplianceService(cfg.DB, cfg.Logger)
	resilienceSvc := service.NewResilienceService(siteRepo, drPairRepo)
	riskSvc := service.NewRiskService(cfg.DB, cfg.Logger)
	predictionSvc := service.NewPredictionService(cfg.DB, cfg.Logger)

	// Initialize enterprise feature services (RBAC, Multitenancy, Compliance)
	// Convert pgxpool to sql.DB for services that need it
	sqlDB := stdlib.OpenDBFromPool(cfg.DB.Pool)
	rbacSvc := rbac.NewService(sqlDB)
	multitenancySvc := multitenancy.NewService(sqlDB)
	complianceEnhancedSvc := compliance.NewService(sqlDB)

	// Initialize handlers
	healthHandler := handlers.NewHealthHandler(cfg.DB, cfg.BuildInfo.Version, cfg.BuildInfo.GitCommit)
	imageHandler := handlers.NewImageHandler(imageSvc, cfg.Logger)
	assetHandler := handlers.NewAssetHandler(assetSvc, cfg.Logger)
	driftHandler := handlers.NewDriftHandler(driftSvc, cfg.Logger)
	siteHandler := handlers.NewSiteHandler(siteSvc, cfg.Logger)
	alertHandler := handlers.NewAlertHandler(alertSvc, cfg.Logger)
	overviewHandler := handlers.NewOverviewHandler(overviewSvc, cfg.Logger)
	complianceHandler := handlers.NewComplianceHandlerWithSvc(complianceSvc, complianceEnhancedSvc, cfg.Logger)
	resilienceHandler := handlers.NewResilienceHandler(resilienceSvc, cfg.Logger)
	lineageHandler := handlers.NewLineageHandler(cfg.DB.Pool, cfg.Logger)
	riskHandler := handlers.NewRiskHandler(riskSvc, cfg.Logger)
	predictionHandler := handlers.NewPredictionHandler(predictionSvc, cfg.Logger)
	userHandler := handlers.NewUserHandler(cfg.Logger)
	rbacHandler := handlers.NewRBACHandler(rbacSvc, cfg.Logger)
	organizationHandler := handlers.NewOrganizationHandler(multitenancySvc, cfg.Logger)

	// Phase 5 services and handlers (SBOM, FinOps, InSpec)
	// SBOM uses slog.Logger internally (nil defaults to slog.Default())
	sbomSvc := sbom.NewService(sqlDB, nil)
	sbomGenerator := sbom.NewGenerator(sqlDB, nil)
	sbomHandler := handlers.NewSBOMHandler(sbomSvc, sbomGenerator, cfg.Logger)

	finopsSvc := finops.NewCostService(cfg.DB.Pool)
	finopsHandler := handlers.NewFinOpsHandler(finopsSvc, cfg.Logger)

	inspecSvc := inspec.NewService(sqlDB)
	inspecHandler := handlers.NewInSpecHandler(inspecSvc, cfg.Logger)

	// Certificate management handler
	certificateHandler := handlers.NewCertificateHandler(cfg.DB.Pool, cfg.Logger)

	// Health endpoints (no auth required)
	r.Get("/healthz", healthHandler.Liveness)
	r.Get("/readyz", healthHandler.Readiness)
	r.Get("/version", healthHandler.Version)

	// Metrics endpoint
	if cfg.Config.Metrics.Enabled {
		r.Get(cfg.Config.Metrics.Path, healthHandler.Metrics)
	}

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// Apply authentication middleware to all API routes
		// Only use dev mode if Clerk is NOT configured
		clerkConfigured := cfg.Config.Clerk.PublishableKey != "" && cfg.Config.Clerk.SecretKey != ""
		devMode := !clerkConfigured
		authConfig := middleware.AuthConfig{
			ClerkPublishableKey: cfg.Config.Clerk.PublishableKey,
			ClerkSecretKey:      cfg.Config.Clerk.SecretKey,
			DevMode:             devMode,
		}
		tenantConfig := middleware.TenantConfig{
			DB:      cfg.DB.Pool,
			DevMode: devMode,
			Log:     cfg.Logger,
		}
		r.Use(middleware.Auth(authConfig, cfg.Logger))
		r.Use(middleware.Tenant(tenantConfig))

		// Images
		r.Route("/images", func(r chi.Router) {
			// Read operations - require read:images permission
			r.Get("/", imageHandler.List)
			r.Get("/{family}/latest", imageHandler.GetLatest)
			r.Get("/{id}", imageHandler.Get)

			// Write operations - require manage:images permission
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequirePermission(models.PermManageImages))
				r.Post("/", imageHandler.Create)
				r.Patch("/{id}", imageHandler.Update)
				r.Delete("/{id}", imageHandler.Delete)
				r.Post("/{id}/coordinates", imageHandler.AddCoordinate)
				r.Post("/{id}/promote", imageHandler.Promote)
			})

			// Lineage routes for image family tree
			r.Get("/families/{family}/lineage-tree", lineageHandler.GetLineageTree)

			// Lineage routes for individual images
			r.Route("/{id}/lineage", func(r chi.Router) {
				r.Get("/", lineageHandler.GetLineage)
				r.With(middleware.RequirePermission(models.PermManageImages)).
					Post("/parents", lineageHandler.AddParent)
			})

			// Vulnerability tracking
			r.Route("/{id}/vulnerabilities", func(r chi.Router) {
				r.Get("/", lineageHandler.GetVulnerabilities)
				r.With(middleware.RequirePermission(models.PermManageImages)).
					Post("/", lineageHandler.AddVulnerability)
				// Bulk import from scanners (Trivy, Grype, Snyk, etc.)
				r.With(middleware.RequirePermission(models.PermManageImages)).
					Post("/import", lineageHandler.ImportScanResults)
			})

			// Build history and deployments
			r.Get("/{id}/builds", lineageHandler.GetBuilds)
			r.Get("/{id}/deployments", lineageHandler.GetDeployments)
			r.Get("/{id}/components", lineageHandler.GetComponents)

			// SBOM import
			r.With(middleware.RequirePermission(models.PermManageImages)).
				Post("/{id}/sbom", lineageHandler.ImportSBOM)
		})

		// Assets
		r.Route("/assets", func(r chi.Router) {
			r.Get("/", assetHandler.List)
			r.Get("/summary", assetHandler.Summary)
			r.Get("/{id}", assetHandler.Get)
		})

		// Drift
		r.Route("/drift", func(r chi.Router) {
			r.Get("/", driftHandler.GetCurrent)
			r.Get("/summary", driftHandler.Summary)
			r.Get("/trends", driftHandler.Trends)
			r.Get("/top-offenders", driftHandler.TopOffenders)
			r.Get("/reports", driftHandler.ListReports)
			r.Get("/reports/{id}", driftHandler.GetReport)
		})

		// Sites
		r.Route("/sites", func(r chi.Router) {
			r.Get("/", siteHandler.List)
			r.Get("/summary", siteHandler.Summary)
			r.Get("/{id}", siteHandler.Get)
		})

		// Alerts
		r.Route("/alerts", func(r chi.Router) {
			// Read operations
			r.Get("/", alertHandler.List)
			r.Get("/summary", alertHandler.Summary)
			r.Get("/{id}", alertHandler.Get)

			// Alert actions - require acknowledge:alerts permission
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequirePermission(models.PermAcknowledgeAlerts))
				r.Post("/{id}/acknowledge", alertHandler.Acknowledge)
				r.Post("/{id}/resolve", alertHandler.Resolve)
			})
		})

		// Overview
		r.Route("/overview", func(r chi.Router) {
			r.Get("/metrics", overviewHandler.GetMetrics)
		})

		// Compliance
		r.Route("/compliance", func(r chi.Router) {
			// Read operations
			r.Get("/summary", complianceHandler.Summary)
			r.Get("/frameworks", complianceHandler.ListFrameworks)
			r.Get("/controls/failing", complianceHandler.FailingControls)
			r.Get("/images", complianceHandler.ImageCompliance)

			// Enhanced compliance endpoints
			r.Get("/frameworks/{frameworkId}/controls", complianceHandler.ListControls)
			r.Get("/controls/{controlId}/mappings", complianceHandler.GetControlMappings)
			r.Get("/assessments", complianceHandler.ListAssessments)
			r.Post("/assessments", complianceHandler.CreateAssessment)
			r.Get("/assessments/{assessmentId}", complianceHandler.GetAssessment)
			r.Get("/evidence", complianceHandler.ListEvidence)
			r.Post("/evidence", complianceHandler.UploadEvidence)
			r.Get("/exemptions", complianceHandler.ListExemptions)
			r.Post("/exemptions", complianceHandler.CreateExemption)
			r.Get("/score", complianceHandler.GetScore)

			// Audit trigger - require trigger:drill permission (similar to DR drill)
			r.With(middleware.RequirePermission(models.PermTriggerDrill)).
				Post("/audit", complianceHandler.TriggerAudit)
		})

		// Resilience / DR
		r.Route("/resilience", func(r chi.Router) {
			// Read operations
			r.Get("/summary", resilienceHandler.Summary)
			r.Get("/dr-pairs", resilienceHandler.ListDRPairs)
			r.Get("/dr-pairs/{id}", resilienceHandler.GetDRPair)

			// DR actions - require trigger:drill permission
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequirePermission(models.PermTriggerDrill))
				r.Post("/dr-pairs/{id}/test", resilienceHandler.TriggerFailoverTest)
				r.Post("/dr-pairs/{id}/sync", resilienceHandler.TriggerSync)
			})
		})

		// Risk Scoring
		r.Route("/risk", func(r chi.Router) {
			r.Get("/summary", riskHandler.Summary)
			r.Get("/top", riskHandler.TopRisks)

			// Predictive risk endpoints
			r.Get("/forecast", predictionHandler.GetForecast)
			r.Get("/recommendations", predictionHandler.GetRecommendations)
			r.Get("/anomalies", predictionHandler.GetAnomalies)
			r.Get("/assets/{id}/prediction", predictionHandler.GetAssetPrediction)
		})

		// Users
		r.Route("/users", func(r chi.Router) {
			r.Get("/me", userHandler.GetCurrentUser)
		})

		// RBAC
		r.Route("/rbac", func(r chi.Router) {
			// Roles
			r.Get("/roles", rbacHandler.ListRoles)
			r.Get("/roles/{roleId}", rbacHandler.GetRole)
			r.Get("/permissions", rbacHandler.ListPermissions)

			// User roles
			r.Get("/users/{userId}/roles", rbacHandler.GetUserRoles)
			r.Post("/users/{userId}/roles", rbacHandler.AssignRole)
			r.Delete("/users/{userId}/roles/{roleId}", rbacHandler.RevokeRole)
			r.Get("/users/{userId}/permissions", rbacHandler.GetUserPermissions)

			// Permission checking
			r.Post("/check", rbacHandler.CheckPermission)

			// Teams
			r.Get("/teams", rbacHandler.ListTeams)
			r.Post("/teams", rbacHandler.CreateTeam)
			r.Get("/teams/{teamId}/members", rbacHandler.GetTeamMembers)
			r.Post("/teams/{teamId}/members", rbacHandler.AddTeamMember)
		})

		// Organization / Multi-tenancy
		r.Route("/organization", func(r chi.Router) {
			r.Get("/", organizationHandler.GetCurrentOrganization)
			r.Get("/quota", organizationHandler.GetQuota)
			r.Get("/usage", organizationHandler.GetUsage)
			r.Get("/quota-status", organizationHandler.GetQuotaStatus)
			r.Get("/subscription", organizationHandler.GetSubscription)
			r.Get("/plans", organizationHandler.ListPlans)
			r.Get("/check", organizationHandler.CheckUserOrganization)
			r.Post("/seed-demo", organizationHandler.SeedDemoData)
		})

		// Organizations (create and manage)
		r.Route("/organizations", func(r chi.Router) {
			r.Post("/", organizationHandler.CreateOrganization)
		})

		// SBOM (Software Bill of Materials)
		r.Route("/sbom", func(r chi.Router) {
			r.Get("/", sbomHandler.ListSBOMs)
			r.Get("/{id}", sbomHandler.GetSBOM)
			r.Delete("/{id}", sbomHandler.DeleteSBOM)
			r.Get("/{id}/export", sbomHandler.ExportSBOM)
			r.Get("/{id}/vulnerabilities", sbomHandler.GetSBOMVulnerabilities)
		})

		// Additional SBOM routes under images
		r.Route("/images/{id}/sbom", func(r chi.Router) {
			r.Get("/", sbomHandler.GetImageSBOM)
			r.With(middleware.RequirePermission(models.PermManageImages)).
				Post("/generate", sbomHandler.GenerateSBOM)
		})

		// FinOps (Cost Optimization)
		r.Route("/finops", func(r chi.Router) {
			r.Get("/summary", finopsHandler.GetSummary)
			r.Get("/breakdown", finopsHandler.GetBreakdown)
			r.Get("/trend", finopsHandler.GetTrend)
			r.Get("/recommendations", finopsHandler.GetRecommendations)
			r.Get("/resources", finopsHandler.GetResourceCosts)

			// Budgets
			r.Get("/budgets", finopsHandler.ListBudgets)
			r.Post("/budgets", finopsHandler.CreateBudget)
		})

		// InSpec (Compliance Scanning)
		r.Route("/inspec", func(r chi.Router) {
			// Profiles
			r.Get("/profiles", inspecHandler.ListProfiles)
			r.Get("/profiles/{profileId}", inspecHandler.GetProfile)
			r.Post("/profiles", inspecHandler.CreateProfile)
			r.Get("/profiles/{profileId}/mappings", inspecHandler.GetControlMappings)
			r.Post("/profiles/{profileId}/mappings", inspecHandler.CreateControlMapping)

			// Runs
			r.Post("/run", inspecHandler.RunProfile)
			r.Get("/runs", inspecHandler.ListRuns)
			r.Get("/runs/{runId}", inspecHandler.GetRun)
			r.Get("/runs/{runId}/results", inspecHandler.GetRunResults)
			r.Post("/runs/{runId}/cancel", inspecHandler.CancelRun)
		})

		// Certificates (Certificate Lifecycle Management)
		r.Route("/certificates", func(r chi.Router) {
			// Read operations
			r.Get("/", certificateHandler.ListCertificates)
			r.Get("/summary", certificateHandler.GetCertificateSummary)
			r.Get("/{id}", certificateHandler.GetCertificate)
			r.Get("/{id}/usage", certificateHandler.GetCertificateUsage)

			// Rotations
			r.Get("/rotations", certificateHandler.ListRotations)
			r.Get("/rotations/{id}", certificateHandler.GetRotation)

			// Alerts
			r.Get("/alerts", certificateHandler.ListAlerts)
			r.With(middleware.RequirePermission(models.PermAcknowledgeAlerts)).
				Post("/alerts/{id}/acknowledge", certificateHandler.AcknowledgeAlert)
		})
	})

	return r
}
