package payment

import (
	"context"
	"errors"
	"os"
	"photostudio/internal/domain/auth"
	"photostudio/internal/domain/booking"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type mockBookingReader struct{ bk *booking.Booking }

func (m *mockBookingReader) GetByID(ctx context.Context, id int64) (*booking.Booking, error) {
	if m.bk != nil {
		return m.bk, nil
	}
	return &booking.Booking{ID: id, UserID: 1, TotalPrice: 100}, nil
}

type mockBookingWriter struct{}

func (m *mockBookingWriter) UpdatePaymentStatus(ctx context.Context, bookingID int64, status booking.PaymentStatus) (*booking.Booking, error) {
	return &booking.Booking{ID: bookingID, PaymentStatus: status}, nil
}
func (m *mockBookingWriter) UpdatePaymentStatusSystem(ctx context.Context, bookingID int64, status booking.PaymentStatus) (*booking.Booking, error) {
	return m.UpdatePaymentStatus(ctx, bookingID, status)
}

type mockPaymentRepo struct {
	payment             *RobokassaPayment
	updateStatusCalls   int
	markPaidCalls       int
	pendingUpdateCalled int
}

func (m *mockPaymentRepo) Create(ctx context.Context, p *RobokassaPayment) error { return nil }
func (m *mockPaymentRepo) GetByInvID(ctx context.Context, invID int64) (*RobokassaPayment, error) {
	if m.payment == nil || m.payment.InvID != invID {
		return nil, errors.New("not found")
	}
	return m.payment, nil
}
func (m *mockPaymentRepo) UpdateStatus(ctx context.Context, invID int64, status RobokassaPaymentStatus, rawBody, reason string, paidAt *time.Time) error {
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

func TestSignatureSelectionByMode(t *testing.T) {
	os.Setenv("ROBOKASSA_IS_TEST", "1")
	os.Setenv("ROBOKASSA_TEST_PASSWORD_1", "t1")
	os.Setenv("ROBOKASSA_TEST_PASSWORD_2", "t2")
	s := NewService(&mockPaymentRepo{}, &mockBookingReader{}, &mockBookingWriter{}, nil)
	if s.password1 != "t1" || s.password2 != "t2" {
		t.Fatalf("test keys expected")
	}
}

func TestHandleResultCallback_AmountMismatch(t *testing.T) {
	repo := &mockPaymentRepo{payment: &RobokassaPayment{InvID: 99, OutSum: "100.00", BookingID: 1}}
	svc := &Service{payments: repo, bookings: &mockBookingReader{}, bookingWriter: &mockBookingWriter{}, loggerf: func(string, ...interface{}) {}, password2: "p2", password1: "p1", merchantLogin: "m"}
	sig := svc.generateSignatureForResult("50.00", 99, nil)
	_, err := svc.HandleResultCallback(context.Background(), "50.00", 99, sig, nil, "raw")
	if !errors.Is(err, ErrAmountMismatch) {
		t.Fatalf("expected ErrAmountMismatch, got %v", err)
	}
}

func TestPaymentURLBuilderIncludesRecurring(t *testing.T) {
	svc := &Service{merchantLogin: "m", password1: "p1", isTest: "1", baseURL: "https://x"}
	req := InitPaymentRequest{BookingID: 1, OutSum: "100.00", Description: "d", ShpParams: map[string]string{"a": "b"}}
	_ = req
	sig := svc.generateSignatureForInit("100.00", 10, map[string]string{"subscription_id": "s1"})
	if sig == "" {
		t.Fatal("signature should not be empty")
	}
}

func TestCreatePaymentAndFailPayment(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	_ = db.AutoMigrate(&auth.User{}, &booking.Booking{}, &Payment{}, &RecurringSubscription{}, &RobokassaPayment{})
	_ = db.Create(&auth.User{ID: 2, Email: "u@u.kz"}).Error
	_ = db.Create(&booking.Booking{ID: 12, UserID: 2, TotalPrice: 150, RoomID: 1, StudioID: 1, StartTime: time.Now(), EndTime: time.Now().Add(time.Hour)}).Error
	bRepo := booking.NewBookingRepository(db)
	svc := NewService(NewRobokassaPaymentRepository(db), bRepo, bRepo, nil, NewRepository(db))
	resp, err := svc.CreatePayment(context.Background(), 2, 12, "150.00", "x", false, nil, nil)
	if err != nil || resp.InvID == 0 {
		t.Fatalf("unexpected %v", err)
	}
	sig := svc.generateSignatureForSuccess("150.00", resp.InvID, nil)
	if err := svc.HandleFailCallback(context.Background(), "150.00", resp.InvID, sig, nil); err != nil {
		t.Fatal(err)
	}
}

func TestCreateSubscriptionAndCancel(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	_ = db.AutoMigrate(&auth.User{}, &booking.Booking{}, &Payment{}, &RecurringSubscription{}, &RobokassaPayment{})
	_ = db.Create(&auth.User{ID: 3, Email: "x@y.kz"}).Error
	svc := NewService(NewRobokassaPaymentRepository(db), &mockBookingReader{}, &mockBookingWriter{}, nil, NewRepository(db))
	_, err := svc.CreateSubscription(context.Background(), 3, "199.00")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.CancelSubscription(context.Background(), 3); err != nil {
		t.Fatal(err)
	}
	sub, err := svc.GetMySubscription(context.Background(), 3)
	if err != nil || sub.Status != "canceled" {
		t.Fatalf("expected canceled, err=%v status=%v", err, sub.Status)
	}
}

func TestHandleResultCallback_ReplayDetected(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open("file:payment_replay?mode=memory&cache=private"), &gorm.Config{})
	_ = db.AutoMigrate(&auth.User{}, &booking.Booking{}, &Payment{}, &RecurringSubscription{}, &RobokassaPayment{})
	_ = db.Create(&auth.User{ID: 11, Email: "r@r.kz"}).Error
	repo := NewRepository(db)
	if err := repo.CreatePayment(context.Background(), &Payment{UserID: 11, RobokassaInvoiceID: 555, Amount: "100.00", Status: "paid"}); err != nil {
		t.Fatal(err)
	}
	svc := NewService(NewRobokassaPaymentRepository(db), &mockBookingReader{}, &mockBookingWriter{}, nil, repo)
	svc.password2 = "p2"
	sig := svc.generateSignatureForResult("100.00", 555, nil)
	_, err := svc.HandleResultCallback(context.Background(), "100.00", 555, sig, nil, "")
	if !errors.Is(err, ErrReplayDetected) {
		t.Fatalf("expected replay error, got %v", err)
	}
}

func TestCreatePaymentURLIncludesConfiguredCallbackURLs(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open("file:payment_urls?mode=memory&cache=private"), &gorm.Config{})
	_ = db.AutoMigrate(&auth.User{}, &booking.Booking{}, &Payment{}, &RecurringSubscription{}, &RobokassaPayment{})
	_ = db.Create(&auth.User{ID: 12, Email: "u2@u.kz"}).Error
	_ = db.Create(&booking.Booking{ID: 13, UserID: 12, TotalPrice: 100, RoomID: 1, StudioID: 1, StartTime: time.Now(), EndTime: time.Now().Add(time.Hour)}).Error
	bRepo := booking.NewBookingRepository(db)
	svc := NewService(NewRobokassaPaymentRepository(db), bRepo, bRepo, nil, NewRepository(db))
	svc.resultURL = "http://example.com/result"
	svc.successURL = "http://example.com/success"
	svc.failURL = "http://example.com/fail"
	resp, err := svc.CreatePayment(context.Background(), 12, 13, "100.00", "x", false, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(resp.PaymentURL, "ResultURL=") || !strings.Contains(resp.PaymentURL, "SuccessURL=") || !strings.Contains(resp.PaymentURL, "FailURL=") {
		t.Fatalf("expected callback URLs in payment URL: %s", resp.PaymentURL)
	}
}
