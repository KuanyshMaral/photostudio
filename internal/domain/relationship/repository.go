package relationship

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrAlreadyBlocked  = errors.New("user is already blocked")
	ErrNotBlocked      = errors.New("user is not blocked")
	ErrCannotBlockSelf = errors.New("cannot block yourself")
)

// BlockRelation represents a user-to-user block
type BlockRelation struct {
	ID        string    `gorm:"column:id;primaryKey"`
	BlockerID int64     `gorm:"column:blocker_id"`
	BlockedID int64     `gorm:"column:blocked_id"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (BlockRelation) TableName() string { return "block_relations" }

// Repository handles persistence for block relations
type Repository interface {
	Block(ctx context.Context, blockerID, blockedID int64) error
	Unblock(ctx context.Context, blockerID, blockedID int64) error
	IsBlocked(ctx context.Context, blockerID, blockedID int64) (bool, error)
	ListBlocked(ctx context.Context, blockerID int64) ([]*BlockRelation, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Block(ctx context.Context, blockerID, blockedID int64) error {
	rel := &BlockRelation{
		ID:        uuid.New().String(),
		BlockerID: blockerID,
		BlockedID: blockedID,
		CreatedAt: time.Now(),
	}
	err := r.db.WithContext(ctx).Create(rel).Error
	if err != nil && isDuplicateError(err) {
		return ErrAlreadyBlocked
	}
	return err
}

func (r *repository) Unblock(ctx context.Context, blockerID, blockedID int64) error {
	result := r.db.WithContext(ctx).
		Where("blocker_id = ? AND blocked_id = ?", blockerID, blockedID).
		Delete(&BlockRelation{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotBlocked
	}
	return nil
}

func (r *repository) IsBlocked(ctx context.Context, blockerID, blockedID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&BlockRelation{}).
		Where("(blocker_id = ? AND blocked_id = ?) OR (blocker_id = ? AND blocked_id = ?)",
			blockerID, blockedID, blockedID, blockerID).
		Count(&count).Error
	return count > 0, err
}

func (r *repository) ListBlocked(ctx context.Context, blockerID int64) ([]*BlockRelation, error) {
	var rels []*BlockRelation
	err := r.db.WithContext(ctx).
		Where("blocker_id = ?", blockerID).
		Order("created_at DESC").
		Find(&rels).Error
	return rels, err
}

func isDuplicateError(err error) bool {
	return err != nil && (contains(err.Error(), "duplicate") || contains(err.Error(), "unique"))
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
