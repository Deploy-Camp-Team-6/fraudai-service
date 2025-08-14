package repo

import (
	"context"

	"github.com/jules-labs/go-api-prod-template/internal/db"
)

type UserRepository interface {
	CreateUser(ctx context.Context, arg db.CreateUserParams) (db.CreateUserRow, error)
	ListUsersPaged(ctx context.Context, arg db.ListUsersPagedParams) ([]db.ListUsersPagedRow, error)
	GetUserByID(ctx context.Context, id int64) (db.GetUserByIDRow, error)
	GetUserByEmail(ctx context.Context, email string) (db.GetUserByEmailRow, error)
	GetUserByEmailForLogin(ctx context.Context, email string) (db.GetUserByEmailForLoginRow, error)
}

type postgresUserRepository struct {
	q db.Querier
}

func NewUserRepository(q db.Querier) UserRepository {
	return &postgresUserRepository{
		q: q,
	}
}

func (r *postgresUserRepository) CreateUser(ctx context.Context, arg db.CreateUserParams) (db.CreateUserRow, error) {
	return r.q.CreateUser(ctx, arg)
}

func (r *postgresUserRepository) ListUsersPaged(ctx context.Context, arg db.ListUsersPagedParams) ([]db.ListUsersPagedRow, error) {
	return r.q.ListUsersPaged(ctx, arg)
}

func (r *postgresUserRepository) GetUserByID(ctx context.Context, id int64) (db.GetUserByIDRow, error) {
	return r.q.GetUserByID(ctx, id)
}

func (r *postgresUserRepository) GetUserByEmail(ctx context.Context, email string) (db.GetUserByEmailRow, error) {
	return r.q.GetUserByEmail(ctx, email)
}

func (r *postgresUserRepository) GetUserByEmailForLogin(ctx context.Context, email string) (db.GetUserByEmailForLoginRow, error) {
	return r.q.GetUserByEmailForLogin(ctx, email)
}
