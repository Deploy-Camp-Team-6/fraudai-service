package http

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	chi "github.com/go-chi/chi/v5"
	validator "github.com/go-playground/validator/v10"
	"github.com/jules-labs/go-api-prod-template/internal/service"
	app_middleware "github.com/jules-labs/go-api-prod-template/internal/transport/http/middleware"
	"github.com/jules-labs/go-api-prod-template/internal/transport/http/response"
	redis "github.com/redis/go-redis/v9"
)

var validate = validator.New()

func HealthzHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func ReadinessHandler(db *sql.DB, redisClient *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check DB
		if err := db.PingContext(r.Context()); err != nil {
			response.RespondWithError(w, http.StatusServiceUnavailable, "database not ready")
			return
		}

		// Check Redis
		if _, err := redisClient.Ping(r.Context()).Result(); err != nil {
			response.RespondWithError(w, http.StatusServiceUnavailable, "redis not ready")
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}
}

func ProfileHandler(profileSvc service.ProfileService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		identity, ok := app_middleware.IdentityFrom(r.Context())
		if !ok {
			response.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		user, err := profileSvc.GetUserProfile(r.Context(), identity.UserID)
		if err != nil {
			response.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		response.RespondWithJSON(w, http.StatusOK, user)
	}
}

type apiKeyRequest struct {
	Label   string `json:"label" validate:"required,min=3,max=50"`
	RateRPM int    `json:"rate_rpm" validate:"omitempty,min=1,max=10000"`
}

func APIKeyHandler(apiKeySvc service.APIKeyService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		identity, ok := app_middleware.IdentityFrom(r.Context())
		if !ok {
			response.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		var req apiKeyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.RespondWithError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if err := validate.Struct(req); err != nil {
			response.RespondWithError(w, http.StatusBadRequest, "validation failed: "+err.Error())
			return
		}

		rateRPM := 100 // default
		if req.RateRPM > 0 {
			rateRPM = req.RateRPM
		}

		plaintextKey, createdKey, err := apiKeySvc.CreateAPIKey(r.Context(), identity.UserID, req.Label, rateRPM)
		if err != nil {
			if errors.Is(err, service.ErrAPIKeyLabelExists) {
				response.RespondWithError(w, http.StatusConflict, "label already exists")
				return
			}
			response.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		response.RespondWithJSON(w, http.StatusCreated, map[string]interface{}{
			"key":     plaintextKey,
			"details": createdKey,
		})
	}
}

func ListAPIKeysHandler(apiKeySvc service.APIKeyService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		identity, ok := app_middleware.IdentityFrom(r.Context())
		if !ok {
			response.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		keys, err := apiKeySvc.ListAPIKeys(r.Context(), identity.UserID)
		if err != nil {
			response.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		response.RespondWithJSON(w, http.StatusOK, keys)
	}
}

func DeleteAPIKeyHandler(apiKeySvc service.APIKeyService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		identity, ok := app_middleware.IdentityFrom(r.Context())
		if !ok {
			response.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		idParam := chi.URLParam(r, "id")
		keyID, err := strconv.ParseInt(idParam, 10, 64)
		if err != nil {
			response.RespondWithError(w, http.StatusBadRequest, "invalid key id")
			return
		}

		if err := apiKeySvc.DeleteAPIKey(r.Context(), identity.UserID, keyID); err != nil {
			response.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func VendorPingHandler(vendorSvc service.VendorService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pong, err := vendorSvc.Ping(r.Context())
		if err != nil {
			response.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		response.RespondWithJSON(w, http.StatusOK, map[string]string{"message": pong})
	}
}
