package auth

import (
	"context"
	_ "errors"
	"testing"

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
	jwtSvc := new(mockJWTService)

	// Setup expectations
	userRepo.On("ExistsByEmail", mock.Anything, "test@example.com").Return(false, nil)
	userRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	jwtSvc.On("GenerateToken", mock.Anything, "client").Return("fake-jwt-token", nil)

	service := NewService(userRepo, studioOwnerRepo, jwtSvc)

	user, token, err := service.RegisterClient(context.Background(), RegisterClientRequest{
		Name:     "Test User",
		Email:    "test@example.com",
		Phone:    "+77001234567",
		Password: "securepass123",
	})

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "fake-jwt-token", token)

	userRepo.AssertExpectations(t)
	jwtSvc.AssertExpectations(t)
}

func TestService_RegisterClient_EmailExists(t *testing.T) {
	userRepo := new(mockUserRepo)
	studioOwnerRepo := new(mockStudioOwnerRepo)
	jwtSvc := new(mockJWTService)

	userRepo.On("ExistsByEmail", mock.Anything, "exists@example.com").Return(true, nil)

	service := NewService(userRepo, studioOwnerRepo, jwtSvc)

	_, _, err := service.RegisterClient(context.Background(), RegisterClientRequest{
		Email: "exists@example.com",
	})

	assert.ErrorIs(t, err, ErrEmailAlreadyExists)
}

func TestService_Login_Success(t *testing.T) {
	userRepo := new(mockUserRepo)
	studioOwnerRepo := new(mockStudioOwnerRepo)
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

	service := NewService(userRepo, studioOwnerRepo, jwtSvc)

	_, token, err := service.Login(context.Background(), LoginRequest{
		Email:    "user@example.com",
		Password: "password123",
	})

	assert.NoError(t, err)
	assert.Equal(t, "login-token", token)
}

func TestService_Login_WrongPassword(t *testing.T) {
	userRepo := new(mockUserRepo)
	studioOwnerRepo := new(mockStudioOwnerRepo)
	jwtSvc := new(mockJWTService)

	hashed, _ := bcrypt.GenerateFromPassword([]byte("correct"), bcrypt.DefaultCost)
	user := &domain.User{PasswordHash: string(hashed)}

	userRepo.On("GetByEmail", mock.Anything, mock.Anything).Return(user, nil)

	service := NewService(userRepo, studioOwnerRepo, jwtSvc)

	_, _, err := service.Login(context.Background(), LoginRequest{Password: "wrong"})

	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestService_AppendVerificationDocs(t *testing.T) {
	userRepo := new(mockUserRepo)
	studioOwnerRepo := new(mockStudioOwnerRepo)
	jwtSvc := new(mockJWTService)

	urls := []string{"/static/verification/doc1.pdf"}

	studioOwnerRepo.On("AppendVerificationDocs", mock.Anything, int64(5), urls).Return(nil)

	service := NewService(userRepo, studioOwnerRepo, jwtSvc)

	err := service.AppendVerificationDocs(context.Background(), 5, urls)

	assert.NoError(t, err)
	studioOwnerRepo.AssertExpectations(t)
}
