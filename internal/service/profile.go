package service

import (
	"context"

	"github.com/jules-labs/go-api-prod-template/internal/db"
	"github.com/jules-labs/go-api-prod-template/internal/repo"
)

type ProfileService interface {
	GetUserProfile(ctx context.Context, userID int64) (db.GetUserByIDRow, error)
}

type profileService struct {
	userRepo repo.UserRepository
}

func NewProfileService(userRepo repo.UserRepository) ProfileService {
	return &profileService{
		userRepo: userRepo,
	}
}

func (s *profileService) GetUserProfile(ctx context.Context, userID int64) (db.GetUserByIDRow, error) {
	return s.userRepo.GetUserByID(ctx, userID)
}
