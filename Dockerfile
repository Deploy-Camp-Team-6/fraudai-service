# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Install sqlc and migrate for local development and CI
RUN go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
RUN go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest

RUN CGO_ENABLED=0 GOOS=linux go build -v -o /app/server ./cmd/api

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/server .
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/api/openapi.yaml ./api/openapi.yaml
COPY --from=builder /app/config.yaml ./config.yaml

EXPOSE 8080

CMD ["./server"]
