package payment

import (
	"context"
	"errors"
	"testing"
	"time"

	"photostudio/internal/domain"
)

type mockBookingReader struct{}

func (m *mockBookingReader) GetByID(ctx context.Context, id int64) (*domain.Booking, error) {
	return &domain.Booking{ID: id}, nil
}

type mockBookingWriter struct{}

func (m *mockBookingWriter) UpdatePaymentStatus(ctx context.Context, bookingID int64, status domain.PaymentStatus) (*domain.Booking, error) {
	return &domain.Booking{ID: bookingID, PaymentStatus: status}, nil
}

type mockPaymentRepo struct {
	payment             *domain.RobokassaPayment
	updateStatusCalls   int
	markPaidCalls       int
	pendingUpdateCalled int
}

func (m *mockPaymentRepo) Create(ctx context.Context, p *domain.RobokassaPayment) error { return nil }
func (m *mockPaymentRepo) GetByInvID(ctx context.Context, invID int64) (*domain.RobokassaPayment, error) {
	if m.payment == nil || m.payment.InvID != invID {
		return nil, errors.New("not found")
	}
	return m.payment, nil
}
func (m *mockPaymentRepo) UpdateStatus(ctx context.Context, invID int64, status domain.RobokassaPaymentStatus, rawBody, reason string, paidAt *time.Time) error {
	m.updateStatusCalls++
	return nil
}
func (m *mockPaymentRepo) UpdateStatusPendingIfNotPaid(ctx context.Context, invID int64, rawBody string) error {
	m.pendingUpdateCalled++
	return nil
}
func (m *mockPaymentRepo) SaveSuccessRawBody(ctx context.Context, invID int64, rawBody string) error {
	return nil
}
func (m *mockPaymentRepo) MarkPaidIdempotent(ctx context.Context, invID int64, rawBody string, paidAt time.Time) (bool, error) {
	m.markPaidCalls++
	return true, nil
}

func TestHandleResultCallback_AmountMismatch(t *testing.T) {
	repo := &mockPaymentRepo{payment: &domain.RobokassaPayment{InvID: 99, OutSum: "100.00", BookingID: 1}}
	svc := &Service{payments: repo, bookings: &mockBookingReader{}, bookingWriter: &mockBookingWriter{}, loggerf: func(string, ...interface{}) {}, password2: "p2", password1: "p1", merchantLogin: "m"}

	outSum := "50.00"
	sig := svc.generateSignatureForResult(outSum, 99, nil)
	_, err := svc.HandleResultCallback(context.Background(), outSum, 99, sig, nil, "raw")
	if !errors.Is(err, ErrAmountMismatch) {
		t.Fatalf("expected ErrAmountMismatch, got %v", err)
	}
	if repo.markPaidCalls != 0 {
		t.Fatalf("expected MarkPaidIdempotent not called")
	}
	if repo.updateStatusCalls == 0 {
		t.Fatalf("expected UpdateStatus called to mark failed")
	}
}

func TestHandleSuccessCallback_AmountMismatch(t *testing.T) {
	repo := &mockPaymentRepo{payment: &domain.RobokassaPayment{InvID: 77, OutSum: "300.00", BookingID: 1}}
	svc := &Service{payments: repo, bookings: &mockBookingReader{}, bookingWriter: &mockBookingWriter{}, loggerf: func(string, ...interface{}) {}, password2: "p2", password1: "p1", merchantLogin: "m"}

	outSum := "300"
	sig := svc.generateSignatureForSuccess(outSum, 77, nil)
	ok, err := svc.HandleSuccessCallback(context.Background(), outSum, 77, sig, nil, "raw")
	if err != nil || !ok {
		t.Fatalf("expected success for equivalent numeric values, got ok=%v err=%v", ok, err)
	}

	badSig := svc.generateSignatureForSuccess("100.00", 77, nil)
	ok, err = svc.HandleSuccessCallback(context.Background(), "100.00", 77, badSig, nil, "raw")
	if !errors.Is(err, ErrAmountMismatch) || ok {
		t.Fatalf("expected amount mismatch, got ok=%v err=%v", ok, err)
	}
}


