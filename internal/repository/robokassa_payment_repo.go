package repository

import (
	"context"
	"errors"
	"photostudio/internal/domain"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RobokassaPaymentRepository struct {
	db *gorm.DB
}

func NewRobokassaPaymentRepository(db *gorm.DB) *RobokassaPaymentRepository {
	return &RobokassaPaymentRepository{db: db}
}

func (r *RobokassaPaymentRepository) Create(ctx context.Context, p *domain.RobokassaPayment) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *RobokassaPaymentRepository) GetByInvID(ctx context.Context, invID int64) (*domain.RobokassaPayment, error) {
	var p domain.RobokassaPayment
	if err := r.db.WithContext(ctx).Where("inv_id = ?", invID).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *RobokassaPaymentRepository) UpdateStatus(ctx context.Context, invID int64, status domain.RobokassaPaymentStatus, rawBody, reason string, paidAt *time.Time) error {
	updates := map[string]interface{}{
		"status":          status,
		"result_raw_body": rawBody,
		"failure_reason":  reason,
	}
	if paidAt != nil {
		updates["paid_at"] = *paidAt
	}
	return r.db.WithContext(ctx).Model(&domain.RobokassaPayment{}).Where("inv_id = ?", invID).Updates(updates).Error
}

func (r *RobokassaPaymentRepository) UpdateStatusPendingIfNotPaid(ctx context.Context, invID int64, rawBody string) error {
	res := r.db.WithContext(ctx).
		Model(&domain.RobokassaPayment{}).
		Where("inv_id = ? AND status <> ?", invID, domain.PaymentStatusPaid).
		Updates(map[string]interface{}{
			"status":           domain.PaymentStatusPending,
			"success_raw_body": rawBody,
		})
	if res.Error != nil {
		return res.Error
	}
	var existing int64
	if err := r.db.WithContext(ctx).Model(&domain.RobokassaPayment{}).Where("inv_id = ?", invID).Count(&existing).Error; err != nil {
		return err
	}
	if existing == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *RobokassaPaymentRepository) SaveSuccessRawBody(ctx context.Context, invID int64, rawBody string) error {
	return r.db.WithContext(ctx).Model(&domain.RobokassaPayment{}).Where("inv_id = ?", invID).Update("success_raw_body", rawBody).Error
}

func (r *RobokassaPaymentRepository) MarkPaidIdempotent(ctx context.Context, invID int64, rawBody string, paidAt time.Time) (bool, error) {
	var changed bool
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var p domain.RobokassaPayment
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("inv_id = ?", invID).First(&p).Error; err != nil {
			return err
		}
		if p.Status == domain.PaymentStatusPaid {
			changed = false
			return nil
		}
		res := tx.Model(&domain.RobokassaPayment{}).Where("inv_id = ?", invID).Updates(map[string]interface{}{
			"status":          domain.PaymentStatusPaid,
			"result_raw_body": rawBody,
			"paid_at":         paidAt,
		})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errors.New("payment row not updated")
		}
		changed = true
		return nil
	})
	return changed, err
}
