package favorite

import (
	"time"
)

type Favorite struct {
	ID         int64     `json:"id" gorm:"primaryKey"`
	UserID     int64     `json:"user_id" gorm:"not null"`
	EntityType string    `json:"entity_type" gorm:"not null"`
	EntityID   int64     `json:"entity_id" gorm:"not null"`
	CreatedAt  time.Time `json:"created_at" gorm:"autoCreateTime"`
}

func (Favorite) TableName() string {
	return "favorites"
}
