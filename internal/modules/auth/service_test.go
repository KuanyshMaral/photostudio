package auth

import (
	"context"
	_ "errors"
	_ "github.com/golang-jwt/jwt/v5"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"photostudio/internal/domain"
)

// Mock User Repository implementing the interface
type mockUserRepo struct {
	mock.Mock
}

func (m *mockUserRepo) Create(ctx context.Context, u *domain.User) error {
	args := m.Called(ctx, u)
	return args.Error(0)
}

func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *mockUserRepo) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *mockUserRepo) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	args := m.Called(ctx, email)
	return args.Bool(0), args.Error(1)
}

func (m *mockUserRepo) Update(ctx context.Context, u *domain.User) error {
	args := m.Called(ctx, u)
	return args.Error(0)
}

func (m *mockUserRepo) DB() *gorm.DB {
	return &gorm.DB{} // dummy for transaction tests if needed
}

// Mock Studio Owner Repository
type mockStudioOwnerRepo struct {
	mock.Mock
}

// Mock Refresh Token Repository
type mockRefreshTokenRepo struct {
	mock.Mock
}

func (m *mockRefreshTokenRepo) Create(ctx context.Context, t *domain.RefreshToken) error {
	args := m.Called(ctx, t)
	return args.Error(0)
}

func (m *mockRefreshTokenRepo) GetByHash(ctx context.Context, hash string) (*domain.RefreshToken, error) {
	args := m.Called(ctx, hash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.RefreshToken), args.Error(1)
}

func (m *mockRefreshTokenRepo) Revoke(ctx context.Context, id int64, replacedByID *int64) error {
	args := m.Called(ctx, id, replacedByID)
	return args.Error(0)
}

func (m *mockRefreshTokenRepo) RevokeByUser(ctx context.Context, userID int64) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *mockRefreshTokenRepo) DeleteExpired(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockStudioOwnerRepo) AppendVerificationDocs(ctx context.Context, userID int64, urls []string) error {
	args := m.Called(ctx, userID, urls)
	return args.Error(0)
}

// Mock JWT service
type mockJWTService struct {
	mock.Mock
}

func (m *mockJWTService) GenerateToken(userID int64, role string) (string, error) {
	args := m.Called(userID, role)
	return args.String(0), args.Error(1)
}

func TestService_RegisterClient_Success(t *testing.T) {
	userRepo := new(mockUserRepo)
	studioOwnerRepo := new(mockStudioOwnerRepo)
	refreshRepo := new(mockRefreshTokenRepo)
	jwtSvc := new(mockJWTService)

	// Setup expectations
	userRepo.On("ExistsByEmail", mock.Anything, "test@example.com").Return(false, nil)
	userRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	jwtSvc.On("GenerateToken", mock.Anything, "client").Return("fake-jwt-token", nil)
	refreshRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	service := NewService(userRepo, studioOwnerRepo, refreshRepo, jwtSvc, 15*time.Minute, 7*24*time.Hour)

	user, tokens, err := service.RegisterClient(context.Background(), RegisterClientRequest{
		Name:     "Test User",
		Email:    "test@example.com",
		Phone:    "+77001234567",
		Password: "securepass123",
	})

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "fake-jwt-token", tokens.AccessToken)
	assert.NotEmpty(t, tokens.RefreshToken)

	userRepo.AssertExpectations(t)
	jwtSvc.AssertExpectations(t)
	refreshRepo.AssertExpectations(t)
}

func TestService_RegisterClient_EmailExists(t *testing.T) {
	userRepo := new(mockUserRepo)
	studioOwnerRepo := new(mockStudioOwnerRepo)
	refreshRepo := new(mockRefreshTokenRepo)
	jwtSvc := new(mockJWTService)

	userRepo.On("ExistsByEmail", mock.Anything, "exists@example.com").Return(true, nil)

	service := NewService(userRepo, studioOwnerRepo, refreshRepo, jwtSvc, 15*time.Minute, 7*24*time.Hour)

	_, _, err := service.RegisterClient(context.Background(), RegisterClientRequest{
		Email: "exists@example.com",
	})

	assert.ErrorIs(t, err, ErrEmailAlreadyExists)
}

func TestService_Login_Success(t *testing.T) {
	userRepo := new(mockUserRepo)
	studioOwnerRepo := new(mockStudioOwnerRepo)
	refreshRepo := new(mockRefreshTokenRepo)
	jwtSvc := new(mockJWTService)

	hashed, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	existingUser := &domain.User{
		ID:           10,
		Email:        "user@example.com",
		PasswordHash: string(hashed),
		Role:         domain.RoleClient,
	}

	userRepo.On("GetByEmail", mock.Anything, "user@example.com").Return(existingUser, nil)
	jwtSvc.On("GenerateToken", int64(10), "client").Return("login-token", nil)
	refreshRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	service := NewService(userRepo, studioOwnerRepo, refreshRepo, jwtSvc, 15*time.Minute, 7*24*time.Hour)

	_, tokens, err := service.Login(context.Background(), LoginRequest{
		Email:    "user@example.com",
		Password: "password123",
	})

	assert.NoError(t, err)
	assert.Equal(t, "login-token", tokens.AccessToken)
	assert.NotEmpty(t, tokens.RefreshToken)
}

func TestService_AppendVerificationDocs(t *testing.T) {
	userRepo := new(mockUserRepo)
	studioOwnerRepo := new(mockStudioOwnerRepo)
	refreshRepo := new(mockRefreshTokenRepo)
	jwtSvc := new(mockJWTService)

	urls := []string{"/static/verification/doc1.pdf"}

	studioOwnerRepo.On("AppendVerificationDocs", mock.Anything, int64(5), urls).Return(nil)

	service := NewService(userRepo, studioOwnerRepo, refreshRepo, jwtSvc, 15*time.Minute, 7*24*time.Hour)

	err := service.AppendVerificationDocs(context.Background(), 5, urls)

	assert.NoError(t, err)
	studioOwnerRepo.AssertExpectations(t)
}
