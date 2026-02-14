package owner

import (
	"context"
	"errors"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"time"
)

var (
	ErrInvalidPIN   = errors.New("invalid PIN")
	ErrPINNotSet    = errors.New("PIN not set")
	ErrItemNotFound = errors.New("item not found")
)

type OwnerCRMRepository struct {
	db *gorm.DB
}

func NewOwnerCRMRepository(db *gorm.DB) *OwnerCRMRepository {
	return &OwnerCRMRepository{db: db}
}

// ==================== PIN Methods ====================

// SetPIN устанавливает или обновляет PIN владельца
func (r *OwnerCRMRepository) SetPIN(ctx context.Context, ownerID int64, pin string) error {
	// Хэшируем PIN (4-6 цифр)
	hash, err := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	ownerPIN := OwnerPIN{
		UserID:  ownerID,
		PinHash: string(hash),
	}

	// Upsert: создать или обновить
	return r.db.WithContext(ctx).
		Where("user_id = ?", ownerID).
		Assign(ownerPIN).
		FirstOrCreate(&ownerPIN).Error
}

// VerifyPIN проверяет PIN владельца
func (r *OwnerCRMRepository) VerifyPIN(ctx context.Context, ownerID int64, pin string) error {
	var ownerPIN OwnerPIN
	err := r.db.WithContext(ctx).
		Where("user_id = ?", ownerID).
		First(&ownerPIN).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrPINNotSet
	}
	if err != nil {
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(ownerPIN.PinHash), []byte(pin)); err != nil {
		return ErrInvalidPIN
	}

	return nil
}

// HasPIN проверяет, установлен ли PIN
func (r *OwnerCRMRepository) HasPIN(ctx context.Context, ownerID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&OwnerPIN{}).
		Where("user_id = ?", ownerID).
		Count(&count).Error
	return count > 0, err
}

// ==================== Procurement Methods ====================

// GetProcurementItems возвращает список закупок владельца
func (r *OwnerCRMRepository) GetProcurementItems(ctx context.Context, ownerID int64, showCompleted bool) ([]ProcurementItem, error) {
	var items []ProcurementItem
	query := r.db.WithContext(ctx).Where("owner_id = ?", ownerID)

	if !showCompleted {
		query = query.Where("is_completed = false")
	}

	err := query.Order("priority DESC, created_at DESC").Find(&items).Error
	return items, err
}

// CreateProcurementItem создаёт новую закупку
func (r *OwnerCRMRepository) CreateProcurementItem(ctx context.Context, item *ProcurementItem) error {
	return r.db.WithContext(ctx).Create(item).Error
}

// UpdateProcurementItem обновляет закупку
func (r *OwnerCRMRepository) UpdateProcurementItem(ctx context.Context, ownerID, itemID int64, updates map[string]interface{}) error {
	result := r.db.WithContext(ctx).
		Model(&ProcurementItem{}).
		Where("id = ? AND owner_id = ?", itemID, ownerID).
		Updates(updates)

	if result.RowsAffected == 0 {
		return ErrItemNotFound
	}
	return result.Error
}

// DeleteProcurementItem удаляет закупку
func (r *OwnerCRMRepository) DeleteProcurementItem(ctx context.Context, ownerID, itemID int64) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND owner_id = ?", itemID, ownerID).
		Delete(&ProcurementItem{})

	if result.RowsAffected == 0 {
		return ErrItemNotFound
	}
	return result.Error
}

// ==================== Maintenance Methods ====================

// GetMaintenanceItems возвращает список задач обслуживания
func (r *OwnerCRMRepository) GetMaintenanceItems(ctx context.Context, ownerID int64, status string) ([]MaintenanceItem, error) {
	var items []MaintenanceItem
	query := r.db.WithContext(ctx).Where("owner_id = ?", ownerID)

	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}

	err := query.Order("priority DESC, due_date ASC").Find(&items).Error
	return items, err
}

// CreateMaintenanceItem создаёт новую задачу
func (r *OwnerCRMRepository) CreateMaintenanceItem(ctx context.Context, item *MaintenanceItem) error {
	return r.db.WithContext(ctx).Create(item).Error
}

// UpdateMaintenanceItem обновляет задачу
func (r *OwnerCRMRepository) UpdateMaintenanceItem(ctx context.Context, ownerID, itemID int64, updates map[string]interface{}) error {
	result := r.db.WithContext(ctx).
		Model(&MaintenanceItem{}).
		Where("id = ? AND owner_id = ?", itemID, ownerID).
		Updates(updates)

	if result.RowsAffected == 0 {
		return ErrItemNotFound
	}
	return result.Error
}

