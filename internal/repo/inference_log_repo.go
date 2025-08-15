package repo

import (
	"context"

	"github.com/jules-labs/go-api-prod-template/internal/db"
)

type InferenceLogRepository interface {
	CreateInferenceLog(ctx context.Context, arg db.CreateInferenceLogParams) error
}

type postgresInferenceLogRepository struct {
	q db.Querier
}

func NewInferenceLogRepository(q db.Querier) InferenceLogRepository {
	return &postgresInferenceLogRepository{q: q}
}

func (r *postgresInferenceLogRepository) CreateInferenceLog(ctx context.Context, arg db.CreateInferenceLogParams) error {
	return r.q.CreateInferenceLog(ctx, arg)
}
