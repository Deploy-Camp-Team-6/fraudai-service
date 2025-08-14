package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jules-labs/go-api-prod-template/internal/service"
	app_middleware "github.com/jules-labs/go-api-prod-template/internal/transport/http/middleware"
	"github.com/jules-labs/go-api-prod-template/internal/transport/http/response"
)

type signUpRequest struct {
	Name     string `json:"name" validate:"required,min=3,max=50"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=72"`
}

func SignUpHandler(authSvc service.AuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req signUpRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.RespondWithJSON(w, http.StatusBadRequest, map[string]string{"code": "bad_request", "message": "invalid request body"})
			return
		}

		if err := validate.Struct(req); err != nil {
			response.RespondWithJSON(w, http.StatusBadRequest, map[string]string{"code": "validation_failed", "message": "validation failed: " + err.Error()})
			return
		}

		user, token, err := authSvc.SignUp(r.Context(), req.Name, req.Email, req.Password)
		if err != nil {
			if errors.Is(err, service.ErrEmailExists) {
				response.RespondWithJSON(w, http.StatusConflict, map[string]string{"code": "email_exists", "message": "email already exists"})
				return
			}
			response.RespondWithJSON(w, http.StatusInternalServerError, map[string]string{"code": "internal_error", "message": err.Error()})
			return
		}

		response.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
			"user": map[string]interface{}{
				"id":    user.ID,
				"name":  user.Name,
				"email": user.Email,
			},
			"token": token,
		})
	}
}

type signInRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

func SignInHandler(authSvc service.AuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req signInRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.RespondWithJSON(w, http.StatusBadRequest, map[string]string{"code": "bad_request", "message": "invalid request body"})
			return
		}

		if err := validate.Struct(req); err != nil {
			response.RespondWithJSON(w, http.StatusBadRequest, map[string]string{"code": "validation_failed", "message": "validation failed: " + err.Error()})
			return
		}

		user, token, err := authSvc.SignIn(r.Context(), req.Email, req.Password)
		if err != nil {
			if errors.Is(err, service.ErrInvalidCredentials) {
				response.RespondWithJSON(w, http.StatusUnauthorized, map[string]string{"code": "invalid_credentials", "message": "invalid credentials"})
				return
			}
			response.RespondWithJSON(w, http.StatusInternalServerError, map[string]string{"code": "internal_error", "message": err.Error()})
			return
		}

		response.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
			"user": map[string]interface{}{
				"id":    user.ID,
				"name":  user.Name,
				"email": user.Email,
			},
			"token": token,
		})
	}
}

func MeHandler(profileSvc service.ProfileService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		identity, ok := app_middleware.IdentityFrom(r.Context())
		if !ok {
			response.RespondWithJSON(w, http.StatusUnauthorized, map[string]string{"code": "unauthorized", "message": "unauthorized"})
			return
		}

		user, err := profileSvc.GetUserProfile(r.Context(), identity.UserID)
		if err != nil {
			response.RespondWithJSON(w, http.StatusInternalServerError, map[string]string{"code": "internal_error", "message": err.Error()})
			return
		}

		response.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
			"id":    user.ID,
			"name":  user.Name,
			"email": user.Email,
			"plan":  user.Plan,
		})
	}
}
