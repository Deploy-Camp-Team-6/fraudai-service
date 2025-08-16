package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	redis "github.com/redis/go-redis/v9"
)

func TestRateLimiterJWT(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("could not start miniredis: %v", err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	limiter := RateLimiter(client, "/v1/inference/predict", 2, time.Minute)
	handler := limiter(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/v1/inference/predict", nil)
	ctx := context.WithValue(req.Context(), ctxKeyIdentity, Identity{UserID: 1})
	req = req.WithContext(ctx)

	// First request should pass
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	// Second request should also pass
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	// Third request should be rate limited
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status 429, got %d", rr.Code)
	}
	if rr.Header().Get("Retry-After") == "" {
		t.Fatalf("expected Retry-After header to be set")
	}
}

func TestRateLimiterAPIKey(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("could not start miniredis: %v", err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	limiter := RateLimiter(client, "/v1/fraud/predict", 1, time.Minute)
	handler := limiter(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	keyID := int64(10)
	rate := 3
	req := httptest.NewRequest(http.MethodPost, "/v1/fraud/predict", nil)
	ctx := context.WithValue(req.Context(), ctxKeyIdentity, Identity{UserID: 1, APIKeyID: &keyID, RateRPM: &rate})
	req = req.WithContext(ctx)

	// Three requests within limit
	for i := 0; i < 3; i++ {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rr.Code)
		}
	}

	// Fourth request should be rate limited
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status 429, got %d", rr.Code)
	}
}
