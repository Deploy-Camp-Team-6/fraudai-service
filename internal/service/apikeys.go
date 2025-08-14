package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"

	"database/sql"

	"github.com/jules-labs/go-api-prod-template/internal/db"
	"github.com/jules-labs/go-api-prod-template/internal/repo"
)

var ErrAPIKeyLabelExists = errors.New("api key label already exists")

type APIKeyService interface {
	CreateAPIKey(ctx context.Context, userID int64, label string, rateRPM int) (string, db.CreateAPIKeyRow, error)
	ListAPIKeys(ctx context.Context, userID int64) ([]db.ListAPIKeysByUserRow, error)
	DeleteAPIKey(ctx context.Context, userID, keyID int64) error
}

type apiKeyService struct {
	apiKeyRepo repo.APIKeyRepository
}

func NewAPIKeyService(apiKeyRepo repo.APIKeyRepository) APIKeyService {
	return &apiKeyService{
		apiKeyRepo: apiKeyRepo,
	}
}

func (s *apiKeyService) CreateAPIKey(ctx context.Context, userID int64, label string, rateRPM int) (string, db.CreateAPIKeyRow, error) {
	plaintextKey, err := generateRandomKey(32)
	if err != nil {
		return "", db.CreateAPIKeyRow{}, err
	}

	hashedKey := repo.HashAPIKey(plaintextKey)

	params := db.CreateAPIKeyParams{
		UserID:  userID,
		KeyHash: hashedKey,
		Label:   sql.NullString{String: label, Valid: label != ""},
		RateRpm: int32(rateRPM),
	}

	createdKey, err := s.apiKeyRepo.CreateAPIKey(ctx, params)
	if err != nil {
		if errors.Is(err, repo.ErrAPIKeyLabelExists) {
			return "", db.CreateAPIKeyRow{}, ErrAPIKeyLabelExists
		}
		return "", db.CreateAPIKeyRow{}, err
	}

	return plaintextKey, createdKey, nil
}

func (s *apiKeyService) ListAPIKeys(ctx context.Context, userID int64) ([]db.ListAPIKeysByUserRow, error) {
	return s.apiKeyRepo.ListAPIKeysByUser(ctx, userID)
}

func (s *apiKeyService) DeleteAPIKey(ctx context.Context, userID, keyID int64) error {
	return s.apiKeyRepo.DeleteAPIKey(ctx, userID, keyID)
}

func generateRandomKey(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
