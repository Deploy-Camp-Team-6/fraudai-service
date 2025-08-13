package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	migrate "github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jules-labs/go-api-prod-template/internal/clients"
	"github.com/jules-labs/go-api-prod-template/internal/config"
	"github.com/jules-labs/go-api-prod-template/internal/db"
	"github.com/jules-labs/go-api-prod-template/internal/repo"
	"github.com/jules-labs/go-api-prod-template/internal/service"
	httptransport "github.com/jules-labs/go-api-prod-template/internal/transport/http"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	postgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	redis "github.com/testcontainers/testcontainers-go/modules/redis"
)

type IntegrationTestSuite struct {
	*suite.Suite
	pgContainer *postgres.PostgresContainer
	rdContainer *redis.RedisContainer
	pgDSN       string
	redisAddr   string
	server      *httptest.Server
	queries     *db.Queries
}

func (s *IntegrationTestSuite) SetupSuite() {
	ctx := context.Background()
	var err error

	// Start Postgres
	s.pgContainer, err = postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("test-db"),
		postgres.WithUsername("user"),
		postgres.WithPassword("password"),
	)
	require.NoError(s.T(), err)
	s.pgDSN, err = s.pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(s.T(), err)

	// Run migrations
	m, err := migrate.New("file://../../migrations", s.pgDSN)
	require.NoError(s.T(), err)
	err = m.Up()
	require.NoError(s.T(), err)

	// Start Redis
	s.rdContainer, err = redis.Run(ctx, "redis:7-alpine")
	require.NoError(s.T(), err)
	s.redisAddr, err = s.rdContainer.Endpoint(ctx, "")
	require.NoError(s.T(), err)

	// Create app
	cfg := config.Config{
		PGDSN:     s.pgDSN,
		RedisAddr: s.redisAddr,
	}
	dbConn, err := sql.Open("pgx", cfg.PGDSN)
	require.NoError(s.T(), err)

	s.queries = db.New(dbConn)
	redisClient := db.NewRedisClient(cfg.RedisAddr, "", 0)

	userRepo := repo.NewUserRepository(s.queries)
	apiKeyRepo := repo.NewAPIKeyRepository(s.queries)

	profileSvc := service.NewProfileService(userRepo)
	apiKeySvc := service.NewAPIKeyService(apiKeyRepo)
	vendorSvc := service.NewVendorService(clients.NewThirdPartyClient("", "", zerolog.Nop())) // Not needed for this test

	router := httptransport.NewRouter(&cfg, dbConn, redisClient, userRepo, apiKeyRepo, profileSvc, apiKeySvc, vendorSvc, zerolog.Nop())
	s.server = httptest.NewServer(router)
}

func (s *IntegrationTestSuite) TearDownSuite() {
	s.server.Close()
	require.NoError(s.T(), s.pgContainer.Terminate(context.Background()))
	require.NoError(s.T(), s.rdContainer.Terminate(context.Background()))
}

func (s *IntegrationTestSuite) TestProfileEndpoint_APIKeyAuth() {
	// 1. Create a user
	user, err := s.queries.CreateUser(context.Background(), db.CreateUserParams{Email: "test@example.com", Plan: "free"})
	require.NoError(s.T(), err)

	// 2. Create an API key
	apiKeyRepo := repo.NewAPIKeyRepository(s.queries)
	apiKeySvc := service.NewAPIKeyService(apiKeyRepo)
	plainTextKey, _, err := apiKeySvc.CreateAPIKey(context.Background(), user.ID, "test key", 100)
	require.NoError(s.T(), err)

	// 3. Make request
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/v1/profile", s.server.URL), nil)
	req.Header.Set("X-API-Key", plainTextKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(s.T(), err)
	defer func() {
		_ = resp.Body.Close()
	}()

	// 4. Assert
	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)
}

func TestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(IntegrationTestSuite))
}
