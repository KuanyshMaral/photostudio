package admin

import (
	"context"
	"photostudio/internal/domain/auth"
	"photostudio/internal/domain/catalog"
	"photostudio/internal/domain/owner"
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

type StudioOwnerRepository interface {
	FindByID(ctx context.Context, id int64) (*owner.StudioOwner, error)
	Update(ctx context.Context, owner *owner.StudioOwner) error
	FindPendingPaginated(ctx context.Context, offset, limit int) ([]owner.PendingStudioOwnerRow, int64, error)
	DB() *gorm.DB
}
