// Package routes configures the HTTP router and middleware.
package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/quantumlayerhq/ql-rf/pkg/config"
	"github.com/quantumlayerhq/ql-rf/pkg/database"
	"github.com/quantumlayerhq/ql-rf/pkg/logger"
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

	// CORS configuration
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "https://*.quantumlayer.dev"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Initialize repository adapters (implement service interfaces)
	imageRepo := repository.NewImageRepositoryAdapter(cfg.DB.Pool)
	assetRepo := repository.NewAssetRepositoryAdapter(cfg.DB.Pool)
	driftRepo := repository.NewDriftRepositoryAdapter(cfg.DB.Pool)

	// Initialize service layer
	imageSvc := service.NewImageService(imageRepo)
	assetSvc := service.NewAssetService(assetRepo)
	driftSvc := service.NewDriftService(driftRepo, assetRepo, imageRepo)

	// Initialize handlers
	healthHandler := handlers.NewHealthHandler(cfg.DB, cfg.BuildInfo.Version, cfg.BuildInfo.GitCommit)
	imageHandler := handlers.NewImageHandler(imageSvc, cfg.Logger)
	assetHandler := handlers.NewAssetHandler(assetSvc, cfg.Logger)
	driftHandler := handlers.NewDriftHandler(driftSvc, cfg.Logger)

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
		r.Use(middleware.Auth(cfg.Config.Clerk.SecretKey, cfg.Logger))
		r.Use(middleware.Tenant(cfg.DB, cfg.Logger))

		// Images
		r.Route("/images", func(r chi.Router) {
			r.Get("/", imageHandler.List)
			r.Post("/", imageHandler.Create)
			r.Get("/{family}/latest", imageHandler.GetLatest)
			r.Get("/{id}", imageHandler.Get)
			r.Patch("/{id}", imageHandler.Update)
			r.Delete("/{id}", imageHandler.Delete)
			r.Post("/{id}/coordinates", imageHandler.AddCoordinate)
			r.Post("/{id}/promote", imageHandler.Promote)
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
			r.Get("/reports", driftHandler.ListReports)
			r.Get("/reports/{id}", driftHandler.GetReport)
		})

		// Organizations (admin only)
		r.Route("/organizations", func(r chi.Router) {
			r.Use(middleware.RequireRole("admin"))
			r.Get("/", handlers.NotImplemented)
			r.Post("/", handlers.NotImplemented)
		})
	})

	return r
}
