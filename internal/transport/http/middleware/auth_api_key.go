package middleware

import (
	"context"
	"net/http"

	"github.com/jules-labs/go-api-prod-template/internal/repo"
	"github.com/jules-labs/go-api-prod-template/internal/transport/http/response"
)

func APIKeyAuth(apiKeyRepo repo.APIKeyRepository, userRepo repo.UserRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				// No API key provided, pass to the next middleware (e.g., JWT auth)
				next.ServeHTTP(w, r)
				return
			}

			hashedKey := repo.HashAPIKey(apiKey)
			apiKeyData, err := apiKeyRepo.GetAPIKeyByHash(r.Context(), hashedKey)
			if err != nil {
				response.RespondWithError(w, http.StatusUnauthorized, "invalid API key")
				return
			}

			if !apiKeyData.Active {
				response.RespondWithError(w, http.StatusUnauthorized, "API key is not active")
				return
			}

			user, err := userRepo.GetUserByID(r.Context(), apiKeyData.UserID)
			if err != nil {
				response.RespondWithError(w, http.StatusInternalServerError, "could not retrieve user")
				return
			}

			rate := int(apiKeyData.RateRpm)
			identity := Identity{
				UserID:   user.ID,
				Plan:     user.Plan,
				APIKeyID: &apiKeyData.ID,
				RateRPM:  &rate,
			}

			ctx := context.WithValue(r.Context(), ctxKeyIdentity, identity)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
