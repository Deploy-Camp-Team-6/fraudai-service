# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# tooling (optional, handy in CI)
RUN go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
RUN go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest

RUN CGO_ENABLED=0 GOOS=linux go build -v -o /app/server ./cmd/api

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates wget

WORKDIR /root/

COPY --from=builder /app/server .
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/api/openapi.yaml ./api/openapi.yaml
COPY --from=builder /app/config.yaml ./config.yaml
COPY --from=builder /go/bin/migrate /usr/local/bin/migrate
COPY --from=builder /app/entrypoint.sh ./entrypoint.sh

RUN chmod +x /root/entrypoint.sh

EXPOSE 8080
ENTRYPOINT ["./entrypoint.sh"]
