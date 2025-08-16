package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/jules-labs/go-api-prod-template/internal/db"
)

// mockUserRepo implements repo.UserRepository for testing

type mockUserRepo struct {
	user db.GetUserByIDRow
	err  error
}

func (m mockUserRepo) CreateUser(ctx context.Context, arg db.CreateUserParams) (db.CreateUserRow, error) {
	return db.CreateUserRow{}, nil
}

func (m mockUserRepo) ListUsersPaged(ctx context.Context, arg db.ListUsersPagedParams) ([]db.ListUsersPagedRow, error) {
	return nil, nil
}

func (m mockUserRepo) GetUserByID(ctx context.Context, id int64) (db.GetUserByIDRow, error) {
	if m.err != nil {
		return db.GetUserByIDRow{}, m.err
	}
	return m.user, nil
}

func (m mockUserRepo) GetUserByEmail(ctx context.Context, email string) (db.GetUserByEmailRow, error) {
	return db.GetUserByEmailRow{}, nil
}

func (m mockUserRepo) GetUserByEmailForLogin(ctx context.Context, email string) (db.GetUserByEmailForLoginRow, error) {
	return db.GetUserByEmailForLoginRow{}, nil
}

func TestJWTAuth(t *testing.T) {
	secret := []byte("test-secret")
	userRepo := mockUserRepo{user: db.GetUserByIDRow{ID: 1, Plan: "basic", CreatedAt: time.Now(), UpdatedAt: time.Now()}}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"user_id": 1})
	tokenStr, err := token.SignedString(secret)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	t.Run("valid token attaches identity", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer "+tokenStr)

		handlerCalled := false
		JWTAuth(secret, userRepo)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			if id, ok := IdentityFrom(r.Context()); !ok || id.UserID != 1 {
				t.Fatalf("identity not set")
			}
			w.WriteHeader(http.StatusOK)
		})).ServeHTTP(rr, req)

		if !handlerCalled {
			t.Fatalf("handler not called")
		}
		if rr.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer invalid")

		JWTAuth(secret, userRepo)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatalf("handler should not be called")
		})).ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
		}
	})
}