// DeleteMaintenanceItem удаляет задачу
func (r *OwnerCRMRepository) DeleteMaintenanceItem(ctx context.Context, ownerID, itemID int64) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND owner_id = ?", itemID, ownerID).
		Delete(&MaintenanceItem{})

	if result.RowsAffected == 0 {
		return ErrItemNotFound
	}
	return result.Error
}

// ==================== Analytics Methods ====================

// OwnerAnalytics — структура для аналитики владельца
type OwnerAnalytics struct {
	TotalBookings     int64            `json:"total_bookings"`
	TotalRevenue      float64          `json:"total_revenue"`
	AvgBookingValue   float64          `json:"avg_booking_value"`
	BookingsThisMonth int64            `json:"bookings_this_month"`
	RevenueThisMonth  float64          `json:"revenue_this_month"`
	TopRooms          []RoomStats      `json:"top_rooms"`
	BookingsByStatus  map[string]int64 `json:"bookings_by_status"`
}

type RoomStats struct {
	RoomID       int64   `json:"room_id"`
	RoomName     string  `json:"room_name"`
	BookingCount int64   `json:"booking_count"`
	Revenue      float64 `json:"revenue"`
}

// GetOwnerAnalytics возвращает аналитику для владельца
func (r *OwnerCRMRepository) GetOwnerAnalytics(ctx context.Context, ownerID int64) (*OwnerAnalytics, error) {
	analytics := &OwnerAnalytics{
		BookingsByStatus: make(map[string]int64),
	}

	// 1. Получаем ID студий владельца
	var studioIDs []int64
	err := r.db.WithContext(ctx).
		Table("studios").
		Where("owner_id = ?", ownerID).
		Pluck("id", &studioIDs).Error
	if err != nil {
		return nil, err
	}

	if len(studioIDs) == 0 {
		return analytics, nil
	}

	// 2. Общая статистика
	var totalStats struct {
		Count   int64   `gorm:"column:count"`
		Revenue float64 `gorm:"column:revenue"`
	}
	r.db.WithContext(ctx).
		Table("bookings").
		Select("COUNT(*) as count, COALESCE(SUM(total_price), 0) as revenue").
		Where("studio_id IN ?", studioIDs).
		Where("status IN ('confirmed', 'completed')").
		Scan(&totalStats)

	analytics.TotalBookings = totalStats.Count
	analytics.TotalRevenue = totalStats.Revenue
	if totalStats.Count > 0 {
		analytics.AvgBookingValue = totalStats.Revenue / float64(totalStats.Count)
	}

	// 3. Статистика за этот месяц
	var monthStats struct {
		Count   int64   `gorm:"column:count"`
		Revenue float64 `gorm:"column:revenue"`
	}
	r.db.WithContext(ctx).
		Table("bookings").
		Select("COUNT(*) as count, COALESCE(SUM(total_price), 0) as revenue").
		Where("studio_id IN ?", studioIDs).
		Where("status IN ('confirmed', 'completed')").
		Where("created_at >= DATE_TRUNC('month', CURRENT_DATE)").
		Scan(&monthStats)

	analytics.BookingsThisMonth = monthStats.Count
	analytics.RevenueThisMonth = monthStats.Revenue

	// 4. Топ комнат
	var topRooms []RoomStats
	r.db.WithContext(ctx).
		Table("bookings b").
		Select("b.room_id, r.name as room_name, COUNT(*) as booking_count, COALESCE(SUM(b.total_price), 0) as revenue").
		Joins("JOIN rooms r ON r.id = b.room_id").
		Where("b.studio_id IN ?", studioIDs).
		Where("b.status IN ('confirmed', 'completed')").
		Group("b.room_id, r.name").
		Order("booking_count DESC").
		Limit(5).
		Scan(&topRooms)
	analytics.TopRooms = topRooms

	// 5. Бронирования по статусам
	var statusCounts []struct {
		Status string `gorm:"column:status"`
		Count  int64  `gorm:"column:count"`
	}
	r.db.WithContext(ctx).
		Table("bookings").
		Select("status, COUNT(*) as count").
		Where("studio_id IN ?", studioIDs).
		Group("status").
		Scan(&statusCounts)

	for _, sc := range statusCounts {
		analytics.BookingsByStatus[sc.Status] = sc.Count
	}

	return analytics, nil
}

// ==================== Company Profile Methods ====================

