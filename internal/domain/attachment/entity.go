package attachment

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// TargetType defines the business entity that owns the attachment.
// Mirrors the CHECK constraint in the migration.
type TargetType string

const (
	TargetStudioGallery TargetType = "studio_gallery"
	TargetRoomGallery   TargetType = "room_gallery"
	TargetReviewPhotos  TargetType = "review_photos"
	TargetChatMessage   TargetType = "chat_message"
)

func IsValidTargetType(t TargetType) bool {
	switch t {
	case TargetStudioGallery, TargetRoomGallery, TargetReviewPhotos, TargetChatMessage:
		return true
	}
	return false
}

// Metadata is optional structured data stored as JSONB per target type.
// e.g. caption for gallery photos.
type Metadata struct {
	Caption string `json:"caption,omitempty"`
	AltText string `json:"alt_text,omitempty"`
}

func (m Metadata) Value() (driver.Value, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("marshal attachment metadata: %w", err)
	}
	return string(b), nil
}

func (m *Metadata) Scan(src interface{}) error {
	var b []byte
	switch v := src.(type) {
	case string:
		b = []byte(v)
	case []byte:
		b = v
	case nil:
		return nil
	default:
		return fmt.Errorf("unexpected type for metadata: %T", src)
	}
	return json.Unmarshal(b, m)
}

// Attachment links an upload to a business entity (polymorphic 1:N).
// The upload record is the "dumb warehouse"; attachment provides the business label.
type Attachment struct {
	ID         int64      `gorm:"column:id;primaryKey" json:"id"`
	UploadID   string     `gorm:"column:upload_id" json:"upload_id"`
	TargetID   int64      `gorm:"column:target_id" json:"target_id"`
	TargetType TargetType `gorm:"column:target_type" json:"target_type"`
	SortOrder  int        `gorm:"column:sort_order" json:"sort_order"`
	Metadata   Metadata   `gorm:"column:metadata;type:jsonb;serializer:json" json:"metadata"`
	CreatedAt  time.Time  `gorm:"column:created_at" json:"created_at"`
}

func (Attachment) TableName() string { return "attachments" }

// AttachmentWithURL enriches an Attachment with its resolved public file URL and file info.
type AttachmentWithURL struct {
	Attachment
	URL          string `json:"url"`
	OriginalName string `json:"original_name"`
	MimeType     string `json:"mime_type"`
	Size         int64  `json:"size"`
}
