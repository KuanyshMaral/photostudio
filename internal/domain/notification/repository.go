package notification

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
)

// Repository defines notification data access interface
type Repository interface {
	Create(ctx context.Context, n *Notification) error
	GetByID(ctx context.Context, id int64) (*Notification, error)
	ListByUser(ctx context.Context, userID int64, limit, offset int) ([]*Notification, error)
	CountByUser(ctx context.Context, userID int64) (int64, error)
	CountUnreadByUser(ctx context.Context, userID int64) (int64, error)
	MarkAsRead(ctx context.Context, id int64) error
	MarkAllAsRead(ctx context.Context, userID int64) error
	Delete(ctx context.Context, id int64) error
	DeleteByUser(ctx context.Context, userID int64) error
	DeleteOldByUser(ctx context.Context, userID int64, days int) (int64, error)
	DeleteOlderThan(ctx context.Context, age time.Duration) (int64, error)
}

// PreferencesRepository defines user preferences data access interface
type PreferencesRepository interface {
	GetByUserID(ctx context.Context, userID int64) (*UserPreferences, error)
	Create(ctx context.Context, prefs *UserPreferences) error
	Update(ctx context.Context, prefs *UserPreferences) error
	Delete(ctx context.Context, userID int64) error
	ResetToDefaults(ctx context.Context, userID int64) error
}

// DeviceTokenRepository defines device tokens data access interface
type DeviceTokenRepository interface {
	Create(ctx context.Context, token *DeviceToken) error
	GetByID(ctx context.Context, id int64) (*DeviceToken, error)
	GetByToken(ctx context.Context, token string) (*DeviceToken, error)
	ListByUser(ctx context.Context, userID int64, activeOnly bool) ([]*DeviceToken, error)
	Update(ctx context.Context, token *DeviceToken) error
	Deactivate(ctx context.Context, id int64) error
	DeactivateByUser(ctx context.Context, userID int64) error
	Delete(ctx context.Context, id int64) error
	DeleteInactive(ctx context.Context, olderThan time.Duration) (int64, error)
}

// notificationRepository implements Repository interface
type notificationRepository struct {
	db *gorm.DB
}

// NewRepository creates notification repository
func NewRepository(db *gorm.DB) Repository {
	return &notificationRepository{db: db}
}

// NewRepositoryLegacy creates legacy repository for backward compatibility
func NewRepositoryLegacy(db *gorm.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

func (r *notificationRepository) Create(ctx context.Context, n *Notification) error {
	return r.db.WithContext(ctx).Create(n).Error
}

func (r *notificationRepository) GetByID(ctx context.Context, id int64) (*Notification, error) {
	var n Notification
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&n).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &n, nil
}

func (r *notificationRepository) ListByUser(ctx context.Context, userID int64, limit, offset int) ([]*Notification, error) {
	var notifications []*Notification
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&notifications).Error
	return notifications, err
}

func (r *notificationRepository) CountByUser(ctx context.Context, userID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&Notification{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	return count, err
}

func (r *notificationRepository) CountUnreadByUser(ctx context.Context, userID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Count(&count).Error
	return count, err
}

func (r *notificationRepository) MarkAsRead(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).
		Model(&Notification{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": time.Now(),
		}).Error
}

func (r *notificationRepository) MarkAllAsRead(ctx context.Context, userID int64) error {
	return r.db.WithContext(ctx).
		Model(&Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": time.Now(),
		}).Error
}

func (r *notificationRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&Notification{}, id).Error
}

func (r *notificationRepository) DeleteByUser(ctx context.Context, userID int64) error {
	return r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&Notification{}).Error
}

func (r *notificationRepository) DeleteOldByUser(ctx context.Context, userID int64, days int) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -days)
	result := r.db.WithContext(ctx).
		Where("user_id = ? AND created_at < ?", userID, cutoff).
		Delete(&Notification{})
	return result.RowsAffected, result.Error
}

func (r *notificationRepository) DeleteOlderThan(ctx context.Context, age time.Duration) (int64, error) {
	cutoff := time.Now().Add(-age)
	result := r.db.WithContext(ctx).
		Where("created_at < ?", cutoff).
		Delete(&Notification{})
	return result.RowsAffected, result.Error
}

// preferencesRepository implements PreferencesRepository interface
type preferencesRepository struct {
	db *gorm.DB
}

// NewPreferencesRepository creates preferences repository
func NewPreferencesRepository(db *gorm.DB) PreferencesRepository {
	return &preferencesRepository{db: db}
}

func (r *preferencesRepository) GetByUserID(ctx context.Context, userID int64) (*UserPreferences, error) {
	var prefs UserPreferences
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		First(&prefs).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Return default preferences
			return GetDefaultPreferences(userID), nil
		}
		return nil, err
	}
	return &prefs, nil
}