// GetCompanyProfile возвращает профиль компании
func (r *OwnerCRMRepository) GetCompanyProfile(ctx context.Context, ownerID int64) (*CompanyProfile, error) {
	var profile CompanyProfile
	err := r.db.WithContext(ctx).
		Where("owner_id = ?", ownerID).
		First(&profile).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Создаём пустой профиль
		profile = CompanyProfile{OwnerID: ownerID}
		if err := r.db.WithContext(ctx).Create(&profile).Error; err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	return &profile, nil
}

// UpdateCompanyProfile обновляет профиль компании
func (r *OwnerCRMRepository) UpdateCompanyProfile(ctx context.Context, ownerID int64, updates *CompanyProfile) error {
	updates.OwnerID = ownerID
	return r.db.WithContext(ctx).
		Where("owner_id = ?", ownerID).
		Updates(updates).Error
}

// ==================== Portfolio Methods ====================

// GetPortfolio возвращает проекты портфолио
func (r *OwnerCRMRepository) GetPortfolio(ctx context.Context, ownerID int64) ([]PortfolioProject, error) {
	var projects []PortfolioProject
	err := r.db.WithContext(ctx).
		Where("owner_id = ?", ownerID).
		Order("sort_order ASC, created_at DESC").
		Find(&projects).Error
	return projects, err
}

// AddPortfolioProject добавляет проект в портфолио
func (r *OwnerCRMRepository) AddPortfolioProject(ctx context.Context, project *PortfolioProject) error {
	return r.db.WithContext(ctx).Create(project).Error
}

// DeletePortfolioProject удаляет проект
func (r *OwnerCRMRepository) DeletePortfolioProject(ctx context.Context, ownerID, projectID int64) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND owner_id = ?", projectID, ownerID).
		Delete(&PortfolioProject{})

	if result.RowsAffected == 0 {
		return ErrItemNotFound
	}
	return result.Error
}

// ReorderPortfolio меняет порядок проектов
func (r *OwnerCRMRepository) ReorderPortfolio(ctx context.Context, ownerID int64, projectIDs []int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i, id := range projectIDs {
			if err := tx.Model(&PortfolioProject{}).
				Where("id = ? AND owner_id = ?", id, ownerID).
				Update("sort_order", i).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

type ClientInfo struct {
	ID            int64      `json:"id"`
	Name          string     `json:"name"`
	Email         string     `json:"email"`
	Phone         string     `json:"phone"`
	TotalBookings int64      `json:"total_bookings"`
	TotalSpent    float64    `json:"total_spent"`
	LastBookingAt *time.Time `json:"last_booking_at,omitempty"`
}

func (r *OwnerCRMRepository) GetClients(ctx context.Context, ownerID int64, search string, page, perPage int) ([]ClientInfo, int64, error) {
	if perPage == 0 {
		perPage = 20
	}
	if page == 0 {
		page = 1
	}
	offset := (page - 1) * perPage

	// 1) студии владельца
	var studioIDs []int64
	if err := r.db.WithContext(ctx).
		Table("studios").
		Where("owner_id = ?", ownerID).
		Pluck("id", &studioIDs).Error; err != nil {
		return nil, 0, err
	}
	if len(studioIDs) == 0 {
		return []ClientInfo{}, 0, nil
	}

	// 2) query
	query := r.db.WithContext(ctx).
		Table("users u").
		Select(`
			u.id,
			u.name,
			u.email,
			u.phone,
			COUNT(b.id) as total_bookings,
			COALESCE(SUM(b.total_price), 0) as total_spent,
			MAX(b.created_at) as last_booking_at
		`).
		Joins("JOIN bookings b ON b.user_id = u.id").
		Where("b.studio_id IN ?", studioIDs).
		Group("u.id, u.name, u.email, u.phone")

	// ⚠️ SQLite+PG совместимый поиск
	if search != "" {
		query = query.Where(`
			LOWER(u.name) LIKE LOWER(?) OR LOWER(u.email) LIKE LOWER(?) OR LOWER(u.phone) LIKE LOWER(?)
		`, "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	// 3) count через subquery
	var total int64
	countQ := r.db.WithContext(ctx).Table("(?) as subquery", query).Count(&total)
	if countQ.Error != nil {
		return nil, 0, countQ.Error
	}

	// 4) fetch
	var clients []ClientInfo
	if err := query.
		Order("total_bookings DESC, total_spent DESC").
		Limit(perPage).
		Offset(offset).
		Scan(&clients).Error; err != nil {
		return nil, 0, err
	}

	return clients, total, nil
}