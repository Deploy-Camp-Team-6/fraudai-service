package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/jules-labs/go-api-prod-template/internal/transport/http/response"
	redis "github.com/redis/go-redis/v9"
)

// RateLimiter enforces a rate limit for the given endpoint on a per-identity basis.
// For JWT-authenticated requests, the provided jwtLimit is used. For API key requests
// the limit is taken from the identity's RateRPM value. Limits are tracked in Redis
// to ensure consistent enforcement across distributed instances.
func RateLimiter(redisClient *redis.Client, endpoint string, jwtLimit int, window time.Duration) func(http.Handler) http.Handler {
	script := redis.NewScript(`
local current = redis.call("INCR", KEYS[1])
if tonumber(current) == 1 then
    redis.call("EXPIRE", KEYS[1], ARGV[1])
end
return current
`)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			identity, ok := IdentityFrom(r.Context())
			if !ok {
				// This should not happen if auth middleware is properly configured
				response.RespondWithError(w, http.StatusInternalServerError, "identity not found in context")
				return
			}

			limit := jwtLimit
			identifier := fmt.Sprintf("user:%d", identity.UserID)
			if identity.APIKeyID != nil {
				identifier = fmt.Sprintf("apikey:%d", *identity.APIKeyID)
				if identity.RateRPM != nil {
					limit = *identity.RateRPM
				}
			}

			now := time.Now().UTC()
			windowStart := now.Truncate(window).Unix()
			key := fmt.Sprintf("ratelimit:%s:%s:%d", identifier, endpoint, windowStart)

			current, err := script.Run(r.Context(), redisClient, []string{key}, int(window.Seconds())).Int()
			if err != nil {
				response.RespondWithError(w, http.StatusInternalServerError, "rate limit check failed")
				return
			}

			ttl, err := redisClient.TTL(r.Context(), key).Result()
			if err != nil {
				response.RespondWithError(w, http.StatusInternalServerError, "rate limit ttl failed")
				return
			}

			remaining := limit - current
			if remaining < 0 {
				remaining = 0
			}

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(int64(ttl.Seconds()), 10))

			if current > limit {
				w.Header().Set("Retry-After", strconv.FormatInt(int64(ttl.Seconds()), 10))
				response.RespondWithError(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
