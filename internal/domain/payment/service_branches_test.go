package payment

import (
	"context"
	"errors"
	"photostudio/internal/domain/booking"
	"testing"
	"time"
)

type errBookingReader struct{ err error }

func (e *errBookingReader) GetByID(ctx context.Context, id int64) (*booking.Booking, error) {
	return nil, e.err
}

type errPaymentRepo struct{}

func (e *errPaymentRepo) Create(ctx context.Context, p *RobokassaPayment) error {
	return errors.New("create")
}
func (e *errPaymentRepo) GetByInvID(ctx context.Context, invID int64) (*RobokassaPayment, error) {
	return nil, errors.New("get")
}
func (e *errPaymentRepo) UpdateStatus(ctx context.Context, invID int64, status RobokassaPaymentStatus, rawBody, reason string, paidAt *time.Time) error {
	return nil
}
func (e *errPaymentRepo) UpdateStatusPendingIfNotPaid(ctx context.Context, invID int64, rawBody string) error {
	return errors.New("pending")
}
func (e *errPaymentRepo) SaveSuccessRawBody(ctx context.Context, invID int64, rawBody string) error {
	return nil
}
func (e *errPaymentRepo) MarkPaidIdempotent(ctx context.Context, invID int64, rawBody string, paidAt time.Time) (bool, error) {
	return false, errors.New("mark")
}

type errBookingWriter struct{}

func (e *errBookingWriter) UpdatePaymentStatus(ctx context.Context, bookingID int64, status booking.PaymentStatus) (*booking.Booking, error) {
	return nil, errors.New("u")
}
func (e *errBookingWriter) UpdatePaymentStatusSystem(ctx context.Context, bookingID int64, status booking.PaymentStatus) (*booking.Booking, error) {
	return nil, errors.New("u")
}

func TestServiceErrorBranches(t *testing.T) {
	s := &Service{payments: &errPaymentRepo{}, bookings: &errBookingReader{err: errors.New("bk")}, bookingWriter: &errBookingWriter{}, merchantLogin: "m", password1: "p1", password2: "p2", isTest: "1", baseURL: "http://x"}
	if _, err := s.CreatePayment(context.Background(), 1, 2, "1.00", "x", false, nil, nil); err == nil {
		t.Fatal("expected err")
	}
	if _, err := s.CreateSubscription(context.Background(), 1, "1.00"); err == nil {
		t.Fatal("expected repo nil error")
	}
	if _, err := s.InitPayment(context.Background(), InitPaymentRequest{BookingID: 1, OutSum: "1.00"}); err == nil {
		t.Fatal("expected init err")
	}
	if _, err := s.HandleResultCallback(context.Background(), "1", 1, "bad", nil, ""); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("%v", err)
	}
}

func TestServiceLegacyBranches(t *testing.T) {
	repo := &mockPaymentRepo{payment: &RobokassaPayment{InvID: 7, OutSum: "10.00", BookingID: 3}}
	s := &Service{payments: repo, bookings: &mockBookingReader{}, bookingWriter: &mockBookingWriter{}, merchantLogin: "m", password1: "p1", password2: "p2", isTest: "1", baseURL: "http://x"}
	sig := s.generateSignatureForResult("10.00", 7, nil)
	ack, err := s.HandleResultCallback(context.Background(), "10.00", 7, sig, nil, "raw")
	if err != nil || ack != "OK7" {
		t.Fatalf("ack=%s err=%v", ack, err)
	}

	ok, err := s.HandleSuccessCallback(context.Background(), "10.00", 7, s.generateSignatureForSuccess("10.00", 7, nil), nil, "raw")
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	_, err = s.HandleSuccessCallback(context.Background(), "10.00", 7, "bad", nil, "raw")
	if !errors.Is(err, ErrInvalidSignature) {
		t.Fatal(err)
	}

	s2 := &Service{payments: &errPaymentRepo{}, bookings: &mockBookingReader{}, bookingWriter: &mockBookingWriter{}, merchantLogin: "m", password1: "p1", password2: "p2"}
	_, err = s2.HandleSuccessCallback(context.Background(), "1", 1, s2.generateSignatureForSuccess("1", 1, nil), nil, "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAmountEqualAndEnvDefault(t *testing.T) {
	if !amountEqual("10.0", "10.00") {
		t.Fatal("amount equal")
	}
	if amountEqual("xx", "10") {
		t.Fatal("bad parse should fail")
	}
	if v := envOrDefault("__NO_SUCH_ENV__", "d"); v != "d" {
		t.Fatal(v)
	}
}
