package admin

import (
	"context"
	"runtime"
	"testing"

	"photostudio/internal/domain"
	"photostudio/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

/* ==================== MOCKS ==================== */

/* -------- UserRepository -------- */

type MockUserRepository struct {
	mock.Mock
	db *gorm.DB
}

func (m *MockUserRepository) DB() *gorm.DB {
	return m.db
}

func (m *MockUserRepository) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) Update(ctx context.Context, u *domain.User) error {
	args := m.Called(ctx, u)
	return args.Error(0)
}

/* -------- StudioRepository -------- */

type MockStudioRepository struct {
	mock.Mock
	db *gorm.DB
}

func (m *MockStudioRepository) DB() *gorm.DB {
	return m.db
}

func (m *MockStudioRepository) GetByID(ctx context.Context, id int64) (*domain.Studio, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Studio), args.Error(1)
}

/* unused methods, required by interface */

func (m *MockStudioRepository) Create(_ context.Context, _ *domain.Studio) error {
	return nil
}

func (m *MockStudioRepository) Update(_ context.Context, _ *domain.Studio) error {
	return nil
}

func (m *MockStudioRepository) GetAll(
	_ context.Context,
	_ repository.StudioFilters,
) ([]domain.Studio, int64, error) {
	return nil, 0, nil
}

func (m *MockStudioRepository) GetPending(
	_ context.Context,
	_ int,
	_ int,
) ([]domain.Studio, int, error) {
	return nil, 0, nil
}

/* -------- BookingRepository -------- */

type MockBookingRepository struct {
	mock.Mock
	db *gorm.DB
}

func (m *MockBookingRepository) DB() *gorm.DB {
	return m.db
}

func (m *MockBookingRepository) Create(_ context.Context, _ *domain.Booking) error {
	return nil
}

func (m *MockBookingRepository) GetByID(_ context.Context, _ int64) (*domain.Booking, error) {
	return nil, nil
}

func (m *MockBookingRepository) Update(_ context.Context, _ *domain.Booking) error {
	return nil
}

func (m *MockBookingRepository) Delete(_ context.Context, _ int64) error {
	return nil
}

func (m *MockBookingRepository) GetByStudioID(
	_ context.Context,
	_ int64,
	_ int,
	_ int,
) ([]domain.Booking, int64, error) {
	return nil, 0, nil
}

func (m *MockBookingRepository) GetByUserID(
	_ context.Context,
	_ int64,
	_ int,
	_ int,
) ([]domain.Booking, int64, error) {
	return nil, 0, nil
}

/* -------- ReviewRepository -------- */

type MockReviewRepository struct{}

func (m *MockReviewRepository) DB() *gorm.DB {
	return nil
}

func (m *MockReviewRepository) GetByID(_ context.Context, _ int64) (*domain.Review, error) {
	return nil, nil
}

func (m *MockReviewRepository) Update(_ context.Context, _ *domain.Review) error {
	return nil
}

/* ==================== SQLITE TEST DB ==================== */

func testDB(t *testing.T) *gorm.DB {
	if runtime.GOOS == "windows" {
		t.Skip("skipping sqlite test on windows because CGO is disabled")
	}

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}

	_ = db.AutoMigrate(
		&domain.User{},
		&domain.Studio{},
		&domain.Booking{},
		&domain.Review{},
	)

	return db
}

/* ==================== TESTS (TASK 3.7) ==================== */

func TestVerifyStudio_Success(t *testing.T) {
	ctx := context.Background()

	studio := &domain.Studio{
		ID:      1,
		OwnerID: 10,
	}

	owner := &domain.User{
		ID:           10,
		StudioStatus: domain.StatusPending,
	}

	userRepo := new(MockUserRepository)
	studioRepo := new(MockStudioRepository)

	studioRepo.On("GetByID", ctx, int64(1)).Return(studio, nil)
	userRepo.On("GetByID", ctx, int64(10)).Return(owner, nil)
	userRepo.On("Update", ctx, mock.MatchedBy(func(u *domain.User) bool {
		return u.StudioStatus == domain.StatusVerified
	})).Return(nil)

	service := NewService(
		userRepo,
		studioRepo,
		&MockBookingRepository{},
		&MockReviewRepository{},
	)

	res, err := service.VerifyStudio(ctx, 1, 100, "OK")

	assert.NoError(t, err)
	assert.Equal(t, studio, res)
	userRepo.AssertExpectations(t)
	studioRepo.AssertExpectations(t)
}

func TestVerifyStudio_NotFound(t *testing.T) {
	ctx := context.Background()

	studioRepo := new(MockStudioRepository)
	studioRepo.On("GetByID", ctx, int64(999)).
		Return(nil, gorm.ErrRecordNotFound)

	service := NewService(
		new(MockUserRepository),
		studioRepo,
		&MockBookingRepository{},
		&MockReviewRepository{},
	)

	res, err := service.VerifyStudio(ctx, 999, 1, "OK")

	assert.Nil(t, res)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestRejectStudio_Success(t *testing.T) {
	ctx := context.Background()

	studio := &domain.Studio{
		ID:      1,
		OwnerID: 10,
	}

	owner := &domain.User{
		ID:           10,
		StudioStatus: domain.StatusPending,
	}

	userRepo := new(MockUserRepository)
	studioRepo := new(MockStudioRepository)

	studioRepo.On("GetByID", ctx, int64(1)).Return(studio, nil)
	userRepo.On("GetByID", ctx, int64(10)).Return(owner, nil)
	userRepo.On("Update", ctx, mock.MatchedBy(func(u *domain.User) bool {
		return u.StudioStatus == domain.StatusRejected
	})).Return(nil)

	service := NewService(
		userRepo,
		studioRepo,
		&MockBookingRepository{},
		&MockReviewRepository{},
	)

	res, err := service.RejectStudio(ctx, 1, 1, "Invalid docs")

	assert.NoError(t, err)
	assert.Equal(t, studio, res)
	userRepo.AssertExpectations(t)
	studioRepo.AssertExpectations(t)
}

func TestGetStatistics_Success(t *testing.T) {
	ctx := context.Background()
	db := testDB(t)

	mockUser := &MockUserRepository{db: db}
	mockStudio := &MockStudioRepository{db: db}
	mockBooking := &MockBookingRepository{db: db}
	mockReview := new(MockReviewRepository)

	service := NewService(mockUser, mockStudio, mockBooking, mockReview)

	stats, err := service.GetStatistics(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, stats)
}
