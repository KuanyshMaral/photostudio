package payment

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) CreatePayment(ctx context.Context, p *Payment) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *Repository) GetPaymentByInvoiceID(ctx context.Context, invID int64) (*Payment, error) {
	var p Payment
	if err := r.db.WithContext(ctx).Where("robokassa_invoice_id = ?", invID).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *Repository) MarkPaymentStatus(ctx context.Context, invID int64, status string, paidAt *time.Time) (bool, error) {
	changed := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var p Payment
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("robokassa_invoice_id = ?", invID).First(&p).Error; err != nil {
			return err
		}
		if p.Status == status {
			return nil
		}
		updates := map[string]interface{}{"status": status}
		if paidAt != nil {
			updates["paid_at"] = *paidAt
		}
		if err := tx.Model(&Payment{}).Where("id = ?", p.ID).Updates(updates).Error; err != nil {
			return err
		}
		changed = true
		return nil
	})
	return changed, err
}

func (r *Repository) CreateSubscription(ctx context.Context, s *RecurringSubscription) error {
	return r.db.WithContext(ctx).Create(s).Error
}

func (r *Repository) GetSubscriptionByUserID(ctx context.Context, userID int64) (*RecurringSubscription, error) {
	var s RecurringSubscription
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at desc").First(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repository) UpdateSubscription(ctx context.Context, id string, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&RecurringSubscription{}).Where("id = ?", id).Updates(updates).Error
}
