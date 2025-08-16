package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/jules-labs/go-api-prod-template/internal/db"
	app_middleware "github.com/jules-labs/go-api-prod-template/internal/transport/http/middleware"
)

type stubAPIKeyService struct{}

func (s *stubAPIKeyService) CreateAPIKey(ctx context.Context, userID int64, label string, rateRPM int) (string, db.CreateAPIKeyRow, error) {
	return "", db.CreateAPIKeyRow{}, nil
}

func (s *stubAPIKeyService) ListAPIKeys(ctx context.Context, userID int64) ([]db.ListAPIKeysByUserRow, error) {
	return nil, nil
}

func (s *stubAPIKeyService) DeleteAPIKey(ctx context.Context, userID, keyID int64) error {
	return nil
}

type stubUserRepo struct{}

func (s *stubUserRepo) CreateUser(ctx context.Context, arg db.CreateUserParams) (db.CreateUserRow, error) {
	return db.CreateUserRow{}, nil
}

func (s *stubUserRepo) ListUsersPaged(ctx context.Context, arg db.ListUsersPagedParams) ([]db.ListUsersPagedRow, error) {
	return nil, nil
}

func (s *stubUserRepo) GetUserByID(ctx context.Context, id int64) (db.GetUserByIDRow, error) {
	return db.GetUserByIDRow{ID: id, Plan: "basic"}, nil
}

func (s *stubUserRepo) GetUserByEmail(ctx context.Context, email string) (db.GetUserByEmailRow, error) {
	return db.GetUserByEmailRow{}, nil
}

func (s *stubUserRepo) GetUserByEmailForLogin(ctx context.Context, email string) (db.GetUserByEmailForLoginRow, error) {
	return db.GetUserByEmailForLoginRow{}, nil
}

func TestMaskAPIKey(t *testing.T) {
	t.Run("mask long key", func(t *testing.T) {
		key := "7061807972fbda86d89f899bc73124dcbee53a5a31b0e526cdd157110a6a9be3"
		expected := key[:10] + strings.Repeat("*", len(key)-16) + key[len(key)-6:]
		if got := maskAPIKey(key); got != expected {
			t.Fatalf("expected %q, got %q", expected, got)
		}
	})

	t.Run("short key unchanged", func(t *testing.T) {
		key := "123456789012345" // length 15
		if got := maskAPIKey(key); got != key {
			t.Fatalf("expected %q, got %q", key, got)
		}
	})
}

func TestAPIKeyHandler_PayloadTooLarge(t *testing.T) {
	// prepare JWT secret file
	secret := []byte("test-secret")
	secretFile := t.TempDir() + "/jwt.secret"
	if err := os.WriteFile(secretFile, secret, 0600); err != nil {
		t.Fatalf("failed to write secret file: %v", err)
	}

	// create JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"user_id": 1})
	tokenStr, err := token.SignedString(secret)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	// oversized JSON payload
	payload := `{"label":"` + strings.Repeat("a", maxBodyBytes) + `"}`

	req := httptest.NewRequest(http.MethodPost, "/apikeys", strings.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	rr := httptest.NewRecorder()

	handler := app_middleware.JWTAuth(secret, &stubUserRepo{})(APIKeyHandler(&stubAPIKeyService{}))
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status %d, got %d", http.StatusRequestEntityTooLarge, rr.Code)
	}
}
