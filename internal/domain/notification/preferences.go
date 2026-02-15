package notification

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// UserPreferences holds user notification preferences
type UserPreferences struct {
	ID              int64                      `gorm:"primaryKey;column:id" json:"id"`
	UserID          int64                      `gorm:"column:user_id;uniqueIndex" json:"user_id"`
	EmailEnabled    bool                       `gorm:"column:email_enabled;default:true" json:"email_enabled"`
	PushEnabled     bool                       `gorm:"column:push_enabled;default:true" json:"push_enabled"`
	InAppEnabled    bool                       `gorm:"column:in_app_enabled;default:true" json:"in_app_enabled"`
	DigestEnabled   bool                       `gorm:"column:digest_enabled;default:false" json:"digest_enabled"`
	DigestFrequency string                     `gorm:"column:digest_frequency;default:'weekly'" json:"digest_frequency"` // daily, weekly, monthly
	PerTypeSettings PerTypeSettingsMap         `gorm:"column:per_type_settings;type:jsonb;serializer:json" json:"per_type_settings"`
	CreatedAt       time.Time                  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time                  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName specifies table name for GORM
func (UserPreferences) TableName() string {
	return "user_notification_preferences"
}

// PerTypeSettingsMap holds channel settings per notification type
type PerTypeSettingsMap map[string]ChannelSettings

// Value implements driver.Valuer interface
func (p PerTypeSettingsMap) Value() (driver.Value, error) {
	if len(p) == 0 {
		return "{}", nil
	}
	return json.Marshal(p)
}

// Scan implements sql.Scanner interface
func (p *PerTypeSettingsMap) Scan(value interface{}) error {
	if value == nil {
		*p = make(PerTypeSettingsMap)
		return nil
	}

	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	result := make(PerTypeSettingsMap)
	if err := json.Unmarshal(b, &result); err != nil {
		return err
	}

	*p = result
	return nil
}

// GetChannelSettings gets settings for a notification type
func (p *UserPreferences) GetChannelSettings(notifType Type) ChannelSettings {
	if p.PerTypeSettings == nil {
		p.PerTypeSettings = make(PerTypeSettingsMap)
	}

	// Return default if not found
	if settings, ok := p.PerTypeSettings[string(notifType)]; ok {
		return settings
	}

	// Default: all channels enabled
	return ChannelSettings{
		InApp: p.InAppEnabled,
		Email: p.EmailEnabled,
		Push:  p.PushEnabled,
	}
}

// SetChannelSettings sets settings for a notification type
func (p *UserPreferences) SetChannelSettings(notifType Type, settings ChannelSettings) {
	if p.PerTypeSettings == nil {
		p.PerTypeSettings = make(PerTypeSettingsMap)
	}
	p.PerTypeSettings[string(notifType)] = settings
}

// GetDefaults returns default preferences for new user
func GetDefaultPreferences(userID int64) *UserPreferences {
	return &UserPreferences{
		UserID:          userID,
		EmailEnabled:    true,
		PushEnabled:     true,
		InAppEnabled:    true,
		DigestEnabled:   true,
		DigestFrequency: "weekly",
		PerTypeSettings: make(PerTypeSettingsMap),
	}
}

// DeviceToken represents a push notification device token
type DeviceToken struct {
	ID         int64     `gorm:"primaryKey;column:id" json:"id"`
	UserID     int64     `gorm:"column:user_id;index:idx_device_tokens_user" json:"user_id"`
	Token      string    `gorm:"column:token;uniqueIndex" json:"token"`
	Platform   string    `gorm:"column:platform" json:"platform"` // web, android, ios
	DeviceName string    `gorm:"column:device_name" json:"device_name,omitempty"`
	IsActive   bool      `gorm:"column:is_active;default:true;index:idx_device_tokens_active,expression:user_id,is_active" json:"is_active"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	LastUsedAt time.Time `gorm:"column:last_used_at" json:"last_used_at"`
}

// TableName specifies table name for GORM
func (DeviceToken) TableName() string {
	return "device_tokens"
}

// UpdateLastUsed updates the last_used_at timestamp
func (d *DeviceToken) UpdateLastUsed() {
	d.LastUsedAt = time.Now()
}
