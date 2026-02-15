package payment

import (
	"context"
	"photostudio/internal/domain/booking"
	"time"
)

type bookingReader interface {
	GetByID(ctx context.Context, id int64) (*booking.Booking, error)
}

type paymentRepo interface {
	Create(ctx context.Context, p *RobokassaPayment) error
	GetByInvID(ctx context.Context, invID int64) (*RobokassaPayment, error)
	UpdateStatus(ctx context.Context, invID int64, status RobokassaPaymentStatus, rawBody, reason string, paidAt *time.Time) error
	UpdateStatusPendingIfNotPaid(ctx context.Context, invID int64, rawBody string) error
	SaveSuccessRawBody(ctx context.Context, invID int64, rawBody string) error
	MarkPaidIdempotent(ctx context.Context, invID int64, rawBody string, paidAt time.Time) (bool, error)
}

type bookingPaymentWriter interface {
	UpdatePaymentStatus(ctx context.Context, bookingID int64, status booking.PaymentStatus) (*booking.Booking, error)
	UpdatePaymentStatusSystem(ctx context.Context, bookingID int64, status booking.PaymentStatus) (*booking.Booking, error)
}
