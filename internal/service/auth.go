package service

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jules-labs/go-api-prod-template/internal/db"
	"github.com/jules-labs/go-api-prod-template/internal/repo"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type AuthService interface {
	SignUp(ctx context.Context, name, email, password string) (db.CreateUserRow, string, error)
	SignIn(ctx context.Context, email, password string) (db.GetUserByEmailRow, string, error)
}

type authService struct {
	userRepo      repo.UserRepository
	jwtSecretFile string
}

func NewAuthService(userRepo repo.UserRepository, jwtSecretFile string) AuthService {
	return &authService{
		userRepo:      userRepo,
		jwtSecretFile: jwtSecretFile,
	}
}

func (s *authService) SignUp(ctx context.Context, name, email, password string) (db.CreateUserRow, string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return db.CreateUserRow{}, "", err
	}

	params := db.CreateUserParams{
		Name:         sql.NullString{String: name, Valid: true},
		Email:        email,
		PasswordHash: sql.NullString{String: string(hashedPassword), Valid: true},
	}

	user, err := s.userRepo.CreateUser(ctx, params)
	if err != nil {
		return db.CreateUserRow{}, "", err
	}

	token, err := s.generateJWT(user.ID)
	if err != nil {
		return db.CreateUserRow{}, "", err
	}

	return user, token, nil
}

func (s *authService) SignIn(ctx context.Context, email, password string) (db.GetUserByEmailRow, string, error) {
	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return db.GetUserByEmailRow{}, "", ErrUserNotFound
		}
		return db.GetUserByEmailRow{}, "", err
	}

	if !user.PasswordHash.Valid {
		return db.GetUserByEmailRow{}, "", ErrInvalidCredentials
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash.String), []byte(password))
	if err != nil {
		return db.GetUserByEmailRow{}, "", ErrInvalidCredentials
	}

	token, err := s.generateJWT(user.ID)
	if err != nil {
		return db.GetUserByEmailRow{}, "", err
	}

	return user, token, nil
}

func (s *authService) generateJWT(userID int64) (string, error) {
	secret, err := os.ReadFile(s.jwtSecretFile)
	if err != nil {
		return "", err
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString(secret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
