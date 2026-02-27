package attachment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"photostudio/internal/domain/upload"
)

var (
	ErrNotFound      = errors.New("attachment not found")
	ErrNotOwner      = errors.New("you do not own this attachment")
	ErrInvalidTarget = errors.New("invalid target type")
)

// Service handles attachment business logic.
// It delegates all file I/O to the upload service and only manages business labels.
type Service struct {
	repo    Repository
	uploads *upload.Service
}

func NewService(repo Repository, uploads *upload.Service) *Service {
	return &Service{repo: repo, uploads: uploads}
}

// Attach links one or more existing uploads to a business entity.
// The caller provides upload IDs (from POST /uploads), a target, and optional metadata.
func (s *Service) Attach(
	ctx context.Context,
	uploadIDs []string,
	callerID int64,
	targetType TargetType,
	targetID int64,
	metadata Metadata,
) ([]*AttachmentWithURL, error) {
	if !IsValidTargetType(targetType) {
		return nil, ErrInvalidTarget
	}

	// Get current count for auto sort_order
	count, err := s.repo.CountByTarget(ctx, targetType, targetID)
	if err != nil {
		return nil, fmt.Errorf("count attachments: %w", err)
	}

	results := make([]*AttachmentWithURL, 0, len(uploadIDs))
	for i, uploadID := range uploadIDs {
		// Verify the upload exists and belongs to the caller
		u, err := s.uploads.GetByID(ctx, uploadID)
		if err != nil {
			return nil, fmt.Errorf("upload %s: %w", uploadID, err)
		}
		if u.UserID != callerID {
			return nil, fmt.Errorf("upload %s: %w", uploadID, ErrNotOwner)
		}

		a := &Attachment{
			UploadID:   uploadID,
			TargetID:   targetID,
			TargetType: targetType,
			SortOrder:  count + i,
			Metadata:   metadata,
			CreatedAt:  time.Now(),
		}

		if err := s.repo.Create(ctx, a); err != nil {
			return nil, fmt.Errorf("create attachment: %w", err)
		}

		results = append(results, s.enrich(a, u))
	}
	return results, nil
}

// ListByTarget returns all attachments for an entity, enriched with file URL info.
func (s *Service) ListByTarget(ctx context.Context, targetType TargetType, targetID int64) ([]*AttachmentWithURL, error) {
	if !IsValidTargetType(targetType) {
		return nil, ErrInvalidTarget
	}
	attachments, err := s.repo.ListByTarget(ctx, targetType, targetID)
	if err != nil {
		return nil, err
	}
	result := make([]*AttachmentWithURL, 0, len(attachments))
	for _, a := range attachments {
		u, err := s.uploads.GetByID(ctx, a.UploadID)
		if err != nil {
			continue // gracefully skip orphaned attachments
		}
		result = append(result, s.enrich(a, u))
	}
	return result, nil
}

// Delete removes an attachment record. The underlying upload file is NOT deleted.
func (s *Service) Delete(ctx context.Context, id int64, callerID int64) error {
	a, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return ErrNotFound
	}

	// Verify ownership through the upload's user_id
	u, err := s.uploads.GetByID(ctx, a.UploadID)
	if err != nil {
		return err
	}
	if u.UserID != callerID {
		return ErrNotOwner
	}

	return s.repo.Delete(ctx, id)
}

// Reorder updates the sort_order for a slice of attachment IDs.
func (s *Service) Reorder(ctx context.Context, ids []int64) error {
	for i, id := range ids {
		if err := s.repo.UpdateSortOrder(ctx, id, i); err != nil {
			return fmt.Errorf("update sort order for %d: %w", id, err)
		}
	}
	return nil
}

func (s *Service) enrich(a *Attachment, u *upload.Upload) *AttachmentWithURL {
	return &AttachmentWithURL{
		Attachment:   *a,
		URL:          u.FileURL,
		OriginalName: u.OriginalName,
		MimeType:     u.MimeType,
		Size:         u.Size,
	}
}
