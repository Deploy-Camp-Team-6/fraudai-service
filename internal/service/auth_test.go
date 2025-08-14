package service

import (
	"context"
	"testing"
	"time"

	"github.com/jules-labs/go-api-prod-template/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

type mockUserRepository struct {
	mock.Mock
}

func (m *mockUserRepository) CreateUser(ctx context.Context, arg db.CreateUserParams) (db.CreateUserRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(db.CreateUserRow), args.Error(1)
}

func (m *mockUserRepository) ListUsersPaged(ctx context.Context, arg db.ListUsersPagedParams) ([]db.ListUsersPagedRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]db.ListUsersPagedRow), args.Error(1)
}

func (m *mockUserRepository) GetUserByID(ctx context.Context, id int64) (db.GetUserByIDRow, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(db.GetUserByIDRow), args.Error(1)
}

func (m *mockUserRepository) GetUserByEmail(ctx context.Context, email string) (db.GetUserByEmailRow, error) {
	args := m.Called(ctx, email)
	return args.Get(0).(db.GetUserByEmailRow), args.Error(1)
}

func (m *mockUserRepository) GetUserByEmailForLogin(ctx context.Context, email string) (db.GetUserByEmailForLoginRow, error) {
	args := m.Called(ctx, email)
	return args.Get(0).(db.GetUserByEmailForLoginRow), args.Error(1)
}

func TestAuthService_SignUp(t *testing.T) {
	mockUserRepo := new(mockUserRepository)
	authService := NewAuthService(mockUserRepo, []byte("secret"), time.Hour)

	t.Run("success", func(t *testing.T) {
		mockUserRepo.On("GetUserByEmail", mock.Anything, "test@example.com").Return(db.GetUserByEmailRow{}, assert.AnError).Once()
		mockUserRepo.On("CreateUser", mock.Anything, mock.Anything).Return(db.CreateUserRow{ID: 1, Name: "Test User", Email: "test@example.com"}, nil).Once()

		user, token, err := authService.SignUp(context.Background(), "Test User", "test@example.com", "password")

		assert.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.Equal(t, int64(1), user.ID)
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("email exists", func(t *testing.T) {
		mockUserRepo.On("GetUserByEmail", mock.Anything, "test@example.com").Return(db.GetUserByEmailRow{}, nil).Once()

		_, _, err := authService.SignUp(context.Background(), "Test User", "test@example.com", "password")

		assert.ErrorIs(t, err, ErrEmailExists)
		mockUserRepo.AssertExpectations(t)
	})
}

func TestAuthService_SignIn(t *testing.T) {
	mockUserRepo := new(mockUserRepository)
	authService := NewAuthService(mockUserRepo, []byte("secret"), time.Hour)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)

	t.Run("success", func(t *testing.T) {
		mockUserRepo.On("GetUserByEmailForLogin", mock.Anything, "test@example.com").Return(db.GetUserByEmailForLoginRow{
			ID:           1,
			PasswordHash: string(hashedPassword),
		}, nil).Once()

		_, token, err := authService.SignIn(context.Background(), "test@example.com", "password")

		assert.NoError(t, err)
		assert.NotEmpty(t, token)
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("invalid credentials", func(t *testing.T) {
		mockUserRepo.On("GetUserByEmailForLogin", mock.Anything, "test@example.com").Return(db.GetUserByEmailForLoginRow{}, assert.AnError).Once()

		_, _, err := authService.SignIn(context.Background(), "test@example.com", "password")

		assert.ErrorIs(t, err, ErrInvalidCredentials)
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("wrong password", func(t *testing.T) {
		mockUserRepo.On("GetUserByEmailForLogin", mock.Anything, "test@example.com").Return(db.GetUserByEmailForLoginRow{
			ID:           1,
			PasswordHash: string(hashedPassword),
		}, nil).Once()

		_, _, err := authService.SignIn(context.Background(), "test@example.com", "wrongpassword")

		assert.ErrorIs(t, err, ErrInvalidCredentials)
		mockUserRepo.AssertExpectations(t)
	})
}
