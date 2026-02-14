package notification

import (
	"context"
	"encoding/json"
	"errors"
	"gorm.io/gorm"
)

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

	var msg *string
	if n.Message != "" {
		m := n.Message
		msg = &m
	}

	m := &notificationModel{
		UserID:  n.UserID,
		Type:    string(n.Type),
		Title:   n.Title,
		Message: msg,
		IsRead:  n.IsRead,
		Data:    raw,
	}

	return r.db.WithContext(ctx).Table("notifications").Create(m).Error
}

func (r *NotificationRepository) GetByUserID(ctx context.Context, userID int64, limit int) ([]Notification, error) {
	type row struct {
		ID        int64               `gorm:"column:id"`
		UserID    int64               `gorm:"column:user_id"`
		Type      string              `gorm:"column:type"`
		Title     string              `gorm:"column:title"`
		Message   *string             `gorm:"column:message"`
		IsRead    bool                `gorm:"column:is_read"`
		Data      []byte              `gorm:"column:data"`
		CreatedAt Notification `gorm:"-"`
	}

	var rows []struct {
		ID        int64   `gorm:"column:id"`
		UserID    int64   `gorm:"column:user_id"`
		Type      string  `gorm:"column:type"`
		Title     string  `gorm:"column:title"`
		Message   *string `gorm:"column:message"`
		IsRead    bool    `gorm:"column:is_read"`
		Data      []byte  `gorm:"column:data"`
		CreatedAt string  `gorm:"column:created_at"`
	}

	q := r.db.WithContext(ctx).
		Table("notifications").
		Where("user_id = ?", userID).
		Order("created_at DESC")

	if limit > 0 {
		q = q.Limit(limit)
	}

	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]Notification, 0, len(rows))
	for _, it := range rows {
		var decoded any
		if len(it.Data) > 0 {
			_ = json.Unmarshal(it.Data, &decoded)
		}

		msg := ""
		if it.Message != nil {
			msg = *it.Message
		}

		out = append(out, Notification{
			ID:      it.ID,
			UserID:  it.UserID,
			Type:    NotificationType(it.Type),
			Title:   it.Title,
			Message: msg,
			IsRead:  it.IsRead,
			Data:    decoded,
		})
	}
	return out, nil
}

func (r *NotificationRepository) CountUnread(ctx context.Context, userID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("notifications").
		Where("user_id = ? AND is_read = FALSE", userID).
		Count(&count).Error
	return count, err
}

func (r *NotificationRepository) MarkAsRead(ctx context.Context, id, userID int64) error {
	res := r.db.WithContext(ctx).
		Table("notifications").
		Where("id = ? AND user_id = ?", id, userID).
		Update("is_read", true)

	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *NotificationRepository) MarkAllAsRead(ctx context.Context, userID int64) error {
	return r.db.WithContext(ctx).
		Table("notifications").
		Where("user_id = ? AND is_read = FALSE", userID).
		Update("is_read", true).Error
}

var ErrNotificationNotFound = errors.New("notification not found")