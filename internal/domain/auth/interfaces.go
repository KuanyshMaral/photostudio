package auth

import (
	"context"
	"photostudio/internal/domain/profile"
	"time"

	"gorm.io/gorm"
)

// UserRepositoryInterface — only the methods auth service uses
type UserRepositoryInterface interface {
	Create(ctx context.Context, u *User) error
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id int64) (*User, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	Update(ctx context.Context, u *User) error
	DB() *gorm.DB // changed to *gorm.DB for transaction
}

// OwnerProfileRepositoryInterface - interface for profile interactions
type OwnerProfileRepositoryInterface interface {
	GetByUserID(ctx context.Context, userID int64) (*profile.OwnerProfile, error)
	Update(ctx context.Context, profile *profile.OwnerProfile) error
	Create(ctx context.Context, profile *profile.OwnerProfile) error // For completeness if needed
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
