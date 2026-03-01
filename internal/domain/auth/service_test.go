package auth

import (
	"context"
	"photostudio/internal/domain/profile"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Mock User Repository implementing the interface
type mockUserRepo struct {
	mock.Mock
}

func (m *mockUserRepo) Create(ctx context.Context, u *User) error {
	args := m.Called(ctx, u)
	return args.Error(0)
}

func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*User), args.Error(1)
}

func (m *mockUserRepo) GetByID(ctx context.Context, id int64) (*User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*User), args.Error(1)
}

func (m *mockUserRepo) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	args := m.Called(ctx, email)
	return args.Bool(0), args.Error(1)
}

func (m *mockUserRepo) Update(ctx context.Context, u *User) error {
	args := m.Called(ctx, u)
	return args.Error(0)
}

func (m *mockUserRepo) DB() *gorm.DB {
	return &gorm.DB{} // dummy for transaction tests if needed
}

// Mock Owner Profile Repository
type mockOwnerProfileRepo struct {
	mock.Mock
}

func (m *mockOwnerProfileRepo) GetByUserID(ctx context.Context, userID int64) (*profile.OwnerProfile, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*profile.OwnerProfile), args.Error(1)
}

func (m *mockOwnerProfileRepo) Create(ctx context.Context, p *profile.OwnerProfile) error {
	args := m.Called(ctx, p)
	return args.Error(0)
}

// Mock Profile Service
type mockProfileService struct {
	mock.Mock
}

func (m *mockProfileService) EnsureClientProfile(ctx context.Context, userID int64) (*profile.ClientProfile, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*profile.ClientProfile), args.Error(1)
}

func (m *mockProfileService) EnsureOwnerProfile(ctx context.Context, userID int64, req *profile.CreateOwnerProfileRequest) (*profile.OwnerProfile, error) {
	args := m.Called(ctx, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*profile.OwnerProfile), args.Error(1)
}

func (m *mockProfileService) EnsureAdminProfile(ctx context.Context, userID uuid.UUID, req *profile.CreateAdminProfileRequest) (*profile.AdminProfile, error) {
	args := m.Called(ctx, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*profile.AdminProfile), args.Error(1)
}

func (m *mockProfileService) GetClientProfile(ctx context.Context, userID int64) (*profile.ClientProfile, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*profile.ClientProfile), args.Error(1)
}

func (m *mockProfileService) GetOwnerProfile(ctx context.Context, userID int64) (*profile.OwnerProfile, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*profile.OwnerProfile), args.Error(1)
}

func (m *mockProfileService) GetAdminProfile(ctx context.Context, userID uuid.UUID) (*profile.AdminProfile, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*profile.AdminProfile), args.Error(1)
}

func (m *mockProfileService) UpdateClientProfile(ctx context.Context, userID int64, req *profile.UpdateClientProfileRequest) (*profile.ClientProfile, error) {
	args := m.Called(ctx, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*profile.ClientProfile), args.Error(1)
}

func (m *mockProfileService) UpdateOwnerProfile(ctx context.Context, userID int64, req *profile.UpdateOwnerProfileRequest) (*profile.OwnerProfile, error) {
	args := m.Called(ctx, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*profile.OwnerProfile), args.Error(1)
}

func (m *mockProfileService) UpdateAdminProfile(ctx context.Context, userID uuid.UUID, req *profile.UpdateAdminProfileRequest) (*profile.AdminProfile, error) {
	args := m.Called(ctx, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*profile.AdminProfile), args.Error(1)
}

func (m *mockOwnerProfileRepo) Update(ctx context.Context, p *profile.OwnerProfile) error {
	args := m.Called(ctx, p)
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
	ownerProfileRepo := new(mockOwnerProfileRepo)
	jwtSvc := new(mockJWTService)

	// Setup expectations
	userRepo.On("ExistsByEmail", mock.Anything, "test@example.com").Return(false, nil)
	userRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	// Return ErrRecordNotFound so RequestEmailVerification exits safely without hitting the unmocked GORM DB
	userRepo.On("GetByEmail", mock.Anything, "test@example.com").Return((*User)(nil), gorm.ErrRecordNotFound)
	jwtSvc.On("GenerateToken", mock.Anything, "client").Return("fake-jwt-token", nil)

	// Create a mock profile service
	profileSvc := new(mockProfileService)
	profileSvc.On("EnsureClientProfile", mock.Anything, mock.Anything).Return(nil, nil)

	service := NewService(userRepo, ownerProfileRepo, profileSvc, jwtSvc, NewDevConsoleMailer(false), "pepper", time.Minute*5, time.Minute, "refresh_pepper", time.Hour*24)

	user, verificationSent, err := service.RegisterClient(context.Background(), RegisterClientRequest{
		Email:    "test@example.com",
		Password: "securepass123",
	})

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.True(t, verificationSent)

	userRepo.AssertExpectations(t)
}

func TestService_RegisterClient_EmailExists(t *testing.T) {
	userRepo := new(mockUserRepo)
	ownerProfileRepo := new(mockOwnerProfileRepo)
	jwtSvc := new(mockJWTService)

	userRepo.On("ExistsByEmail", mock.Anything, "exists@example.com").Return(true, nil)

	service := NewService(userRepo, ownerProfileRepo, nil, jwtSvc, NewDevConsoleMailer(false), "pepper", time.Minute*5, time.Minute, "refresh_pepper", time.Hour*24)

	_, _, err := service.RegisterClient(context.Background(), RegisterClientRequest{
		Email: "exists@example.com",
	})

	assert.ErrorIs(t, err, ErrEmailAlreadyExists)
}

func TestService_Login_Success(t *testing.T) {
	userRepo := new(mockUserRepo)
	ownerProfileRepo := new(mockOwnerProfileRepo)
	jwtSvc := new(mockJWTService)

	hashed, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	existingUser := &User{
		ID:            10,
		Email:         "user@example.com",
		PasswordHash:  string(hashed),
		Role:          RoleClient,
		EmailVerified: true,
	}

	userRepo.On("GetByEmail", mock.Anything, "user@example.com").Return(existingUser, nil)
	jwtSvc.On("GenerateToken", int64(10), "client").Return("login-token", nil)

	profileSvc := new(mockProfileService)
	profileSvc.On("EnsureClientProfile", mock.Anything, int64(10)).Return(nil, nil)

	service := NewService(userRepo, ownerProfileRepo, profileSvc, jwtSvc, NewDevConsoleMailer(false), "pepper", time.Minute*5, time.Minute, "refresh_pepper", time.Hour*24)

	res, err := service.Login(context.Background(), LoginRequest{
		Email:    "user@example.com",
		Password: "password123",
	}, "", "")

	assert.NoError(t, err)
	assert.Equal(t, "login-token", res.AccessToken)
}

func TestService_Login_WrongPassword(t *testing.T) {
	userRepo := new(mockUserRepo)
	ownerProfileRepo := new(mockOwnerProfileRepo)
	jwtSvc := new(mockJWTService)

	hashed, _ := bcrypt.GenerateFromPassword([]byte("correct"), bcrypt.DefaultCost)
	user := &User{PasswordHash: string(hashed), EmailVerified: true}

	userRepo.On("GetByEmail", mock.Anything, mock.Anything).Return(user, nil)
	userRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

	service := NewService(userRepo, ownerProfileRepo, nil, jwtSvc, NewDevConsoleMailer(false), "pepper", time.Minute*5, time.Minute, "refresh_pepper", time.Hour*24)

	_, err := service.Login(context.Background(), LoginRequest{Password: "wrong"}, "", "")

	assert.ErrorIs(t, err, ErrInvalidCredentials)
}
