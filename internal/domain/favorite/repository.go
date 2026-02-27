package favorite

import (
	"errors"

	"gorm.io/gorm"
)

type FavoriteRepository interface {
	Add(userID int64, entityType string, entityID int64) (*Favorite, error)
	Remove(userID int64, entityType string, entityID int64) error
	GetByUserID(userID int64, entityType *string, limit, offset int) ([]Favorite, int64, error)
	Exists(userID int64, entityType string, entityID int64) (bool, error)
	Count(userID int64) (int64, error)
}

type favoriteRepository struct {
	db *gorm.DB
}

func NewFavoriteRepository(db *gorm.DB) FavoriteRepository {
	return &favoriteRepository{db: db}
}

func (r *favoriteRepository) Add(userID int64, entityType string, entityID int64) (*Favorite, error) {
	exists, err := r.Exists(userID, entityType, entityID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("entity already in favorites")
	}

	favorite := &Favorite{
		UserID:     userID,
		EntityType: entityType,
		EntityID:   entityID,
	}

	if err := r.db.Create(favorite).Error; err != nil {
		return nil, err
	}

	return favorite, nil
}

func (r *favoriteRepository) Remove(userID int64, entityType string, entityID int64) error {
	result := r.db.Where("user_id = ? AND entity_type = ? AND entity_id = ?", userID, entityType, entityID).
		Delete(&Favorite{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("favorite not found")
	}

	return nil
}

func (r *favoriteRepository) GetByUserID(userID int64, entityType *string, limit, offset int) ([]Favorite, int64, error) {
	var favorites []Favorite
	var total int64

	query := r.db.Model(&Favorite{}).Where("user_id = ?", userID)
	if entityType != nil && *entityType != "" {
		query = query.Where("entity_type = ?", *entityType)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	dataQuery := r.db.Where("user_id = ?", userID)
	if entityType != nil && *entityType != "" {
		dataQuery = dataQuery.Where("entity_type = ?", *entityType)
	}

	dataQuery = dataQuery.Order("created_at DESC")

	if limit > 0 {
		dataQuery = dataQuery.Limit(limit).Offset(offset)
	}

	if err := dataQuery.Find(&favorites).Error; err != nil {
		return nil, 0, err
	}

	return favorites, total, nil
}

func (r *favoriteRepository) Exists(userID int64, entityType string, entityID int64) (bool, error) {
	var count int64
	err := r.db.Model(&Favorite{}).
		Where("user_id = ? AND entity_type = ? AND entity_id = ?", userID, entityType, entityID).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *favoriteRepository) Count(userID int64) (int64, error) {
	var count int64
	err := r.db.Model(&Favorite{}).
		Where("user_id = ?", userID).
		Count(&count).Error

	return count, err
}
