package middleware

import (
	"net/http"
	"strconv"

	redis_rate "github.com/go-redis/redis_rate/v10"
	"github.com/jules-labs/go-api-prod-template/internal/transport/http/response"
)

func PlanAwareRateLimiter(limiter *redis_rate.Limiter, defaultRate int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			identity, ok := IdentityFrom(r.Context())
			if !ok {
				// This should not happen if auth middleware is properly configured
				response.RespondWithError(w, http.StatusInternalServerError, "identity not found in context")
				return
			}

			var rateLimitKey string
			var rate int

			if identity.APIKeyID != nil && identity.RateRPM != nil {
				// API Key auth
				rateLimitKey = "apikey:" + strconv.FormatInt(*identity.APIKeyID, 10)
				rate = *identity.RateRPM
			} else {
				// JWT auth
				rateLimitKey = "user:" + strconv.FormatInt(identity.UserID, 10)
				// Determine rate based on plan
				switch identity.Plan {
				case "premium":
					rate = 1000
				default:
					rate = defaultRate
				}
			}

			res, err := limiter.Allow(r.Context(), rateLimitKey, redis_rate.PerMinute(rate))
			if err != nil {
				response.RespondWithError(w, http.StatusInternalServerError, "rate limit check failed")
				return
			}

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(res.Limit.Rate))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(res.Remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(res.ResetAfter.Milliseconds(), 10))

			if res.Allowed == 0 {
				w.Header().Set("Retry-After", strconv.FormatInt(res.RetryAfter.Milliseconds(), 10))
				response.RespondWithError(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
