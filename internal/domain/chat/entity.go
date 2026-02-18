package chat

import (
	"database/sql"
	"time"
)

// RoomType distinguishes direct (1-on-1) from group chats
type RoomType string

const (
	RoomTypeDirect RoomType = "direct"
	RoomTypeGroup  RoomType = "group"
)

// MemberRole distinguishes admins (group creators) from regular members
type MemberRole string

const (
	MemberRoleAdmin  MemberRole = "admin"
	MemberRoleMember MemberRole = "member"
)

// Room is a chat room — either a direct 1-on-1 or a named group
type Room struct {
	ID        string         `gorm:"column:id;primaryKey" json:"id"`
	Type      RoomType       `gorm:"column:type" json:"type"`
	Name      sql.NullString `gorm:"column:name" json:"name,omitempty"`
	CreatorID sql.NullInt64  `gorm:"column:creator_id" json:"creator_id,omitempty"`
	CreatedAt time.Time      `gorm:"column:created_at" json:"created_at"`
}

func (Room) TableName() string { return "chat_rooms" }

// RoomMember is a participant in a room
type RoomMember struct {
	RoomID     string       `gorm:"column:room_id;primaryKey" json:"room_id"`
	UserID     int64        `gorm:"column:user_id;primaryKey" json:"user_id"`
	Role       MemberRole   `gorm:"column:role" json:"role"`
	LastReadAt sql.NullTime `gorm:"column:last_read_at" json:"last_read_at,omitempty"`
	JoinedAt   time.Time    `gorm:"column:joined_at" json:"joined_at"`
}

func (RoomMember) TableName() string { return "chat_room_members" }

func (m *RoomMember) IsAdmin() bool { return m.Role == MemberRoleAdmin }

// Message is a single chat message, optionally with a file attachment
type Message struct {
	ID        string         `gorm:"column:id;primaryKey" json:"id"`
	RoomID    string         `gorm:"column:room_id" json:"room_id"`
	SenderID  int64          `gorm:"column:sender_id" json:"sender_id"`
	Content   string         `gorm:"column:content" json:"content"`
	UploadID  sql.NullString `gorm:"column:upload_id" json:"upload_id,omitempty"` // FK -> uploads.id
	IsRead    bool           `gorm:"column:is_read" json:"is_read"`
	CreatedAt time.Time      `gorm:"column:created_at" json:"created_at"`

	// Joined from uploads table (populated by repo)
	AttachmentURL  string `gorm:"-" json:"attachment_url,omitempty"`
	AttachmentName string `gorm:"-" json:"attachment_name,omitempty"`
	AttachmentMime string `gorm:"-" json:"attachment_mime,omitempty"`
}

func (Message) TableName() string { return "messages" }

// RoomWithUnread is used in list responses
type RoomWithUnread struct {
	*Room
	UnreadCount int
	Members     []*RoomMember
}
