package auth

import (
	"context"
	"gorm.io/gorm"
	"photostudio/internal/domain"
)

// UserRepositoryInterface — only the methods auth service uses
type UserRepositoryInterface interface {
	Create(ctx context.Context, u *domain.User) error
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByID(ctx context.Context, id int64) (*domain.User, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	Update(ctx context.Context, u *domain.User) error
	DB() *gorm.DB // changed to *gorm.DB for transaction
}

// StudioOwnerRepositoryInterface — only append docs for now
type StudioOwnerRepositoryInterface interface {
	AppendVerificationDocs(ctx context.Context, userID int64, urls []string) error
}
