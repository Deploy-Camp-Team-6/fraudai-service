package middleware

import (
	"net/http"

	"github.com/jules-labs/go-api-prod-template/internal/transport/http/response"
)

// Chain chains multiple authentication middleware.
// It's useful for supporting multiple auth methods.
func Chain(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i](next)
		}
		return next
	}
}

// finalAuthCheck is a middleware that should be placed at the end of an auth chain.
// It checks if an identity was attached to the context and returns a 401 if not.
func finalAuthCheck(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := IdentityFrom(r.Context()); !ok {
			response.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// AuthEither provides a convenient way to support both API Key and JWT authentication.
// It chains the two auth middleware and adds a final check.
func AuthEither(apiKeyAuth, jwtAuth func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return Chain(
		apiKeyAuth,
		jwtAuth,
		finalAuthCheck,
	)
}
