package auth

import (
	"context"
	"photostudio/internal/domain"
	"time"

	"gorm.io/gorm"
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

// RefreshTokenRepositoryInterface — storage for refresh tokens
type RefreshTokenRepositoryInterface interface {
	Create(ctx context.Context, t *domain.RefreshToken) error
	GetByHash(ctx context.Context, hash string) (*domain.RefreshToken, error)
	Revoke(ctx context.Context, id int64, replacedByID *int64) error
	RevokeByUser(ctx context.Context, userID int64) error
	DeleteExpired(ctx context.Context) error
}

// BookingStats — агрегированная статистика
type BookingStats struct {
	Total     int64
	Upcoming  int64
	Completed int64
	Cancelled int64
}

// RecentBookingRow — строка для последних бронирований (с уже готовыми названиями)
type RecentBookingRow struct {
	ID         int64
	StudioName string
	RoomName   string
	StartTime  time.Time
	Status     string
}

// BookingStatsReader — интерфейс который будет реализован bookingRepo
type BookingStatsReader interface {
	GetStatsByUserID(userID int64) (*BookingStats, error)
	GetRecentByUserID(userID int64, limit int) ([]RecentBookingRow, error)
}
