package chat

import (
	"context"
	"database/sql"
	"time"

	"gorm.io/gorm"
)

// Repository handles all DB operations for the chat domain
type Repository interface {
	// Rooms
	CreateRoom(ctx context.Context, room *Room) error
	GetRoomByID(ctx context.Context, id string) (*Room, error)
	GetDirectRoomByUsers(ctx context.Context, userA, userB int64) (*Room, error)
	ListRoomsByUser(ctx context.Context, userID int64) ([]*RoomWithUnread, error)

	// Members
	AddMember(ctx context.Context, m *RoomMember) error
	RemoveMember(ctx context.Context, roomID string, userID int64) error
	GetMember(ctx context.Context, roomID string, userID int64) (*RoomMember, error)
	GetMembers(ctx context.Context, roomID string) ([]*RoomMember, error)
	IsMember(ctx context.Context, roomID string, userID int64) (bool, error)
	UpdateLastRead(ctx context.Context, roomID string, userID int64) error

	// Messages
	CreateMessage(ctx context.Context, msg *Message) error
	GetMessages(ctx context.Context, roomID string, limit, offset int) ([]*Message, error)
	CountUnread(ctx context.Context, roomID string, userID int64) (int, error)
	MarkRoomAsRead(ctx context.Context, roomID string, userID int64) error
	CountTotalUnread(ctx context.Context, userID int64) (int, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) CreateRoom(ctx context.Context, room *Room) error {
	return r.db.WithContext(ctx).Create(room).Error
}

func (r *repository) GetRoomByID(ctx context.Context, id string) (*Room, error) {
	var room Room
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&room).Error
	if err == gorm.ErrRecordNotFound {
		return nil, ErrRoomNotFound
	}
	return &room, err
}

func (r *repository) GetDirectRoomByUsers(ctx context.Context, userA, userB int64) (*Room, error) {
	var room Room
	err := r.db.WithContext(ctx).
		Joins("JOIN chat_room_members rm1 ON rm1.room_id = chat_rooms.id AND rm1.user_id = ?", userA).
		Joins("JOIN chat_room_members rm2 ON rm2.room_id = chat_rooms.id AND rm2.user_id = ?", userB).
		Where("chat_rooms.type = ?", RoomTypeDirect).
		First(&room).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &room, err
}

func (r *repository) ListRoomsByUser(ctx context.Context, userID int64) ([]*RoomWithUnread, error) {
	var rooms []*Room
	err := r.db.WithContext(ctx).
		Joins("JOIN chat_room_members rm ON rm.room_id = chat_rooms.id AND rm.user_id = ?", userID).
		Order("chat_rooms.created_at DESC").
		Find(&rooms).Error
	if err != nil {
		return nil, err
	}

	result := make([]*RoomWithUnread, 0, len(rooms))
	for _, room := range rooms {
		unread, _ := r.CountUnread(ctx, room.ID, userID)
		members, _ := r.GetMembers(ctx, room.ID)
		result = append(result, &RoomWithUnread{
			Room:        room,
			UnreadCount: unread,
			Members:     members,
		})
	}
	return result, nil
}

func (r *repository) AddMember(ctx context.Context, m *RoomMember) error {
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *repository) RemoveMember(ctx context.Context, roomID string, userID int64) error {
	return r.db.WithContext(ctx).
		Where("room_id = ? AND user_id = ?", roomID, userID).
		Delete(&RoomMember{}).Error
}

func (r *repository) GetMember(ctx context.Context, roomID string, userID int64) (*RoomMember, error) {
	var m RoomMember
	err := r.db.WithContext(ctx).
		Where("room_id = ? AND user_id = ?", roomID, userID).
		First(&m).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &m, err
}

func (r *repository) GetMembers(ctx context.Context, roomID string) ([]*RoomMember, error) {
	var members []*RoomMember
	err := r.db.WithContext(ctx).
		Where("room_id = ?", roomID).
		Order("joined_at ASC").
		Find(&members).Error
	return members, err
}

func (r *repository) IsMember(ctx context.Context, roomID string, userID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&RoomMember{}).
		Where("room_id = ? AND user_id = ?", roomID, userID).
		Count(&count).Error
	return count > 0, err
}

func (r *repository) UpdateLastRead(ctx context.Context, roomID string, userID int64) error {
	return r.db.WithContext(ctx).
		Model(&RoomMember{}).
		Where("room_id = ? AND user_id = ?", roomID, userID).
		Update("last_read_at", time.Now()).Error
}

func (r *repository) CreateMessage(ctx context.Context, msg *Message) error {
	return r.db.WithContext(ctx).Create(msg).Error
}

func (r *repository) GetMessages(ctx context.Context, roomID string, limit, offset int) ([]*Message, error) {
	var msgs []*Message
	err := r.db.WithContext(ctx).
		Where("room_id = ?", roomID).
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&msgs).Error
	if err != nil {
		return nil, err
	}

	// Enrich with upload attachment data
	for _, msg := range msgs {
		if msg.UploadID.Valid {
			var fileURL, origName, mimeType string
			r.db.WithContext(ctx).
				Table("uploads").
				Select("file_url, original_name, mime_type").
				Where("id = ?", msg.UploadID.String).
				Row().Scan(&fileURL, &origName, &mimeType)
			msg.AttachmentURL = fileURL
			msg.AttachmentName = origName
			msg.AttachmentMime = mimeType
		}
	}
	return msgs, nil
}

func (r *repository) CountUnread(ctx context.Context, roomID string, userID int64) (int, error) {
	var lastRead sql.NullTime
	r.db.WithContext(ctx).
		Model(&RoomMember{}).
		Select("last_read_at").
		Where("room_id = ? AND user_id = ?", roomID, userID).
		Scan(&lastRead)

	var count int64
	q := r.db.WithContext(ctx).
		Model(&Message{}).
		Where("room_id = ? AND sender_id != ?", roomID, userID)
	if lastRead.Valid {
		q = q.Where("created_at > ?", lastRead.Time)
	}
	err := q.Count(&count).Error
	return int(count), err
}

func (r *repository) MarkRoomAsRead(ctx context.Context, roomID string, userID int64) error {
	return r.UpdateLastRead(ctx, roomID, userID)
}

func (r *repository) CountTotalUnread(ctx context.Context, userID int64) (int, error) {
	var total int64
	err := r.db.WithContext(ctx).
		Table("messages m").
		Joins("JOIN chat_room_members rm ON rm.room_id = m.room_id AND rm.user_id = ?", userID).
		Where("m.sender_id != ? AND (rm.last_read_at IS NULL OR m.created_at > rm.last_read_at)", userID).
		Count(&total).Error
	return int(total), err
}
