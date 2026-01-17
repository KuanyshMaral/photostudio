package repository

import (
	"context"

	"photostudio/internal/domain"

	"gorm.io/gorm"
)

type OwnerRepository struct {
	db *gorm.DB
}

func NewOwnerRepository(db *gorm.DB) *OwnerRepository {
	return &OwnerRepository{db: db}
}

func (r *OwnerRepository) DB() *gorm.DB {
	return r.db
}

// Create creates a studio owner verification application.
// NOTE: status is stored in users.studio_status (see migrations), not in studio_owners.
func (r *OwnerRepository) Create(ctx context.Context, owner *domain.StudioOwner) error {
	return r.db.WithContext(ctx).Create(owner).Error
}

// AppendVerificationDocs updates docs array (if used in your project).
func (r *OwnerRepository) AppendVerificationDocs(ctx context.Context, ownerID int64, docs []string) error {
	return r.db.WithContext(ctx).
		Model(&domain.StudioOwner{}).
		Where("id = ?", ownerID).
		Update("verification_docs", docs).Error
}

func (r *OwnerRepository) FindByID(ctx context.Context, id int64) (*domain.StudioOwner, error) {
	var owner domain.StudioOwner
	if err := r.db.WithContext(ctx).First(&owner, id).Error; err != nil {
		return nil, err
	}
	return &owner, nil
}

// Update updates only moderation-related fields (verified_at/verified_by/rejected_reason/admin_notes).
func (r *OwnerRepository) Update(ctx context.Context, owner *domain.StudioOwner) error {
	return r.db.WithContext(ctx).
		Model(&domain.StudioOwner{}).
		Where("id = ?", owner.ID).
		Updates(map[string]interface{}{
			"verified_at":     owner.VerifiedAt,
			"verified_by":     owner.VerifiedBy,
			"rejected_reason": owner.RejectedReason,
			"admin_notes":     owner.AdminNotes,
		}).Error
}

// FindPendingPaginated returns pending applications for moderation.
// IMPORTANT: pending status is taken from users.studio_status.
func (r *OwnerRepository) FindPendingPaginated(ctx context.Context, offset, limit int) ([]domain.PendingStudioOwnerRow, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).
		Table("studio_owners so").
		Joins("JOIN users u ON u.id = so.user_id").
		Where("u.studio_status = ?", domain.StatusPending).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []domain.PendingStudioOwnerRow
	if err := r.db.WithContext(ctx).
		Table("studio_owners so").
		Select("so.id, so.user_id, so.bin, so.company_name, u.studio_status AS status, so.created_at").
		Joins("JOIN users u ON u.id = so.user_id").
		Where("u.studio_status = ?", domain.StatusPending).
		Order("so.created_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(&rows).Error; err != nil {
		return nil, 0, err
	}

	return rows, total, nil
}
