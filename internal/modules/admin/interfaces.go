package admin

import (
	"context"

	"photostudio/internal/domain"
	"photostudio/internal/repository"

	"gorm.io/gorm"
)

type UserRepository interface {
	GetByID(ctx context.Context, id int64) (*domain.User, error)
	Update(ctx context.Context, u *domain.User) error
	DB() *gorm.DB
}

type StudioRepository interface {
	GetByID(ctx context.Context, id int64) (*domain.Studio, error)
	Update(ctx context.Context, studio *domain.Studio) error
	GetAll(ctx context.Context, f repository.StudioFilters) ([]domain.Studio, int64, error)
	DB() *gorm.DB
}

type BookingRepository interface {
	DB() *gorm.DB
}

type ReviewRepository interface {
	GetByID(ctx context.Context, id int64) (*domain.Review, error)
	Update(ctx context.Context, r *domain.Review) error
	DB() *gorm.DB
}

type NotificationSender interface {
	NotifyVerificationApproved(ctx context.Context, ownerUserID, studioID int64) error
	NotifyVerificationRejected(ctx context.Context, ownerUserID, studioID int64, reason string) error
}
