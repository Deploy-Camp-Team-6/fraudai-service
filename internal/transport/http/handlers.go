package http

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	chi "github.com/go-chi/chi/v5"
	validator "github.com/go-playground/validator/v10"
	"github.com/jules-labs/go-api-prod-template/internal/db"
	"github.com/jules-labs/go-api-prod-template/internal/repo"
	"github.com/jules-labs/go-api-prod-template/internal/service"
	app_middleware "github.com/jules-labs/go-api-prod-template/internal/transport/http/middleware"
	"github.com/jules-labs/go-api-prod-template/internal/transport/http/response"
	redis "github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/sqlc-dev/pqtype"
)

var validate = validator.New()

func validationErrorMessage(err error) string {
	if errs, ok := err.(validator.ValidationErrors); ok {
		e := errs[0]
		ns := strings.Split(e.StructNamespace(), ".")[1:]
		t := reflect.TypeOf(predictRequest{})
		fieldName := ""
		for i, name := range ns {
			f, ok := t.FieldByName(name)
			if !ok {
				fieldName = strings.ToLower(name)
				break
			}
			tag := f.Tag.Get("json")
			tagName := strings.Split(tag, ",")[0]
			if i == len(ns)-1 {
				if tagName != "" {
					fieldName = tagName
				} else {
					fieldName = strings.ToLower(name)
				}
			} else {
				t = f.Type
			}
		}
		switch e.Tag() {
		case "required":
			return fieldName + " is required"
		case "oneof":
			return fieldName + " must be one of [" + e.Param() + "]"
		default:
			return "invalid " + fieldName
		}
	}
	return "invalid request"
}

const maxBodyBytes = 1 << 20 // 1MB

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

