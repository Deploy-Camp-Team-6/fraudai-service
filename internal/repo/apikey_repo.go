package repo

import (
	"context"
	"crypto/sha256"

	"github.com/jules-labs/go-api-prod-template/internal/db"
)

type APIKeyRepository interface {
	GetAPIKeyByHash(ctx context.Context, keyHash []byte) (db.GetAPIKeyByHashRow, error)
	CreateAPIKey(ctx context.Context, arg db.CreateAPIKeyParams) (db.CreateAPIKeyRow, error)
	ListAPIKeysByUser(ctx context.Context, userID int64) ([]db.ListAPIKeysByUserRow, error)
	DeleteAPIKey(ctx context.Context, userID, keyID int64) error
}

type postgresAPIKeyRepository struct {
	q db.Querier
}

func NewAPIKeyRepository(q db.Querier) APIKeyRepository {
	return &postgresAPIKeyRepository{
		q: q,
	}
}

func (r *postgresAPIKeyRepository) GetAPIKeyByHash(ctx context.Context, keyHash []byte) (db.GetAPIKeyByHashRow, error) {
	return r.q.GetAPIKeyByHash(ctx, keyHash)
}

func (r *postgresAPIKeyRepository) CreateAPIKey(ctx context.Context, arg db.CreateAPIKeyParams) (db.CreateAPIKeyRow, error) {
	return r.q.CreateAPIKey(ctx, arg)
}

func (r *postgresAPIKeyRepository) ListAPIKeysByUser(ctx context.Context, userID int64) ([]db.ListAPIKeysByUserRow, error) {
	return r.q.ListAPIKeysByUser(ctx, userID)
}

func (r *postgresAPIKeyRepository) DeleteAPIKey(ctx context.Context, userID, keyID int64) error {
	params := db.DeleteAPIKeyParams{
		UserID: userID,
		ID:     keyID,
	}
	return r.q.DeleteAPIKey(ctx, params)
}

// HashAPIKey creates a SHA256 hash of an API key.
func HashAPIKey(key string) []byte {
	hash := sha256.Sum256([]byte(key))
	return hash[:]
}
