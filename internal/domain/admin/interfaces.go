package admin

import (
	"context"
	"photostudio/internal/domain/auth"
	"photostudio/internal/domain/catalog"
	"photostudio/internal/domain/profile"
	"photostudio/internal/domain/review"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRepository interface {
	GetByID(ctx context.Context, id int64) (*auth.User, error)
	Update(ctx context.Context, u *auth.User) error
	DB() *gorm.DB
}

type StudioRepository interface {
	GetByID(ctx context.Context, id int64) (*catalog.Studio, error)
	Update(ctx context.Context, studio *catalog.Studio) error
	GetAll(ctx context.Context, f catalog.StudioFilters) ([]catalog.Studio, int64, error)
	DB() *gorm.DB
}

type BookingRepository interface {
	DB() *gorm.DB
}

type ReviewRepository interface {
	DB() *gorm.DB
	GetByID(ctx context.Context, id int64) (*review.Review, error)
	Update(ctx context.Context, r *review.Review) error
	// count/find by studio/user...
}

type ProfileService interface {
	EnsureAdminProfile(ctx context.Context, userID uuid.UUID, req *profile.CreateAdminProfileRequest) (*profile.AdminProfile, error)
}

type NotificationSender interface {
	NotifyVerificationApproved(ctx context.Context, userID, studioID int64) error
	NotifyVerificationRejected(ctx context.Context, userID, studioID int64, reason string) error
}

type OwnerProfileRepository interface {
	GetByID(ctx context.Context, id int64) (*profile.OwnerProfile, error)
	GetByUserID(ctx context.Context, userID int64) (*profile.OwnerProfile, error)
	Update(ctx context.Context, profile *profile.OwnerProfile) error
	FindPendingPaginated(ctx context.Context, offset, limit int) ([]profile.PendingOwnerProfileRow, int64, error)
	UpdateVerificationStatus(ctx context.Context, userID, adminID int64, status, reason, notes string) error
} // No DB() method needed as we use it via methods
