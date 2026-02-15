package admin

import (
	"time"

	"github.com/google/uuid"
)

// AdminUser represents an administrator in the system
type AdminUser struct {
	ID           uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Email        string     `json:"email" gorm:"unique;not null"`
	PasswordHash string     `json:"-" gorm:"not null"`
	Role         string     `json:"role" gorm:"default:'support'"` // super_admin, support, moderator
	Name         string     `json:"name" gorm:"not null"`
	AvatarURL    string     `json:"avatar_url"`
	IsActive     bool       `json:"is_active" gorm:"default:true"`
	LastLoginAt  *time.Time `json:"last_login_at"`
	LastLoginIP  string     `json:"last_login_ip"`
	CreatedAt    time.Time  `json:"created_at" gorm:"default:now()"`
	UpdatedAt    time.Time  `json:"updated_at" gorm:"default:now()"`
}

func (AdminUser) TableName() string {
	return "admin_users"
}
