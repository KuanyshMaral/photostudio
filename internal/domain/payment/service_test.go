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

type mockBookingWriter struct{ updateSystemCalls int }

func (m *mockBookingWriter) UpdatePaymentStatus(ctx context.Context, bookingID int64, status booking.PaymentStatus) (*booking.Booking, error) {
	return &booking.Booking{ID: bookingID, PaymentStatus: status}, nil
}
func (m *mockBookingWriter) UpdatePaymentStatusSystem(ctx context.Context, bookingID int64, status booking.PaymentStatus) (*booking.Booking, error) {
	m.updateSystemCalls++
	return m.UpdatePaymentStatus(ctx, bookingID, status)
}

type mockPaymentRepo struct {
	payment             *RobokassaPayment
	created             *RobokassaPayment
	updateStatusCalls   int
	markPaidCalls       int
	pendingUpdateCalled int
	markPaidChanged     *bool
}

func (m *mockPaymentRepo) Create(ctx context.Context, p *RobokassaPayment) error {
	m.created = p
	return nil
}
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
	if m.markPaidChanged != nil {
		return *m.markPaidChanged, nil
	}
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
	resp, err := svc.CreatePayment(context.Background(), 2, 12, "150.00", "x", nil, false, nil, nil)
	if err != nil || resp.InvID == 0 {
		t.Fatalf("unexpected %v", err)
	}
	if err := svc.FailPayment(context.Background(), resp.InvID); err != nil {
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
	if err != nil {
		t.Fatalf("expected idempotent OK, got %v", err)
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
	resp, err := svc.CreatePayment(context.Background(), 12, 13, "100.00", "x", nil, false, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(resp.PaymentURL, "ResultURL=") || !strings.Contains(resp.PaymentURL, "SuccessURL=") || !strings.Contains(resp.PaymentURL, "FailURL=") || !strings.Contains(resp.PaymentURL, "Encoding=utf-8") {
		t.Fatalf("expected callback URLs in payment URL: %s", resp.PaymentURL)
	}
}

func TestInitPayment_UsesCreatedStatusForLegacyPayment(t *testing.T) {
	repo := &mockPaymentRepo{}
	svc := &Service{
		payments:      repo,
		bookings:      &mockBookingReader{},
		bookingWriter: &mockBookingWriter{},
		merchantLogin: "merchant",
		password1:     "p1",
		loggerf:       func(string, ...interface{}) {},
		baseURL:       "https://auth.robokassa.ru/Merchant/Index.aspx",
		isTest:        "1",
	}

	resp, err := svc.InitPayment(context.Background(), InitPaymentRequest{
		BookingID:   77,
		OutSum:      "25000",
		Description: "invoice",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.created == nil {
		t.Fatal("expected payment to be persisted")
	}
	if repo.created.Status != PaymentStatusCreated {
		t.Fatalf("expected status %q, got %q", PaymentStatusCreated, repo.created.Status)
	}
	if resp.Status != string(PaymentStatusCreated) {
		t.Fatalf("expected response status %q, got %q", PaymentStatusCreated, resp.Status)
	}
}

func TestSignatureHashAlgorithmSHA256(t *testing.T) {
	t.Setenv("ROBOKASSA_HASH_ALGO", "sha256")
	t.Setenv("ROBOKASSA_IS_TEST", "1")
	t.Setenv("ROBOKASSA_TEST_PASSWORD_1", "p1")
	s := NewService(&mockPaymentRepo{}, &mockBookingReader{}, &mockBookingWriter{}, nil)
	s.merchantLogin = "m"
	if s.hashAlgo != "sha256" {
		t.Fatalf("expected hash algorithm sha256, got %s", s.hashAlgo)
	}
	got := s.generateSignatureForInit("100.00", 10, map[string]string{"k": "v"})
	if len(got) != 64 {
		t.Fatalf("expected SHA256 length 64, got %d", len(got))
	}
}

func TestLegacyInitPaymentURLIncludesConfiguredCallbackURLs(t *testing.T) {
	repo := &mockPaymentRepo{}
	svc := &Service{
		payments:      repo,
		bookings:      &mockBookingReader{},
		bookingWriter: &mockBookingWriter{},
		merchantLogin: "merchant",
		password1:     "p1",
		loggerf:       func(string, ...interface{}) {},
		baseURL:       "https://auth.robokassa.ru/Merchant/Index.aspx",
		isTest:        "1",
		resultURL:     "http://example.com/result",
		successURL:    "http://example.com/success",
		failURL:       "http://example.com/fail",
	}

	resp, err := svc.InitPayment(context.Background(), InitPaymentRequest{BookingID: 77, OutSum: "25000", Description: "invoice"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(resp.PaymentURL, "ResultURL=") || !strings.Contains(resp.PaymentURL, "SuccessURL=") || !strings.Contains(resp.PaymentURL, "FailURL=") || !strings.Contains(resp.PaymentURL, "Encoding=utf-8") {
		t.Fatalf("expected callback URLs + encoding in legacy payment URL: %s", resp.PaymentURL)
	}
}

func TestCreatePaymentURLIncludesPassedShpParams(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open("file:payment_shp?mode=memory&cache=private"), &gorm.Config{})
	_ = db.AutoMigrate(&auth.User{}, &booking.Booking{}, &Payment{}, &RecurringSubscription{}, &RobokassaPayment{})
	_ = db.Create(&auth.User{ID: 22, Email: "shp@u.kz"}).Error
	_ = db.Create(&booking.Booking{ID: 23, UserID: 22, TotalPrice: 100, RoomID: 1, StudioID: 1, StartTime: time.Now(), EndTime: time.Now().Add(time.Hour)}).Error
	bRepo := booking.NewBookingRepository(db)
	svc := NewService(NewRobokassaPaymentRepository(db), bRepo, bRepo, nil, NewRepository(db))

	resp, err := svc.CreatePayment(context.Background(), 22, 23, "100.00", "x", map[string]string{"custom": "v"}, false, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(resp.PaymentURL, "Shp_custom=v") {
		t.Fatalf("expected custom shp parameter in payment URL: %s", resp.PaymentURL)
	}
}

func TestHandleResultCallbackFailsOnMissingPassword2(t *testing.T) {
	svc := &Service{password2: ""}
	_, err := svc.HandleResultCallback(context.Background(), "100.00", 1, "sig", nil, "")
	if !errors.Is(err, ErrMisconfigured) {
		t.Fatalf("expected ErrMisconfigured, got %v", err)
	}
}

func TestNormalizeAmount(t *testing.T) {
	tests := []struct {
		in      string
		out     string
		wantErr bool
	}{
		{in: "25000", out: "25000.00"},
		{in: "025000.5", out: "25000.50"},
		{in: "0.01", out: "0.01"},
		{in: "-1", wantErr: true},
		{in: "1.999", wantErr: true},
		{in: "1,00", wantErr: true},
	}
	for _, tc := range tests {
		got, err := normalizeAmount(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("expected error for %q", tc.in)
			}
			continue
		}
		if err != nil || got != tc.out {
			t.Fatalf("normalizeAmount(%q)=%q err=%v, want %q", tc.in, got, err, tc.out)
		}
	}
}

func TestCreatePayment_PreventsShpOverrideOfSystemKeys(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open("file:payment_shp_override?mode=memory&cache=private"), &gorm.Config{})
	_ = db.AutoMigrate(&auth.User{}, &booking.Booking{}, &Payment{}, &RecurringSubscription{}, &RobokassaPayment{})
	_ = db.Create(&auth.User{ID: 32, Email: "secure@u.kz"}).Error
	_ = db.Create(&booking.Booking{ID: 33, UserID: 32, TotalPrice: 100, RoomID: 1, StudioID: 1, StartTime: time.Now(), EndTime: time.Now().Add(time.Hour)}).Error
	bRepo := booking.NewBookingRepository(db)
	svc := NewService(NewRobokassaPaymentRepository(db), bRepo, bRepo, nil, NewRepository(db))

	resp, err := svc.CreatePayment(context.Background(), 32, 33, "100.00", "x", map[string]string{"user_id": "999", "booking_id": "999", "custom": "ok"}, false, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(resp.PaymentURL, "Shp_user_id=999") || strings.Contains(resp.PaymentURL, "Shp_booking_id=999") {
		t.Fatalf("system shp params were overridden in URL: %s", resp.PaymentURL)
	}
	if !strings.Contains(resp.PaymentURL, "Shp_user_id=32") || !strings.Contains(resp.PaymentURL, "Shp_booking_id=33") {
		t.Fatalf("system shp params are missing in URL: %s", resp.PaymentURL)
	}
}

func TestHandleResultCallback_LegacyDuplicateDoesNotUpdateBooking(t *testing.T) {
	changed := false
	payRepo := &mockPaymentRepo{payment: &RobokassaPayment{InvID: 99, OutSum: "100.00", BookingID: 1}, markPaidChanged: &changed}
	writer := &mockBookingWriter{}
	svc := &Service{payments: payRepo, bookings: &mockBookingReader{}, bookingWriter: writer, loggerf: func(string, ...interface{}) {}, password2: "p2", password1: "p1", merchantLogin: "m"}
	sig := svc.generateSignatureForResult("100.00", 99, nil)
	if _, err := svc.HandleResultCallback(context.Background(), "100.00", 99, sig, nil, "raw"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if writer.updateSystemCalls != 0 {
		t.Fatalf("booking status must not be updated on duplicate callback")
	}
}
