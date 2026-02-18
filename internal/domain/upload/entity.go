package upload

import "time"

// Upload represents a physical file stored on the local filesystem.
// It is a shared infrastructure entity — any domain can reference an upload by its ID.
type Upload struct {
	ID           string    `gorm:"column:id;primaryKey" json:"id"`
	UserID       int64     `gorm:"column:user_id" json:"user_id"`
	OriginalName string    `gorm:"column:original_name" json:"original_name"`
	FilePath     string    `gorm:"column:file_path" json:"-"`  // relative disk path
	FileURL      string    `gorm:"column:file_url" json:"url"` // public HTTP URL
	MimeType     string    `gorm:"column:mime_type" json:"mime_type"`
	Size         int64     `gorm:"column:size" json:"size"`
	CreatedAt    time.Time `gorm:"column:created_at" json:"created_at"`
}

func (Upload) TableName() string { return "uploads" }
