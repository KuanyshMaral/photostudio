package subscription

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// Repository handles persistence for subscription data
type Repository interface {
	// Plans
	ListPlans(ctx context.Context) ([]*Plan, error)
	GetPlanByID(ctx context.Context, id PlanID) (*Plan, error)

	// Subscriptions
	GetActiveByOwnerID(ctx context.Context, ownerID int64) (*Subscription, error)
	GetByID(ctx context.Context, id string) (*Subscription, error)
	Create(ctx context.Context, sub *Subscription) error
	Update(ctx context.Context, sub *Subscription) error
	Cancel(ctx context.Context, id string, reason string) error
	ExpireOldSubscriptions(ctx context.Context) (int, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) ListPlans(ctx context.Context) ([]*Plan, error) {
	var plans []*Plan
	err := r.db.WithContext(ctx).Where("is_active = ?", true).Order("price_monthly ASC").Find(&plans).Error
	return plans, err
}

func (r *repository) GetPlanByID(ctx context.Context, id PlanID) (*Plan, error) {
	var plan Plan
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&plan).Error
	if err != nil {
		return nil, err
	}
	return &plan, nil
}

func (r *repository) GetActiveByOwnerID(ctx context.Context, ownerID int64) (*Subscription, error) {
	var sub Subscription
	err := r.db.WithContext(ctx).
		Where("owner_id = ? AND status = ?", ownerID, StatusActive).
		Order("created_at DESC").
		First(&sub).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &sub, nil
}

func (r *repository) GetByID(ctx context.Context, id string) (*Subscription, error) {
	var sub Subscription
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&sub).Error
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

func (r *repository) Create(ctx context.Context, sub *Subscription) error {
	return r.db.WithContext(ctx).Create(sub).Error
}

func (r *repository) Update(ctx context.Context, sub *Subscription) error {
	return r.db.WithContext(ctx).Save(sub).Error
}

func (r *repository) Cancel(ctx context.Context, id string, reason string) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&Subscription{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":        StatusCancelled,
			"cancel_reason": reason,
			"cancelled_at":  now,
			"updated_at":    now,
		}).Error
}

func (r *repository) ExpireOldSubscriptions(ctx context.Context) (int, error) {
	result := r.db.WithContext(ctx).
		Model(&Subscription{}).
		Where("status = ? AND expires_at IS NOT NULL AND expires_at < NOW()", StatusActive).
		Updates(map[string]any{
			"status":     StatusExpired,
			"updated_at": time.Now(),
		})
	return int(result.RowsAffected), result.Error
}
