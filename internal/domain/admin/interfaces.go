package admin

import (
	"context"
	"gorm.io/gorm"
	"photostudio/internal/domain/auth"
	"photostudio/internal/domain/catalog"
	"photostudio/internal/domain/owner"
	"photostudio/internal/domain/review"
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
	GetByID(ctx context.Context, id int64) (*review.Review, error)
	Update(ctx context.Context, r *review.Review) error
	DB() *gorm.DB
}

type NotificationSender interface {
	NotifyVerificationApproved(ctx context.Context, ownerUserID, studioID int64) error
	NotifyVerificationRejected(ctx context.Context, ownerUserID, studioID int64, reason string) error
}

type StudioOwnerRepository interface {
	FindByID(ctx context.Context, id int64) (*owner.StudioOwner, error)
	Update(ctx context.Context, owner *owner.StudioOwner) error
	FindPendingPaginated(ctx context.Context, offset, limit int) ([]owner.PendingStudioOwnerRow, int64, error)
	DB() *gorm.DB
}