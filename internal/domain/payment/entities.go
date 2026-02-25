package payment

import "time"

type Payment struct {
	ID                 int64      `gorm:"primaryKey" json:"id"`
	UserID             int64      `gorm:"not null;index" json:"user_id"`
	BookingID          *int64     `gorm:"index" json:"booking_id,omitempty"`
	SubscriptionID     *string    `gorm:"type:uuid;index" json:"subscription_id,omitempty"`
	RobokassaInvoiceID int64      `gorm:"uniqueIndex;not null" json:"robokassa_invoice_id"`
	Amount             string     `gorm:"type:numeric(12,2);not null" json:"amount"`
	Status             string     `gorm:"type:varchar(20);not null;default:'created';index" json:"status"`
	IsRecurring        bool       `gorm:"not null;default:false" json:"is_recurring"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	PaidAt             *time.Time `json:"paid_at,omitempty"`
}

func (Payment) TableName() string { return "payments" }

type RecurringSubscription struct {
	ID             string     `gorm:"type:uuid;primaryKey" json:"id"`
	UserID         int64      `gorm:"not null;index" json:"user_id"`
	Status         string     `gorm:"type:varchar(20);not null;default:'pending';index" json:"status"`
	FirstInvoiceID *int64     `gorm:"uniqueIndex" json:"first_invoice_id,omitempty"`
	NextBillingAt  *time.Time `json:"next_billing_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (RecurringSubscription) TableName() string { return "recurring_subscriptions" }
