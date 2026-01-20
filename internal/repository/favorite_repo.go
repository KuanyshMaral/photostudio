package repository

import (
	"errors"
	"photostudio/internal/domain"

	"gorm.io/gorm"
)

// FavoriteRepository определяет методы для работы с избранным
type FavoriteRepository interface {
	Add(userID, studioID int64) (*domain.Favorite, error)
	Remove(userID, studioID int64) error
	GetByUserID(userID int64, limit, offset int) ([]domain.Favorite, int64, error)
	Exists(userID, studioID int64) (bool, error)
	Count(userID int64) (int64, error)
}

// favoriteRepository реализует FavoriteRepository
type favoriteRepository struct {
	db *gorm.DB
}

// NewFavoriteRepository создаёт новый экземпляр репозитория
func NewFavoriteRepository(db *gorm.DB) FavoriteRepository {
	return &favoriteRepository{db: db}
}

// Add добавляет студию в избранное пользователя.
// Возвращает ошибку если студия уже в избранном (duplicate key).
func (r *favoriteRepository) Add(userID, studioID int64) (*domain.Favorite, error) {
	// Проверяем, не добавлена ли уже
	exists, err := r.Exists(userID, studioID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("studio already in favorites")
	}

	favorite := &domain.Favorite{
		UserID:   userID,
		StudioID: studioID,
	}

	if err := r.db.Create(favorite).Error; err != nil {
		return nil, err
	}

	// Загружаем связанную студию для ответа
	if err := r.db.Preload("Studio").First(favorite, favorite.ID).Error; err != nil {
		return nil, err
	}

	return favorite, nil
}

// Remove удаляет студию из избранного пользователя.
// Возвращает ошибку если студия не была в избранном.
func (r *favoriteRepository) Remove(userID, studioID int64) error {
	result := r.db.Where("user_id = ? AND studio_id = ?", userID, studioID).
		Delete(&domain.Favorite{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("favorite not found")
	}

	return nil
}

// GetByUserID возвращает все избранные студии пользователя с пагинацией.
// Также возвращает общее количество для построения пагинации на фронте.
func (r *favoriteRepository) GetByUserID(userID int64, limit, offset int) ([]domain.Favorite, int64, error) {
	var favorites []domain.Favorite
	var total int64

	// Сначала считаем общее количество
	if err := r.db.Model(&domain.Favorite{}).
		Where("user_id = ?", userID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Затем выбираем с пагинацией и preload студий
	query := r.db.Where("user_id = ?", userID).
		Preload("Studio").
		Order("created_at DESC") // Новые сверху

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	if err := query.Find(&favorites).Error; err != nil {
		return nil, 0, err
	}

	return favorites, total, nil
}

// Exists проверяет, есть ли студия в избранном у пользователя.
// Используется для отображения состояния кнопки ❤️ на фронте.
func (r *favoriteRepository) Exists(userID, studioID int64) (bool, error) {
	var count int64
	err := r.db.Model(&domain.Favorite{}).
		Where("user_id = ? AND studio_id = ?", userID, studioID).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// Count возвращает количество избранных студий у пользователя.
// Можно использовать для ограничения (например, max 50 студий в избранном).
func (r *favoriteRepository) Count(userID int64) (int64, error) {
	var count int64
	err := r.db.Model(&domain.Favorite{}).
		Where("user_id = ?", userID).
		Count(&count).Error

	return count, err
}
