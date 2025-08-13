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
	Name     string `json:"name" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

func SignUpHandler(authSvc service.AuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req signUpRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.RespondWithError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if err := validate.Struct(req); err != nil {
			response.RespondWithError(w, http.StatusBadRequest, "validation failed: "+err.Error())
			return
		}

		user, token, err := authSvc.SignUp(r.Context(), req.Name, req.Email, req.Password)
		if err != nil {
			response.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		response.RespondWithJSON(w, http.StatusCreated, map[string]interface{}{
			"user": map[string]interface{}{
				"id":    user.ID,
				"name":  user.Name.String,
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
			response.RespondWithError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if err := validate.Struct(req); err != nil {
			response.RespondWithError(w, http.StatusBadRequest, "validation failed: "+err.Error())
			return
		}

		user, token, err := authSvc.SignIn(r.Context(), req.Email, req.Password)
		if err != nil {
			if errors.Is(err, service.ErrUserNotFound) || errors.Is(err, service.ErrInvalidCredentials) {
				response.RespondWithError(w, http.StatusUnauthorized, "invalid email or password")
				return
			}
			response.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		response.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
			"user": map[string]interface{}{
				"id":    user.ID,
				"name":  user.Name.String,
				"email": user.Email,
			},
			"token": token,
		})
	}
}

func GetMeHandler(profileSvc service.ProfileService) http.HandlerFunc {
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

		response.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
			"id":    user.ID,
			"name":  user.Name.String,
			"email": user.Email,
		})
	}
}