func (r *preferencesRepository) Create(ctx context.Context, prefs *UserPreferences) error {
	return r.db.WithContext(ctx).Create(prefs).Error
}

func (r *preferencesRepository) Update(ctx context.Context, prefs *UserPreferences) error {
	return r.db.WithContext(ctx).
		Model(prefs).
		Save(prefs).Error
}

func (r *preferencesRepository) Delete(ctx context.Context, userID int64) error {
	return r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&UserPreferences{}).Error
}

func (r *preferencesRepository) ResetToDefaults(ctx context.Context, userID int64) error {
	defaults := GetDefaultPreferences(userID)
	perTypeSettings, _ := json.Marshal(defaults.PerTypeSettings)

	return r.db.WithContext(ctx).
		Model(&UserPreferences{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"email_enabled":     true,
			"push_enabled":      true,
			"in_app_enabled":    true,
			"digest_enabled":    true,
			"digest_frequency":  "weekly",
			"per_type_settings": perTypeSettings,
		}).Error
}

// deviceTokenRepository implements DeviceTokenRepository interface
type deviceTokenRepository struct {
	db *gorm.DB
}

// NewDeviceTokenRepository creates device token repository
func NewDeviceTokenRepository(db *gorm.DB) DeviceTokenRepository {
	return &deviceTokenRepository{db: db}
}

func (r *deviceTokenRepository) Create(ctx context.Context, token *DeviceToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *deviceTokenRepository) GetByID(ctx context.Context, id int64) (*DeviceToken, error) {
	var token DeviceToken
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&token).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &token, nil
}

func (r *deviceTokenRepository) GetByToken(ctx context.Context, token string) (*DeviceToken, error) {
	var dt DeviceToken
	err := r.db.WithContext(ctx).Where("token = ?", token).First(&dt).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &dt, nil
}

func (r *deviceTokenRepository) ListByUser(ctx context.Context, userID int64, activeOnly bool) ([]*DeviceToken, error) {
	var tokens []*DeviceToken
	query := r.db.WithContext(ctx).Where("user_id = ?", userID)
	if activeOnly {
		query = query.Where("is_active = ?", true)
	}
	err := query.Order("created_at DESC").Find(&tokens).Error
	return tokens, err
}

func (r *deviceTokenRepository) Update(ctx context.Context, token *DeviceToken) error {
	return r.db.WithContext(ctx).Save(token).Error
}

func (r *deviceTokenRepository) Deactivate(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).
		Model(&DeviceToken{}).
		Where("id = ?", id).
		Update("is_active", false).Error
}

func (r *deviceTokenRepository) DeactivateByUser(ctx context.Context, userID int64) error {
	return r.db.WithContext(ctx).
		Model(&DeviceToken{}).
		Where("user_id = ?", userID).
		Update("is_active", false).Error
}

func (r *deviceTokenRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&DeviceToken{}, id).Error
}

func (r *deviceTokenRepository) DeleteInactive(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	result := r.db.WithContext(ctx).
		Where("is_active = ? AND last_used_at < ?", false, cutoff).
		Delete(&DeviceToken{})
	return result.RowsAffected, result.Error
}

// Legacy code for backward compatibility
type notificationModel struct {
	ID        int64   `gorm:"column:id;primaryKey"`
	UserID    int64   `gorm:"column:user_id"`
	Type      string  `gorm:"column:type"`
	Title     string  `gorm:"column:title"`
	Message   *string `gorm:"column:message"`
	IsRead    bool    `gorm:"column:is_read"`
	Data      []byte  `gorm:"column:data"`
	CreatedAt int64   `gorm:"-"`
}

type NotificationRepository struct {
	db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

func (r *NotificationRepository) DB() *gorm.DB {
	return r.db
}

func (r *NotificationRepository) Create(ctx context.Context, n *Notification, data map[string]any) error {
	var raw []byte
	if data != nil {
		b, err := json.Marshal(data)
		if err != nil {
			return err
		}
		raw = b
	}

	return r.db.WithContext(ctx).Create(n).Error
}

func (r *NotificationRepository) GetByUserID(ctx context.Context, userID int64, limit int) ([]Notification, error) {
	var notifications []Notification
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&notifications).Error
	return notifications, err
}

func (r *NotificationRepository) CountUnread(ctx context.Context, userID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Count(&count).Error
	return count, err
}

func (r *NotificationRepository) MarkAsRead(ctx context.Context, id, userID int64) error {
	result := r.db.WithContext(ctx).
		Model(&Notification{}).
		Where("id = ? AND user_id = ?", id, userID).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": time.Now(),
		})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *NotificationRepository) MarkAllAsRead(ctx context.Context, userID int64) error {
	return r.db.WithContext(ctx).
		Model(&Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": time.Now(),
		}).Error
}

var ErrNotificationNotFound = errors.New("notification not found")