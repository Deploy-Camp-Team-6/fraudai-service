package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jules-labs/go-api-prod-template/internal/clients"
	"github.com/jules-labs/go-api-prod-template/internal/config"
	"github.com/jules-labs/go-api-prod-template/internal/db"
	"github.com/jules-labs/go-api-prod-template/internal/observability"
	"github.com/jules-labs/go-api-prod-template/internal/repo"
	"github.com/jules-labs/go-api-prod-template/internal/service"
	httptransport "github.com/jules-labs/go-api-prod-template/internal/transport/http"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger := observability.NewConsoleLogger(cfg.Debug)
	logger.Info().Msgf("starting %s service in %s mode", cfg.OtelServiceName, cfg.AppEnv)

	// Initialize DB
	dbConn, err := db.NewDatabase(cfg.PGDSN, cfg.PGMaxOpenConns, cfg.PGMaxIdleConns, cfg.PGConnMaxLifetime)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer func() {
		_ = dbConn.Close()
	}()

	// Initialize Redis
	redisClient := db.NewRedisClient(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if _, err := redisClient.Ping(context.Background()).Result(); err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to redis")
	}

	// Setup repositories
	queries := db.New(dbConn)
	userRepo := repo.NewUserRepository(queries)
	apiKeyRepo := repo.NewAPIKeyRepository(queries, redisClient, time.Hour)
	logRepo := repo.NewInferenceLogRepository(queries)

	// Read JWT secret
	jwtSecret, err := os.ReadFile(cfg.JWTSecretFile)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to read jwt secret file")
	}

	// Setup services
	profileSvc := service.NewProfileService(userRepo)
	apiKeySvc := service.NewAPIKeyService(apiKeyRepo)
	vendorClient := clients.NewThirdPartyClient(cfg.VendorBaseURL, cfg.VendorToken, logger)
	vendorSvc := service.NewVendorService(vendorClient, logger)
	authSvc := service.NewAuthService(userRepo, jwtSecret, 24*time.Hour)

	// Setup router
	router := httptransport.NewRouter(&cfg, dbConn, redisClient, userRepo, apiKeyRepo, logRepo, profileSvc, apiKeySvc, vendorSvc, authSvc, logger)

	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.HTTPAddr, cfg.HTTPPort),
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("could not start server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info().Msg("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal().Err(err).Msg("server forced to shutdown")
	}

	logger.Info().Msg("server exiting")
}
