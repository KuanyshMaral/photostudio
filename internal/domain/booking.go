package domain

import "time"

type BookingStatus string

const (
	BookingPending   BookingStatus = "pending"
	BookingConfirmed BookingStatus = "confirmed"
	BookingCancelled BookingStatus = "cancelled"
	BookingCompleted BookingStatus = "completed"
)

type PaymentStatus string

const (
	PaymentUnpaid   PaymentStatus = "unpaid"
	PaymentPaid     PaymentStatus = "paid"
	PaymentRefunded PaymentStatus = "refunded"
)

type Booking struct {
	ID            int64         `json:"id"`
	RoomID        int64         `json:"room_id" validate:"required"`
	StudioID      int64         `json:"studio_id" validate:"required"`
	UserID        int64         `json:"user_id" validate:"required"`
	StartTime     time.Time     `json:"start_time" validate:"required"`
	EndTime       time.Time     `json:"end_time" validate:"required"`
	TotalPrice    float64       `json:"total_price" validate:"required,gte=0"`
	Status        BookingStatus `json:"status"`
	PaymentStatus PaymentStatus `json:"payment_status"`
	Notes         string        `json:"notes,omitempty" gorm:"type:text"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
	CancelledAt   *time.Time    `json:"cancelled_at,omitempty"`

	// Block 9: Причина отмены (заполняется при cancel)
	CancellationReason string `json:"cancellation_reason,omitempty" gorm:"type:text"`

	// Block 10: Предоплата (для менеджеров)
	DepositAmount float64 `json:"deposit_amount,omitempty"`

	// Связи
	User *User `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Room *Room `json:"room,omitempty" gorm:"foreignKey:RoomID"`
}
