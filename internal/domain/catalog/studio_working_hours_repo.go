package catalog

import (
	"errors"
	"gorm.io/gorm"
)

type StudioWorkingHoursRepository interface {
	GetByStudioID(studioID int64) (*StudioWorkingHours, error)
	CreateOrUpdate(hours *StudioWorkingHours) error
	GetHoursForStudio(studioID int64) ([]WorkingHours, error)
}

type studioWorkingHoursRepository struct {
	db *gorm.DB
}

func NewStudioWorkingHoursRepository(db *gorm.DB) StudioWorkingHoursRepository {
	return &studioWorkingHoursRepository{db: db}
}

func (r *studioWorkingHoursRepository) GetByStudioID(studioID int64) (*StudioWorkingHours, error) {
	var hours StudioWorkingHours

	err := r.db.Where("studio_id = ?", studioID).First(&hours).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Возвращаем дефолтные часы
			defaultHours := DefaultWorkingHours()

			// Создаем структуру с дефолтными часами
			return &StudioWorkingHours{
				StudioID: studioID,
				Hours:    defaultHours,
			}, nil
		}
		return nil, err
	}

	return &hours, nil
}

func (r *studioWorkingHoursRepository) CreateOrUpdate(hours *StudioWorkingHours) error {
	return r.db.Save(hours).Error
}

func (r *studioWorkingHoursRepository) GetHoursForStudio(studioID int64) ([]WorkingHours, error) {
	// Получаем запись из базы
	var hours StudioWorkingHours
	err := r.db.Where("studio_id = ?", studioID).First(&hours).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Возвращаем дефолтные часы
			return DefaultWorkingHours(), nil
		}
		return nil, err
	}

	// Если Hours уже []WorkingHours (после исправления структуры), возвращаем как есть
	// Если Hours всё ещё json.RawMessage (пока не исправили структуру), десериализуем

	// Сначала попробуем вернуть как []WorkingHours (если тип правильный)
	if hours.Hours != nil {
		// Предполагаем, что после исправления структуры Hours будет []WorkingHours
		return hours.Hours, nil
	}

	// Запасной вариант: если Hours пустой, возвращаем дефолтные
	return DefaultWorkingHours(), nil
}