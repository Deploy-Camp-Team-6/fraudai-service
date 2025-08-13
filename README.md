# Production-Ready Go Web API

This project is a template for a production-ready web API written in Go. It includes a variety of best practices and tools for building robust, observable, and maintainable services.

## Features

- **Web Framework**: [chi](https://github.com/go-chi/chi) for lightweight and idiomatic routing.
- **Database**: PostgreSQL with [pgx](https://github.com/jackc/pgx) and type-safe queries via [sqlc](https://github.com/sqlc-dev/sqlc).
- **Migrations**: Handled with [golang-migrate](https://github.com/golang-migrate/migrate).
- **Authentication**: API Key and JWT (HS256) based authentication.
- **Rate Limiting**: Per-API key rate limiting using a sliding window algorithm with [redis_rate](https://github.com/go-redis/redis_rate).
- **External API Client**: Resilient external API calls with [resty](https://github.com/go-resty/resty) and [gobreaker](https://github.com/sony/gobreaker).
- **Configuration**: Managed with [viper](https://github.com/spf13/viper), loaded from environment variables.
- **Observability**:
    - Structured logging with [zerolog](https://github.com/rs/zerolog).
    - Metrics exposed for [Prometheus](https://prometheus.io/).
    - Tracing with [OpenTelemetry](https://opentelemetry.io/).
    - Profiling with `net/http/pprof`.
- **Testing**: Unit and integration tests using `testify` and `testcontainers-go`.
- **Containerization**: Multi-stage `Dockerfile` for minimal images, with `docker-compose` for local development and a `stack.yml` for Docker Swarm deployment.
- **CI/CD**: GitHub Actions workflow for automated testing, linting, and image publishing.

## Getting Started

### Prerequisites

- Go 1.22+
- Docker and Docker Compose
- `make`
- `migrate` CLI: `go install -v github.com/golang-migrate/migrate/v4/cmd/migrate@latest`
- `sqlc` CLI: `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`

### Local Development

1.  **Start services:**
    ```bash
    make db-up
    ```

2.  **Set environment variables:**
    Create a `.env` file in the root of the project:
    ```env
    APP_ENV=development
    HTTP_PORT=8080
    PG_DSN="postgres://user:password@localhost:5432/app?sslmode=disable"
    REDIS_ADDR="localhost:6379"
    JWT_SECRET_FILE="jwt.secret"
    DEBUG=true
    ```
    Create the `jwt.secret` file:
    ```bash
    echo "super-secret" > jwt.secret
    ```

3.  **Run migrations:**
    ```bash
    make db-migrate
    ```

4.  **Generate database code:**
    ```bash
    make sqlc
    ```

5.  **Run the application:**
    ```bash
    make run
    ```
The API will be available at `http://localhost:8080`.

### API Usage

**Health Check:**
```bash
curl http://localhost:8080/healthz
```

**Readiness Check:**
```bash
curl http://localhost:8080/readyz
```

**Create User (Example):**
This is not an endpoint, you would create users via a different mechanism or a seed script.

**Create API Key:**
First, you need a JWT for a user. You can generate one with a tool like [jwt.io](https://jwt.io) with a payload like `{"user_id": 1}`.

```bash
curl -X POST http://localhost:8080/v1/apikeys \
  -H "Authorization: Bearer <your-jwt>" \
  -H "Content-Type: application/json" \
  -d '{"label": "my-test-key"}'
```

**Get Profile (API Key):**
```bash
curl http://localhost:8080/v1/profile \
  -H "X-API-Key: <your-api-key>"
```

**Get Profile (JWT):**
```bash
curl http://localhost:8080/v1/profile-jwt \
  -H "Authorization: Bearer <your-jwt>"
```

## Configuration

All configuration is managed via environment variables. See `internal/config/config.go` for all available options.

## Testing

Run all tests:
```bash
make test
```
This will run both unit and integration tests. Integration tests require Docker to be running.
