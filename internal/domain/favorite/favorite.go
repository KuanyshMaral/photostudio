package favorite

import (
	"time"

	"photostudio/internal/domain/auth"
	"photostudio/internal/domain/catalog"
)

// Favorite представляет связь пользователя с избранной студией.
// Каждая запись означает, что пользователь добавил студию в свой список избранного.
type Favorite struct {
	ID        int64     `json:"id" gorm:"primaryKey"`
	UserID    int64     `json:"user_id" gorm:"not null;index;uniqueIndex:idx_user_studio"`
	StudioID  int64     `json:"studio_id" gorm:"not null;index;uniqueIndex:idx_user_studio"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`

	// Virtual fields для preload
	User   *auth.User      `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Studio *catalog.Studio `json:"studio,omitempty" gorm:"foreignKey:StudioID"`
}

// TableName возвращает имя таблицы в БД
func (Favorite) TableName() string {
	return "favorites"
}

// FavoriteWithStudio используется для ответа API с полной информацией о студии
type FavoriteWithStudio struct {
	ID        int64           `json:"id"`
	StudioID  int64           `json:"studio_id"`
	Studio    *catalog.Studio `json:"studio"`
	CreatedAt string          `json:"created_at"`
}
