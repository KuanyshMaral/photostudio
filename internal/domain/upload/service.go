package upload

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	MaxFileSize    = 50 * 1024 * 1024 // 50 MB
	UploadsBaseDir = "./uploads"
	StaticURLBase  = "/static/uploads"
)

// AllowedMimeTypes defines which file types are accepted
var AllowedMimeTypes = map[string]bool{
	"image/jpeg":      true,
	"image/png":       true,
	"image/gif":       true,
	"image/webp":      true,
	"image/svg+xml":   true,
	"video/mp4":       true,
	"video/webm":      true,
	"application/pdf": true,
}

// Service handles file upload to local disk.
// Simple: save file -> record in DB -> return ID + URL.
type Service struct {
	repo       Repository
	baseDir    string // absolute path to uploads dir
	staticBase string // URL prefix for serving files
}

func NewService(repo Repository, baseDir, staticBase string) *Service {
	if baseDir == "" {
		baseDir = UploadsBaseDir
	}
	if staticBase == "" {
		staticBase = StaticURLBase
	}
	return &Service{repo: repo, baseDir: baseDir, staticBase: staticBase}
}

// Upload saves a file to disk and records it in the database.
// Returns the Upload record with ID and URL.
func (s *Service) Upload(ctx context.Context, userID int64, fileHeader *multipart.FileHeader) (*Upload, error) {
	if fileHeader.Size == 0 {
		return nil, ErrEmptyFile
	}
	if fileHeader.Size > MaxFileSize {
		return nil, ErrFileTooLarge
	}

	// Open the file
	file, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Detect MIME type from first 512 bytes
	buf := make([]byte, 512)
	n, _ := file.Read(buf)
	mimeType := http.DetectContentType(buf[:n])
	mimeType = strings.Split(mimeType, ";")[0] // strip charset params

	if !AllowedMimeTypes[mimeType] {
		return nil, ErrInvalidMimeType
	}

	// Seek back to start
	if seeker, ok := file.(io.Seeker); ok {
		_, _ = seeker.Seek(0, io.SeekStart)
	}

	// Build directory: uploads/YYYY/MM/DD/
	now := time.Now()
	relDir := fmt.Sprintf("%d/%02d/%02d", now.Year(), now.Month(), now.Day())
	absDir := filepath.Join(s.baseDir, relDir)
	if err := os.MkdirAll(absDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %w", err)
	}

	// Generate unique filename
	id := uuid.New().String()
	ext := filepath.Ext(fileHeader.Filename)
	if ext == "" {
		ext = mimeToExt(mimeType)
	}
	safeOriginal := sanitizeName(fileHeader.Filename)
	filename := fmt.Sprintf("%s_%s%s", id, safeOriginal, ext)

	// Write file to disk
	absPath := filepath.Join(absDir, filename)
	dst, err := os.Create(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		_ = os.Remove(absPath)
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	// Build relative path and public URL
	relPath := filepath.Join(relDir, filename)
	fileURL := s.staticBase + "/" + strings.ReplaceAll(relPath, "\\", "/")

	// Save record to DB
	upload := &Upload{
		ID:           id,
		UserID:       userID,
		OriginalName: fileHeader.Filename,
		FilePath:     relPath,
		FileURL:      fileURL,
		MimeType:     mimeType,
		Size:         fileHeader.Size,
		CreatedAt:    now,
	}

	if err := s.repo.Create(ctx, upload); err != nil {
		_ = os.Remove(absPath) // rollback file on DB error
		return nil, fmt.Errorf("failed to save upload record: %w", err)
	}

	return upload, nil
}

// GetByID returns upload metadata by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Upload, error) {
	return s.repo.GetByID(ctx, id)
}

// Delete removes the physical file and the DB record.
func (s *Service) Delete(ctx context.Context, id string, userID int64) error {
	upload, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if upload.UserID != userID {
		return ErrNotOwner
	}

	// Delete physical file
	absPath := filepath.Join(s.baseDir, upload.FilePath)
	_ = os.Remove(absPath) // ignore error — file may already be gone

	return s.repo.Delete(ctx, id)
}

// ListByUser returns all uploads for a user.
func (s *Service) ListByUser(ctx context.Context, userID int64) ([]*Upload, error) {
	return s.repo.ListByUserID(ctx, userID)
}

func sanitizeName(name string) string {
	name = filepath.Base(name)
	name = strings.TrimSuffix(name, filepath.Ext(name)) // strip extension (added separately)
	name = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return '_'
	}, name)
	if len(name) > 40 {
		name = name[:40]
	}
	if name == "" {
		return "file"
	}
	return name
}

func mimeToExt(mime string) string {
	switch mime {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "video/mp4":
		return ".mp4"
	case "video/webm":
		return ".webm"
	case "application/pdf":
		return ".pdf"
	default:
		return ".bin"
	}
}
