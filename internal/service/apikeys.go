package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"database/sql"
	"github.com/jules-labs/go-api-prod-template/internal/db"
	"github.com/jules-labs/go-api-prod-template/internal/repo"
)

type APIKeyService interface {
	CreateAPIKey(ctx context.Context, userID int64, label string, rateRPM int) (string, db.CreateAPIKeyRow, error)
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
		return "", db.CreateAPIKeyRow{}, err
	}

	return plaintextKey, createdKey, nil
}

func generateRandomKey(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
