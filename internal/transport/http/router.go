package http

import (
	"database/sql"
	"net/http"
	"time"

	chi "github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	redis_rate "github.com/go-redis/redis_rate/v10"
	"github.com/jules-labs/go-api-prod-template/internal/config"
	"github.com/jules-labs/go-api-prod-template/internal/repo"
	"github.com/jules-labs/go-api-prod-template/internal/service"
	app_middleware "github.com/jules-labs/go-api-prod-template/internal/transport/http/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	redis "github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

func NewRouter(
	cfg *config.Config,
	db *sql.DB,
	redisClient *redis.Client,
	userRepo repo.UserRepository,
	apiKeyRepo repo.APIKeyRepository,
	profileSvc service.ProfileService,
	apiKeySvc service.APIKeyService,
	vendorSvc service.VendorService,
	authSvc service.AuthService,
	logger zerolog.Logger,
) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(app_middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(app_middleware.Logger(logger))
	r.Use(middleware.Recoverer)
	r.Use(app_middleware.CORS(cfg.CORSAllowedOrigins))
	r.Use(middleware.Timeout(15 * time.Second))

	// Observability
	r.Handle("/metrics", promhttp.Handler())
	if cfg.Debug {
		r.Mount("/debug", middleware.Profiler())
	}

	// Health/readiness
	r.Get("/healthz", HealthzHandler)
	r.Get("/readyz", ReadinessHandler(db, redisClient))

	// Docs
	r.Get("/swagger/*", func(w http.ResponseWriter, r *http.Request) {
		// Placeholder for swagger
		http.ServeFile(w, r, "api/openapi.yaml")
	})

	// API v1
	r.Route("/v1", func(v1 chi.Router) {
		// Public auth routes
		v1.Post("/auth/sign-up", SignUpHandler(authSvc))
		v1.Post("/auth/sign-in", SignInHandler(authSvc))

		// Protected routes
		v1.Group(func(protected chi.Router) {
			apiKeyAuth := app_middleware.APIKeyAuth(apiKeyRepo, userRepo)
			jwtAuth := app_middleware.JWTAuth(cfg.JWTSecretFile, userRepo)

			// Rate limiting middleware
			limiter := redis_rate.NewLimiter(redisClient)
			protected.Use(app_middleware.PlanAwareRateLimiter(limiter, cfg.RateLimitRPMDefault))

			// Routes with optional auth (API Key or JWT)
			protected.Group(func(either chi.Router) {
				either.Use(app_middleware.AuthEither(apiKeyAuth, jwtAuth))
				either.Get("/profile", ProfileHandler(profileSvc))
				either.Get("/vendor/ping", VendorPingHandler(vendorSvc))
			})

			// Routes with JWT-only auth
			protected.Group(func(jwtOnly chi.Router) {
				jwtOnly.Use(jwtAuth)
				jwtOnly.Get("/auth/me", GetMeHandler(profileSvc))
				jwtOnly.Post("/apikeys", APIKeyHandler(apiKeySvc))
			})
		})
	})

	return r
}
