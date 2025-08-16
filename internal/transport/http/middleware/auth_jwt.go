package middleware

import (
	"context"
	"net/http"
	"strings"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/jules-labs/go-api-prod-template/internal/repo"
	"github.com/jules-labs/go-api-prod-template/internal/transport/http/response"
)

func JWTAuth(secret []byte, userRepo repo.UserRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// If identity is already present, just pass through
			if _, ok := IdentityFrom(r.Context()); ok {
				next.ServeHTTP(w, r)
				return
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				// No Authorization header, pass to next middleware
				next.ServeHTTP(w, r)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				// Invalid format, but let's not error out, just pass along.
				// another middleware might handle another scheme.
				next.ServeHTTP(w, r)
				return
			}
			tokenString := parts[1]

			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return secret, nil
			})

			if err != nil {
				response.RespondWithError(w, http.StatusUnauthorized, "invalid token")
				return
			}

			if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
				userID, ok := claims["user_id"].(float64)
				if !ok {
					response.RespondWithError(w, http.StatusUnauthorized, "invalid token claims")
					return
				}

				user, err := userRepo.GetUserByID(r.Context(), int64(userID))
				if err != nil {
					response.RespondWithError(w, http.StatusInternalServerError, "could not retrieve user")
					return
				}

				identity := Identity{
					UserID: user.ID,
					Plan:   user.Plan,
				}
				ctx := context.WithValue(r.Context(), ctxKeyIdentity, identity)
				next.ServeHTTP(w, r.WithContext(ctx))
			} else {
				response.RespondWithError(w, http.StatusUnauthorized, "invalid token")
			}
		})
	}
}
