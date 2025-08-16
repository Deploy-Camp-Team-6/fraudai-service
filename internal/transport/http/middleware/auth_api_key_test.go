package middleware

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jules-labs/go-api-prod-template/internal/db"
	"github.com/jules-labs/go-api-prod-template/internal/repo"
)

type mockAPIKeyRepo struct {
	err error
}

func (m *mockAPIKeyRepo) GetAPIKeyByHash(ctx context.Context, keyHash []byte) (db.GetAPIKeyByHashRow, error) {
	return db.GetAPIKeyByHashRow{}, m.err
}

func (m *mockAPIKeyRepo) CreateAPIKey(ctx context.Context, arg db.CreateAPIKeyParams) (db.CreateAPIKeyRow, error) {
	return db.CreateAPIKeyRow{}, nil
}

func (m *mockAPIKeyRepo) ListAPIKeysByUser(ctx context.Context, userID int64) ([]db.ListAPIKeysByUserRow, error) {
	return nil, nil
}

func (m *mockAPIKeyRepo) DeleteAPIKey(ctx context.Context, userID, keyID int64) error {
	return nil
}

type mockUserRepo struct{}

func (m mockUserRepo) CreateUser(ctx context.Context, arg db.CreateUserParams) (db.CreateUserRow, error) {
	return db.CreateUserRow{}, nil
}

func (m mockUserRepo) ListUsersPaged(ctx context.Context, arg db.ListUsersPagedParams) ([]db.ListUsersPagedRow, error) {
	return nil, nil
}

func (m mockUserRepo) GetUserByID(ctx context.Context, id int64) (db.GetUserByIDRow, error) {
	return db.GetUserByIDRow{}, nil
}

func (m mockUserRepo) GetUserByEmail(ctx context.Context, email string) (db.GetUserByEmailRow, error) {
	return db.GetUserByEmailRow{}, nil
}

func (m mockUserRepo) GetUserByEmailForLogin(ctx context.Context, email string) (db.GetUserByEmailForLoginRow, error) {
	return db.GetUserByEmailForLoginRow{}, nil
}

var _ repo.APIKeyRepository = (*mockAPIKeyRepo)(nil)
var _ repo.UserRepository = (*mockUserRepo)(nil)

func TestAPIKeyAuth_GetAPIKeyByHashErrors(t *testing.T) {
	tests := []struct {
		name       string
		repoErr    error
		wantStatus int
	}{
		{"not found", sql.ErrNoRows, http.StatusUnauthorized},
		{"other error", errors.New("db error"), http.StatusInternalServerError},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			apiRepo := &mockAPIKeyRepo{err: tc.repoErr}
			userRepo := mockUserRepo{}

			called := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
			})

			handler := APIKeyAuth(apiRepo, userRepo)(next)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("X-API-Key", "test")
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Fatalf("expected status %d, got %d", tc.wantStatus, rr.Code)
			}
			if called {
				t.Fatalf("next handler should not be called")
			}
		})
	}
}
