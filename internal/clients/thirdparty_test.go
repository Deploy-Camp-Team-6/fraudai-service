package clients

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/sony/gobreaker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestThirdPartyClient_Ping(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("pong"))
	}))
	defer server.Close()

	client := NewThirdPartyClient(server.URL, "test-token", zerolog.Nop())
	pong, err := client.Ping(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, "pong", pong)
}

func TestThirdPartyClient_Retries(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("pong"))
	}))
	defer server.Close()

	client := NewThirdPartyClient(server.URL, "test-token", zerolog.Nop())
	client.client.SetRetryCount(2).SetRetryWaitTime(10 * time.Millisecond)

	pong, err := client.Ping(context.Background())

	require.NoError(t, err)
	assert.Equal(t, "pong", pong)
	assert.Equal(t, 3, requestCount, "client should make 3 requests total")
}

func TestThirdPartyClient_CircuitBreaker(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewThirdPartyClient(server.URL, "test-token", zerolog.Nop())
	client.client.SetRetryCount(0) // Disable retries
	client.cb = gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:     "test",
		Interval: 2 * time.Second,
		Timeout:  1 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 2
		},
	})

	// Trip the circuit breaker
	_, err := client.Ping(context.Background())
	require.Error(t, err)
	_, err = client.Ping(context.Background())
	require.Error(t, err)
	assert.Equal(t, 2, requestCount)

	// It should be open now
	_, err = client.Ping(context.Background())
	require.Error(t, err)
	assert.True(t, errors.Is(err, gobreaker.ErrOpenState) || (err != nil && err.Error() == "circuit breaker is open"), "error should be from circuit breaker")
	assert.Equal(t, 2, requestCount, "circuit breaker should prevent further requests")

	// Wait for the breaker to enter half-open state
	time.Sleep(2100 * time.Millisecond)

	// It should be half-open now, let's make it succeed
	server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("pong"))
	})

	pong, err := client.Ping(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "pong", pong)
	assert.Equal(t, 3, requestCount)
}
