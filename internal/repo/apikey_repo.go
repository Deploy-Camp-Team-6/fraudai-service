package repo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	redis "github.com/redis/go-redis/v9"

	"github.com/jules-labs/go-api-prod-template/internal/db"
)

type APIKeyRepository interface {
	GetAPIKeyByHash(ctx context.Context, keyHash []byte) (db.GetAPIKeyByHashRow, error)
	CreateAPIKey(ctx context.Context, arg db.CreateAPIKeyParams) (db.CreateAPIKeyRow, error)
	ListAPIKeysByUser(ctx context.Context, userID int64) ([]db.ListAPIKeysByUserRow, error)
	DeleteAPIKey(ctx context.Context, userID, keyID int64) error
}

type postgresAPIKeyRepository struct {
	q           db.Querier
	redisClient *redis.Client
	ttl         time.Duration
}

type cachedAPIKey struct {
	ID      int64 `json:"id"`
	UserID  int64 `json:"user_id"`
	Active  bool  `json:"active"`
	RateRpm int32 `json:"rate_rpm"`
}

func apiKeyCacheKey(hash []byte) string {
	return fmt.Sprintf("apikey:%s", hex.EncodeToString(hash))
}

func apiKeyIDCacheKey(id int64) string {
	return fmt.Sprintf("apikey_id:%d", id)
}

func NewAPIKeyRepository(q db.Querier, redisClient *redis.Client, ttl time.Duration) APIKeyRepository {
	return &postgresAPIKeyRepository{
		q:           q,
		redisClient: redisClient,
		ttl:         ttl,
	}
}

func (r *postgresAPIKeyRepository) GetAPIKeyByHash(ctx context.Context, keyHash []byte) (db.GetAPIKeyByHashRow, error) {
	if r.redisClient != nil {
		cacheKey := apiKeyCacheKey(keyHash)
		if data, err := r.redisClient.Get(ctx, cacheKey).Bytes(); err == nil {
			var c cachedAPIKey
			if json.Unmarshal(data, &c) == nil {
				return db.GetAPIKeyByHashRow{
					ID:      c.ID,
					UserID:  c.UserID,
					KeyHash: keyHash,
					Active:  c.Active,
					RateRpm: c.RateRpm,
				}, nil
			}
		}
	}

	apiKey, err := r.q.GetAPIKeyByHash(ctx, keyHash)
	if err != nil {
		return apiKey, err
	}

	if r.redisClient != nil {
		c := cachedAPIKey{
			ID:      apiKey.ID,
			UserID:  apiKey.UserID,
			Active:  apiKey.Active,
			RateRpm: apiKey.RateRpm,
		}
		if data, err := json.Marshal(c); err == nil {
			_ = r.redisClient.Set(ctx, apiKeyCacheKey(keyHash), data, r.ttl).Err()
			_ = r.redisClient.Set(ctx, apiKeyIDCacheKey(apiKey.ID), hex.EncodeToString(keyHash), r.ttl).Err()
		}
	}
	return apiKey, nil
}

func (r *postgresAPIKeyRepository) CreateAPIKey(ctx context.Context, arg db.CreateAPIKeyParams) (db.CreateAPIKeyRow, error) {
	createdKey, err := r.q.CreateAPIKey(ctx, arg)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return db.CreateAPIKeyRow{}, ErrAPIKeyLabelExists
		}
		return db.CreateAPIKeyRow{}, err
	}
	if r.redisClient != nil {
		c := cachedAPIKey{
			ID:      createdKey.ID,
			UserID:  createdKey.UserID,
			Active:  createdKey.Active,
			RateRpm: createdKey.RateRpm,
		}
		if data, err := json.Marshal(c); err == nil {
			_ = r.redisClient.Set(ctx, apiKeyCacheKey(arg.KeyHash), data, r.ttl).Err()
			_ = r.redisClient.Set(ctx, apiKeyIDCacheKey(createdKey.ID), hex.EncodeToString(arg.KeyHash), r.ttl).Err()
		}
	}
	return createdKey, nil
}

func (r *postgresAPIKeyRepository) ListAPIKeysByUser(ctx context.Context, userID int64) ([]db.ListAPIKeysByUserRow, error) {
	return r.q.ListAPIKeysByUser(ctx, userID)
}

func (r *postgresAPIKeyRepository) DeleteAPIKey(ctx context.Context, userID, keyID int64) error {
	params := db.DeleteAPIKeyParams{
		UserID: userID,
		ID:     keyID,
	}
	if err := r.q.DeleteAPIKey(ctx, params); err != nil {
		return err
	}
	if r.redisClient != nil {
		idKey := apiKeyIDCacheKey(keyID)
		if hash, err := r.redisClient.Get(ctx, idKey).Result(); err == nil {
			_ = r.redisClient.Del(ctx, idKey).Err()
			_ = r.redisClient.Del(ctx, fmt.Sprintf("apikey:%s", hash)).Err()
		}
	}
	return nil
}

// HashAPIKey creates a SHA256 hash of an API key.
func HashAPIKey(key string) []byte {
	hash := sha256.Sum256([]byte(key))
	return hash[:]
}
