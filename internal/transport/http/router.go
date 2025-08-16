package http

import (
	"database/sql"
	"net/http"
	"time"

	chi "github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
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
	logRepo repo.InferenceLogRepository,
	profileSvc service.ProfileService,
	apiKeySvc service.APIKeyService,
	vendorSvc service.VendorService,
	authSvc service.AuthService,
	jwtSecret []byte,
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
		// Auth
		jwtAuth := app_middleware.JWTAuth(jwtSecret, userRepo)
		v1.Route("/auth", func(auth chi.Router) {
			auth.Post("/sign-up", SignUpHandler(authSvc))
			auth.Post("/sign-in", SignInHandler(authSvc))

			auth.Group(func(g chi.Router) {
				g.Use(jwtAuth)
				g.Get("/me", MeHandler(profileSvc))
			})
		})

		v1.Route("/apikeys", func(r chi.Router) {
			r.Use(jwtAuth)
			r.Get("/", ListAPIKeysHandler(apiKeySvc))
			r.Post("/", APIKeyHandler(apiKeySvc))
			r.Delete("/{id}", DeleteAPIKeyHandler(apiKeySvc))
		})

		vendorAuth := app_middleware.AuthEither(
			app_middleware.APIKeyAuth(apiKeyRepo, userRepo),
			jwtAuth,
		)

		v1.Route("/vendor", func(r chi.Router) {
			r.Use(vendorAuth)
			r.Get("/ping", VendorPingHandler(vendorSvc))
		})

		modelsLimiter := app_middleware.RateLimiter(redisClient, "/v1/inference/models", cfg.PredictRateLimit, cfg.PredictRateWindow)
		predictLimiter := app_middleware.RateLimiter(redisClient, "/v1/inference/predict", cfg.PredictRateLimit, cfg.PredictRateWindow)
		fraudPredictLimiter := app_middleware.RateLimiter(redisClient, "/v1/fraud/predict", cfg.PredictRateLimit, cfg.PredictRateWindow)

		v1.Route("/inference", func(r chi.Router) {
			r.Use(vendorAuth)
			r.With(modelsLimiter).Get("/models", ListModelsHandler(vendorSvc))
			r.With(predictLimiter).With(jwtAuth).Post("/predict", PredictHandler(vendorSvc, logRepo, logger))
		})

		v1.Route("/fraud", func(r chi.Router) {
			r.Use(app_middleware.APIKeyAuth(apiKeyRepo, userRepo))
			r.With(fraudPredictLimiter).Post("/predict", PredictHandler(vendorSvc, logRepo, logger))
		})
	})

	return r
}
