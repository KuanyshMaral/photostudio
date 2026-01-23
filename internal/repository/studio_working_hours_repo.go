package repository

import (
	"errors"

	"photostudio/internal/domain"

	"gorm.io/gorm"
)

type StudioWorkingHoursRepository interface {
	GetByStudioID(studioID int64) (*domain.StudioWorkingHours, error)
	CreateOrUpdate(hours *domain.StudioWorkingHours) error
	GetHoursForStudio(studioID int64) ([]domain.WorkingHours, error)
}

type studioWorkingHoursRepository struct {
	db *gorm.DB
}

func NewStudioWorkingHoursRepository(db *gorm.DB) StudioWorkingHoursRepository {
	return &studioWorkingHoursRepository{db: db}
}

func (r *studioWorkingHoursRepository) GetByStudioID(studioID int64) (*domain.StudioWorkingHours, error) {
	var hours domain.StudioWorkingHours

	err := r.db.Where("studio_id = ?", studioID).First(&hours).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Возвращаем дефолтные часы
			defaultHours := domain.DefaultWorkingHours()

			// Создаем структуру с дефолтными часами
			return &domain.StudioWorkingHours{
				StudioID: studioID,
				Hours:    defaultHours,
			}, nil
		}
		return nil, err
	}

	return &hours, nil
}

func (r *studioWorkingHoursRepository) CreateOrUpdate(hours *domain.StudioWorkingHours) error {
	return r.db.Save(hours).Error
}

func (r *studioWorkingHoursRepository) GetHoursForStudio(studioID int64) ([]domain.WorkingHours, error) {
	// Получаем запись из базы
	var hours domain.StudioWorkingHours
	err := r.db.Where("studio_id = ?", studioID).First(&hours).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Возвращаем дефолтные часы
			return domain.DefaultWorkingHours(), nil
		}
		return nil, err
	}

	// Если Hours уже []WorkingHours (после исправления структуры), возвращаем как есть
	// Если Hours всё ещё json.RawMessage (пока не исправили структуру), десериализуем

	// Сначала попробуем вернуть как []WorkingHours (если тип правильный)
	if hours.Hours != nil {
		// Предполагаем, что после исправления структуры Hours будет []domain.WorkingHours
		return hours.Hours, nil
	}

	// Запасной вариант: если Hours пустой, возвращаем дефолтные
	return domain.DefaultWorkingHours(), nil
}
