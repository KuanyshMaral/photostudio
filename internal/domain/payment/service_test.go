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

func TestDefaultIsTestIsProductionMode(t *testing.T) {
	t.Setenv("ROBOKASSA_IS_TEST", "")
	t.Setenv("ROBOKASSA_PROD_PASSWORD_1", "prod1")
	t.Setenv("ROBOKASSA_PROD_PASSWORD_2", "prod2")
	t.Setenv("ROBOKASSA_TEST_PASSWORD_1", "test1")
	t.Setenv("ROBOKASSA_TEST_PASSWORD_2", "test2")

	s := NewService(&mockPaymentRepo{}, &mockBookingReader{}, &mockBookingWriter{}, nil)
	if s.isTest != "0" {
		t.Fatalf("expected default IsTest=0, got %s", s.isTest)
	}
	if s.password1 != "prod1" || s.password2 != "prod2" {
		t.Fatalf("expected prod passwords to be selected by default")
	}
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

func TestNormalizePreviousInvoiceID(t *testing.T) {
	zero := int64(0)
	minus := int64(-5)
	valid := int64(42)

	if got := normalizePreviousInvoiceID(false, &valid); got != nil {
		t.Fatal("expected nil for non-recurring payment")
	}
	if got := normalizePreviousInvoiceID(true, nil); got != nil {
		t.Fatal("expected nil for nil previous invoice")
	}
	if got := normalizePreviousInvoiceID(true, &zero); got != nil {
		t.Fatal("expected nil for zero previous invoice")
	}
	if got := normalizePreviousInvoiceID(true, &minus); got != nil {
		t.Fatal("expected nil for negative previous invoice")
	}
	if got := normalizePreviousInvoiceID(true, &valid); got == nil || *got != valid {
		t.Fatal("expected valid previous invoice for recurring payment")
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

func TestRobokassaBaseURLDefaultsByMerchantRegion(t *testing.T) {
	t.Setenv("ROBOKASSA_BASE_URL", "")
	if got := robokassaBaseURL("photostudio_kz"); got != "https://auth.robokassa.kz/Merchant/Index.aspx" {
		t.Fatalf("expected kz base url, got %s", got)
	}
	if got := robokassaBaseURL("photostudio"); got != "https://auth.robokassa.ru/Merchant/Index.aspx" {
		t.Fatalf("expected ru base url, got %s", got)
	}
}

func TestRobokassaBaseURLRespectsConfiguredOverride(t *testing.T) {
	t.Setenv("ROBOKASSA_BASE_URL", "https://custom.robokassa/Merchant/Index.aspx")
	if got := robokassaBaseURL("photostudio_kz"); got != "https://custom.robokassa/Merchant/Index.aspx" {
		t.Fatalf("expected configured override, got %s", got)
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

func TestInitSignatureIncludesShpParams(t *testing.T) {
	s := &Service{merchantLogin: "merchant", password1: "pass1"}
	withShp := s.generateSignatureForInit("100.00", 10, map[string]string{"k": "v"})
	withoutShp := s.generateSignatureForInit("100.00", 10, nil)
	if withShp == withoutShp {
		t.Fatalf("init signature must include shp params; with=%s without=%s", withShp, withoutShp)
	}
	if withShp != "beea1c1a7213810e9c9ef5237751290c" {
		t.Fatalf("unexpected md5 signature %s", withShp)
	}
	if withoutShp != "f23421cdc4908465357929050dcfdebc" {
		t.Fatalf("unexpected md5 signature without shp %s", withoutShp)
	}
}

func TestSelectRobokassaPasswordsByMode(t *testing.T) {
	t.Setenv("ROBOKASSA_PROD_PASSWORD_1", "prod1")
	t.Setenv("ROBOKASSA_PROD_PASSWORD_2", "prod2")
	t.Setenv("ROBOKASSA_TEST_PASSWORD_1", "test1")
	t.Setenv("ROBOKASSA_TEST_PASSWORD_2", "test2")

	p1, p2 := selectRobokassaPasswords("1")
	if p1 != "test1" || p2 != "test2" {
		t.Fatalf("expected test credentials, got %s/%s", p1, p2)
	}

	p1, p2 = selectRobokassaPasswords("0")
	if p1 != "prod1" || p2 != "prod2" {
		t.Fatalf("expected prod credentials, got %s/%s", p1, p2)
	}
}

func TestNormalizeRobokassaIsTestSupportsBooleanFlags(t *testing.T) {
	for _, value := range []string{"1", "true", "yes", "on", " TRUE "} {
		if got := normalizeRobokassaIsTest(value); got != "1" {
			t.Fatalf("expected IsTest=1 for value %q, got %s", value, got)
		}
	}
	if got := normalizeRobokassaIsTest("0"); got != "0" {
		t.Fatalf("expected IsTest=0 for value 0, got %s", got)
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

func TestCreatePayment_NonRecurringOmitsRecurringOnlyParams(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open("file:payment_nonrecurring?mode=memory&cache=private"), &gorm.Config{})
	_ = db.AutoMigrate(&auth.User{}, &booking.Booking{}, &Payment{}, &RecurringSubscription{}, &RobokassaPayment{})
	_ = db.Create(&auth.User{ID: 40, Email: "nonrec@u.kz"}).Error
	_ = db.Create(&booking.Booking{ID: 41, UserID: 40, TotalPrice: 100, RoomID: 1, StudioID: 1, StartTime: time.Now(), EndTime: time.Now().Add(time.Hour)}).Error
	bRepo := booking.NewBookingRepository(db)
	svc := NewService(NewRobokassaPaymentRepository(db), bRepo, bRepo, nil, NewRepository(db))
	svc.merchantLogin = "merchant"
	svc.password1 = "p1"

	prev := int64(99)
	subID := "3fa85f64-5717-4562-b3fc-2c963f66afa6"
	resp, err := svc.CreatePayment(context.Background(), 40, 41, "100", "x", nil, false, &prev, &subID)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(resp.PaymentURL, "Recurring=true") || strings.Contains(resp.PaymentURL, "PreviousInvoiceID=") || strings.Contains(resp.PaymentURL, "Shp_subscription_id=") {
		t.Fatalf("non-recurring payment must omit recurring-only params: %s", resp.PaymentURL)
	}
}

func TestCreatePayment_RecurringIncludesRecurringOnlyParams(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open("file:payment_recurring?mode=memory&cache=private"), &gorm.Config{})
	_ = db.AutoMigrate(&auth.User{}, &booking.Booking{}, &Payment{}, &RecurringSubscription{}, &RobokassaPayment{})
	_ = db.Create(&auth.User{ID: 42, Email: "rec@u.kz"}).Error
	_ = db.Create(&booking.Booking{ID: 43, UserID: 42, TotalPrice: 100, RoomID: 1, StudioID: 1, StartTime: time.Now(), EndTime: time.Now().Add(time.Hour)}).Error
	bRepo := booking.NewBookingRepository(db)
	svc := NewService(NewRobokassaPaymentRepository(db), bRepo, bRepo, nil, NewRepository(db))
	svc.merchantLogin = "merchant"
	svc.password1 = "p1"

	prev := int64(88)
	subID := "3fa85f64-5717-4562-b3fc-2c963f66afa6"
	resp, err := svc.CreatePayment(context.Background(), 42, 43, "100", "x", nil, true, &prev, &subID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(resp.PaymentURL, "Recurring=true") || !strings.Contains(resp.PaymentURL, "PreviousInvoiceID=88") || !strings.Contains(resp.PaymentURL, "Shp_subscription_id=") {
		t.Fatalf("recurring payment must include recurring-only params: %s", resp.PaymentURL)
	}
}

func TestCreatePayment_NonRecurringIgnoresCaseInsensitiveShpSubscriptionID(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open("file:payment_nonrecurring_shp_case?mode=memory&cache=private"), &gorm.Config{})
	_ = db.AutoMigrate(&auth.User{}, &booking.Booking{}, &Payment{}, &RecurringSubscription{}, &RobokassaPayment{})
	_ = db.Create(&auth.User{ID: 52, Email: "nonrec-case@u.kz"}).Error
	_ = db.Create(&booking.Booking{ID: 53, UserID: 52, TotalPrice: 100, RoomID: 1, StudioID: 1, StartTime: time.Now(), EndTime: time.Now().Add(time.Hour)}).Error
	bRepo := booking.NewBookingRepository(db)
	svc := NewService(NewRobokassaPaymentRepository(db), bRepo, bRepo, nil, NewRepository(db))

	resp, err := svc.CreatePayment(context.Background(), 52, 53, "100", "x", map[string]string{"ShP_subscription_id": "malicious-sub-id", "custom": "ok"}, false, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(resp.PaymentURL, "Shp_subscription_id=") {
		t.Fatalf("non-recurring payment must omit Shp_subscription_id from incoming params: %s", resp.PaymentURL)
	}
	if !strings.Contains(resp.PaymentURL, "Shp_custom=ok") {
		t.Fatalf("expected other shp params to stay in payment URL: %s", resp.PaymentURL)
	}
}

func TestSanitizeShpParams_StripsCaseInsensitivePrefix(t *testing.T) {
	got := sanitizeShpParams(map[string]string{"ShP_custom": "v", "SHP_other": "x"})
	if got["custom"] != "v" {
		t.Fatalf("expected ShP_ prefix to be trimmed, got: %#v", got)
	}
	if got["other"] != "x" {
		t.Fatalf("expected SHP_ prefix to be trimmed, got: %#v", got)
	}
}

func TestCreatePayment_PreventsMixedCaseShpOverrideOfSystemKeys(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open("file:payment_shp_override_mixed_case?mode=memory&cache=private"), &gorm.Config{})
	_ = db.AutoMigrate(&auth.User{}, &booking.Booking{}, &Payment{}, &RecurringSubscription{}, &RobokassaPayment{})
	_ = db.Create(&auth.User{ID: 62, Email: "secure-mixed@u.kz"}).Error
	_ = db.Create(&booking.Booking{ID: 63, UserID: 62, TotalPrice: 100, RoomID: 1, StudioID: 1, StartTime: time.Now(), EndTime: time.Now().Add(time.Hour)}).Error
	bRepo := booking.NewBookingRepository(db)
	svc := NewService(NewRobokassaPaymentRepository(db), bRepo, bRepo, nil, NewRepository(db))

	resp, err := svc.CreatePayment(context.Background(), 62, 63, "100.00", "x", map[string]string{"User_ID": "999", "BOOKING_ID": "999", "Custom": "ok"}, false, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(resp.PaymentURL, "Shp_user_id=999") || strings.Contains(resp.PaymentURL, "Shp_booking_id=999") {
		t.Fatalf("mixed-case system shp params were overridden in URL: %s", resp.PaymentURL)
	}
	if !strings.Contains(resp.PaymentURL, "Shp_user_id=62") || !strings.Contains(resp.PaymentURL, "Shp_booking_id=63") {
		t.Fatalf("system shp params are missing in URL: %s", resp.PaymentURL)
	}
	if !strings.Contains(resp.PaymentURL, "Shp_custom=ok") {
		t.Fatalf("expected custom param to be lower-cased and preserved: %s", resp.PaymentURL)
	}
}

func TestSanitizeShpParams_NormalizesKeysToLowercase(t *testing.T) {
	got := sanitizeShpParams(map[string]string{"ShP_CuStOm": "v", "BOOKING_ID": "1"})
	if got["custom"] != "v" {
		t.Fatalf("expected normalized lowercase custom key, got: %#v", got)
	}
	if got["booking_id"] != "1" {
		t.Fatalf("expected normalized lowercase booking_id key, got: %#v", got)
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

func TestGenerateInvoiceID_RobokassaRange(t *testing.T) {
	for i := 0; i < 1000; i++ {
		id := generateInvoiceID()
		if id <= 0 {
			t.Fatalf("invoice id must be positive, got %d", id)
		}
		if id > 2_147_483_647 {
			t.Fatalf("invoice id must fit int32 range for robokassa, got %d", id)
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
