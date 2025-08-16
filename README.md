# Production-Ready Go Web API

This project is a template for a production-ready web API written in Go. It includes a variety of best practices and tools for building robust, observable, and maintainable services.

## Features

- **Web Framework**: [chi](https://github.com/go-chi/chi) for lightweight and idiomatic routing.
- **Database**: PostgreSQL with [pgx](https://github.com/jackc/pgx) and type-safe queries via [sqlc](https://github.com/sqlc-dev/sqlc).
- **Migrations**: Handled with [golang-migrate](https://github.com/golang-migrate/migrate).
- **Authentication**: API Key and JWT (HS256) based authentication.
- **Rate Limiting**: Per-user or API key rate limiting using a sliding window algorithm backed by Redis.
- **External API Client**: Resilient external API calls with [resty](https://github.com/go-resty/resty) and [gobreaker](https://github.com/sony/gobreaker).
- **Configuration**: Managed with [viper](https://github.com/spf13/viper), loaded from a YAML file with secrets supplied via environment variables or container secrets.
- **Observability**:
    - Structured logging with [zerolog](https://github.com/rs/zerolog).
    - Metrics exposed for [Prometheus](https://prometheus.io/).
    - Tracing with [OpenTelemetry](https://opentelemetry.io/).
    - Profiling with `net/http/pprof`.
- **Testing**: Unit and integration tests using `testify` and `testcontainers-go`.
- **Containerization**: Multi-stage `Dockerfile` for minimal images, with `docker-compose` for local development and a `stack.yml` for Docker Swarm deployment.
- **CI/CD**: GitHub Actions workflow for testing, building, pushing, and deploying to Docker Swarm.

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

2.  **Configure the application:**
    Non-secret settings live in [`config.yaml`](./config.yaml). Provide secret values such as database credentials via environment variables:
    ```bash
    export PG_DSN="postgres://user:password@localhost:5432/app?sslmode=disable"
    export JWT_SECRET_FILE="jwt.secret"
    export REDIS_ADDR="localhost:6379"
    export REDIS_PASSWORD="password"
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

Configuration values are read from `config.yaml`. Secret values like `PG_DSN`, `REDIS_PASSWORD`, or `VENDOR_TOKEN` should be provided via environment variables or container secrets. See `internal/config/config.go` for all available options.

## Deployment

When the container starts, it automatically applies database migrations before launching the API server. The entrypoint runs:

```sh
migrate -path /root/migrations -database "$PG_DSN" up
./server
```

Ensure the `PG_DSN` environment variable is set (for example via `deploy/stack.yml`) so the migrations can run successfully before the service begins accepting requests.

## Testing

Run all tests:
```bash
make test
```
This will run both unit and integration tests. Integration tests require Docker to be running.