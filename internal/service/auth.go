package service

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jules-labs/go-api-prod-template/internal/db"
	"github.com/jules-labs/go-api-prod-template/internal/repo"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailExists        = errors.New("email already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type AuthService interface {
	SignUp(ctx context.Context, name, email, password string) (db.CreateUserRow, string, error)
	SignIn(ctx context.Context, email, password string) (db.GetUserByEmailForLoginRow, string, error)
}

type authService struct {
	userRepo   repo.UserRepository
	jwtSecret  []byte
	jwtTimeout time.Duration
}

func NewAuthService(userRepo repo.UserRepository, jwtSecret []byte, jwtTimeout time.Duration) AuthService {
	return &authService{
		userRepo:   userRepo,
		jwtSecret:  jwtSecret,
		jwtTimeout: jwtTimeout,
	}
}

func (s *authService) SignUp(ctx context.Context, name, email, password string) (db.CreateUserRow, string, error) {
	// Check if user already exists
	_, err := s.userRepo.GetUserByEmail(ctx, email)
	if err == nil {
		return db.CreateUserRow{}, "", ErrEmailExists
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return db.CreateUserRow{}, "", err
	}

	// Create user
	params := db.CreateUserParams{
		Name:         name,
		Email:        email,
		PasswordHash: string(hashedPassword),
		Plan:         "free",
	}
	user, err := s.userRepo.CreateUser(ctx, params)
	if err != nil {
		return db.CreateUserRow{}, "", err
	}

	// Generate token
	token, err := s.generateToken(user.ID)
	if err != nil {
		return db.CreateUserRow{}, "", err
	}

	return user, token, nil
}

func (s *authService) SignIn(ctx context.Context, email, password string) (db.GetUserByEmailForLoginRow, string, error) {
	// Get user by email
	user, err := s.userRepo.GetUserByEmailForLogin(ctx, email)
	if err != nil {
		return db.GetUserByEmailForLoginRow{}, "", ErrInvalidCredentials
	}

	// Compare password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return db.GetUserByEmailForLoginRow{}, "", ErrInvalidCredentials
	}

	// Generate token
	token, err := s.generateToken(user.ID)
	if err != nil {
		return db.GetUserByEmailForLoginRow{}, "", err
	}

	return user, token, nil
}

func (s *authService) generateToken(userID int64) (string, error) {
	claims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":     userID,
		"exp":     time.Now().Add(s.jwtTimeout).Unix(),
		"iat":     time.Now().Unix(),
		"user_id": userID,
	})

	tokenString, err := claims.SignedString(s.jwtSecret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
