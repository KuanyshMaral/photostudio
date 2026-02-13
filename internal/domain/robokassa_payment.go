package domain

import "time"

type RobokassaPaymentStatus string

const (
	PaymentStatusCreated RobokassaPaymentStatus = "created"
	PaymentStatusPending RobokassaPaymentStatus = "pending"
	PaymentStatusPaid    RobokassaPaymentStatus = "paid"
	PaymentStatusFailed  RobokassaPaymentStatus = "failed"
)

type RobokassaPayment struct {
	ID             int64                  `gorm:"primaryKey" json:"id"`
	BookingID      int64                  `gorm:"index;not null" json:"booking_id"`
	OutSum         string                 `gorm:"type:varchar(32);not null" json:"out_sum"`
	InvID          int64                  `gorm:"uniqueIndex;not null" json:"inv_id"`
	Description    string                 `gorm:"type:text" json:"description"`
	Status         RobokassaPaymentStatus `gorm:"type:varchar(20);default:'created';index" json:"status"`
	Signature      string                 `gorm:"type:varchar(128)" json:"signature"`
	RobokassaURL   string                 `gorm:"type:text" json:"robokassa_url"`
	ShpParams      string                 `gorm:"type:text" json:"shp_params"`
	ResultRawBody  string                 `gorm:"type:text" json:"result_raw_body"`
	SuccessRawBody string                 `gorm:"type:text" json:"success_raw_body"`
	FailureReason  string                 `gorm:"type:text" json:"failure_reason"`
	PaidAt         *time.Time             `json:"paid_at"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

func (RobokassaPayment) TableName() string { return "robokassa_payments" }