type apiKeyResponse struct {
	ID         int64      `json:"id"`
	Label      string     `json:"label"`
	Key        string     `json:"key"`
	Active     bool       `json:"active"`
	RateRPM    int32      `json:"rate_rpm"`
	LastUsedAt *time.Time `json:"last_used_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

func maskAPIKey(key string) string {
	const (
		prefixLen = 10
		suffixLen = 6
	)

	if len(key) <= prefixLen+suffixLen {
		return key
	}

	maskedLen := len(key) - prefixLen - suffixLen
	return key[:prefixLen] + strings.Repeat("*", maskedLen) + key[len(key)-suffixLen:]
}

func maskSensitiveData(data map[string]interface{}) map[string]interface{} {
	masked := make(map[string]interface{}, len(data))
	for k, v := range data {
		lower := strings.ToLower(k)
		if strings.Contains(lower, "name") || strings.Contains(lower, "email") || strings.Contains(lower, "phone") || strings.Contains(lower, "password") {
			masked[k] = "[REDACTED]"
		} else {
			masked[k] = v
		}
	}
	return masked
}

func APIKeyHandler(apiKeySvc service.APIKeyService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		identity, ok := app_middleware.IdentityFrom(r.Context())
		if !ok {
			response.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		var req apiKeyRequest
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
		if err := decoder.Decode(&req); err != nil {
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				response.RespondWithError(w, http.StatusRequestEntityTooLarge, "payload too large")
				return
			}
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

		resp := make([]apiKeyResponse, len(keys))
		for i, k := range keys {
			label := ""
			if k.Label.Valid {
				label = k.Label.String
			}

			var lastUsed *time.Time
			if k.LastUsedAt.Valid {
				t := k.LastUsedAt.Time
				lastUsed = &t
			}

			keyStr := hex.EncodeToString(k.KeyHash)
			resp[i] = apiKeyResponse{
				ID:         k.ID,
				Label:      label,
				Key:        maskAPIKey(keyStr),
				Active:     k.Active,
				RateRPM:    k.RateRpm,
				LastUsedAt: lastUsed,
				CreatedAt:  k.CreatedAt,
			}
		}

		response.RespondWithJSON(w, http.StatusOK, resp)
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
		_, ok := app_middleware.IdentityFrom(r.Context())
		if !ok {
			response.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		pong, err := vendorSvc.Ping(r.Context())
		if err != nil {
			response.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		response.RespondWithJSON(w, http.StatusOK, map[string]string{"message": pong})
	}
}

func ListModelsHandler(vendorSvc service.VendorService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, ok := app_middleware.IdentityFrom(r.Context())
		if !ok {
			response.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		models, err := vendorSvc.ListModels(r.Context())
		if err != nil {
			response.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		response.RespondWithJSON(w, http.StatusOK, map[string]interface{}{"models": models})
	}
}

func saveInferenceLog(ctx context.Context, logRepo repo.InferenceLogRepository, identity app_middleware.Identity, reqPayload, respPayload []byte, errMsg string, reqTime, respTime time.Time, logger zerolog.Logger) {
	var apiKeyID sql.NullInt64
	if identity.APIKeyID != nil {
		apiKeyID = sql.NullInt64{Int64: *identity.APIKeyID, Valid: true}
	}

	var respRaw pqtype.NullRawMessage
	if respPayload != nil {
		respRaw = pqtype.NullRawMessage{RawMessage: respPayload, Valid: true}
	}

	var errStr sql.NullString
	if errMsg != "" {
		errStr = sql.NullString{String: errMsg, Valid: true}
	}

	params := db.CreateInferenceLogParams{
		UserID:          identity.UserID,
		ApiKeyID:        apiKeyID,
		RequestPayload:  reqPayload,
		ResponsePayload: respRaw,
		Error:           errStr,
		RequestTime:     reqTime,
		ResponseTime:    respTime,
	}

	if err := logRepo.CreateInferenceLog(ctx, params); err != nil {
		logger.Error().Err(err).Msg("failed to log inference")
	}
}

type predictFeatures struct {
	TransactionID int64   `json:"transaction_id" validate:"required"`
	Amount        float64 `json:"amount" validate:"required"`
	MerchantType  string  `json:"merchant_type" validate:"required"`
	DeviceType    string  `json:"device_type" validate:"required"`
}

type predictRequest struct {
	Model    string          `json:"model" validate:"required,oneof=logreg lightgbm xgboost"`
	Features predictFeatures `json:"features" validate:"required"`
}

func PredictHandler(vendorSvc service.VendorService, logRepo repo.InferenceLogRepository, logger zerolog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		identity, ok := app_middleware.IdentityFrom(r.Context())
		if !ok {
			response.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		reqTime := time.Now()

		bodyReader := http.MaxBytesReader(w, r.Body, maxBodyBytes)
		bodyBytes, err := io.ReadAll(bodyReader)
		if err != nil {
			respTime := time.Now()
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				saveInferenceLog(r.Context(), logRepo, identity, nil, nil, "payload too large", reqTime, respTime, logger)
				response.RespondWithError(w, http.StatusRequestEntityTooLarge, "payload too large")
				return
			}
			saveInferenceLog(r.Context(), logRepo, identity, bodyBytes, nil, "invalid request body", reqTime, respTime, logger)
			response.RespondWithError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		var req predictRequest
		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			respTime := time.Now()
			saveInferenceLog(r.Context(), logRepo, identity, bodyBytes, nil, "invalid request body", reqTime, respTime, logger)
			response.RespondWithError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if err := validate.Struct(req); err != nil {
			respTime := time.Now()
			featuresMap := map[string]interface{}{
				"transaction_id": req.Features.TransactionID,
				"amount":         req.Features.Amount,
				"merchant_type":  req.Features.MerchantType,
				"device_type":    req.Features.DeviceType,
			}
			sanitizedReqBytes, _ := json.Marshal(service.PredictRequest{Model: req.Model, Features: maskSensitiveData(featuresMap)})
			msg := validationErrorMessage(err)
			saveInferenceLog(r.Context(), logRepo, identity, sanitizedReqBytes, nil, "validation failed: "+msg, reqTime, respTime, logger)
			response.RespondWithError(w, http.StatusBadRequest, "validation failed: "+msg)
			return
		}

		featuresMap := map[string]interface{}{
			"transaction_id": req.Features.TransactionID,
			"amount":         req.Features.Amount,
			"merchant_type":  req.Features.MerchantType,
			"device_type":    req.Features.DeviceType,
		}

		serviceReq := service.PredictRequest{Model: req.Model, Features: featuresMap}

		resp, err := vendorSvc.Predict(r.Context(), serviceReq)
		respTime := time.Now()

		sanitizedReqBytes, _ := json.Marshal(service.PredictRequest{Model: serviceReq.Model, Features: maskSensitiveData(serviceReq.Features)})
		var respBytes []byte
		if err == nil {
			respBytes, _ = json.Marshal(resp)
		}

		errMsg := ""
		if err != nil {
			errMsg = err.Error()
		}
		saveInferenceLog(r.Context(), logRepo, identity, sanitizedReqBytes, respBytes, errMsg, reqTime, respTime, logger)

		if err != nil {
			response.RespondWithError(w, http.StatusBadGateway, err.Error())
			return
		}

		response.RespondWithJSON(w, http.StatusOK, resp)
	}
}
