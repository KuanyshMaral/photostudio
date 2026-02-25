package payment

import (
	"context"
	"errors"
	"photostudio/internal/domain/auth"
	"photostudio/internal/domain/booking"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func mkDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(name), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&auth.User{}, &booking.Booking{}, &Payment{}, &RecurringSubscription{}, &RobokassaPayment{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestServiceInitPaymentLegacySuccess(t *testing.T) {
	db := mkDB(t, "file:init_legacy?mode=memory&cache=private")
	_ = db.Create(&booking.Booking{ID: 100, UserID: 7, TotalPrice: 11, RoomID: 1, StudioID: 1, StartTime: time.Now(), EndTime: time.Now().Add(time.Hour)}).Error
	b := booking.NewBookingRepository(db)
	s := &Service{payments: NewRobokassaPaymentRepository(db), bookings: b, bookingWriter: b, merchantLogin: "m", password1: "p1", password2: "p2", baseURL: "http://x", isTest: "1"}
	resp, err := s.InitPayment(context.Background(), InitPaymentRequest{BookingID: 100, OutSum: "11.00", Description: "d", ShpParams: map[string]string{"k": "v"}})
	if err != nil || resp.InvID == 0 {
		t.Fatalf("resp=%+v err=%v", resp, err)
	}
}

func TestServiceCreatePaymentOwnershipAndRepoNil(t *testing.T) {
	s := &Service{bookings: &mockBookingReader{bk: &booking.Booking{ID: 1, UserID: 2, TotalPrice: 10}}}
	if _, err := s.CreatePayment(context.Background(), 1, 1, "10.00", "", false, nil, nil); err == nil {
		t.Fatal("expected owner err")
	}
	if _, err := s.CreatePayment(context.Background(), 2, 1, "10.00", "", false, nil, nil); err == nil {
		t.Fatal("expected repo err")
	}
}

func TestServiceHandleResultV2Branches(t *testing.T) {
	db := mkDB(t, "file:result_v2?mode=memory&cache=private")
	repo := NewRepository(db)
	s := &Service{repo: repo, bookings: &mockBookingReader{}, bookingWriter: &mockBookingWriter{}, password2: "p2"}
	sig := s.generateSignatureForResult("1.00", 1, nil)
	if _, err := s.HandleResultCallback(context.Background(), "1.00", 1, sig, nil, ""); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("%v", err)
	}
	_ = repo.CreatePayment(context.Background(), &Payment{UserID: 1, RobokassaInvoiceID: 1, Amount: "2.00", Status: "created"})
	if _, err := s.HandleResultCallback(context.Background(), "1.00", 1, sig, nil, ""); !errors.Is(err, ErrAmountMismatch) {
		t.Fatalf("%v", err)
	}
}

func TestServiceHandleSuccessAndFailBranches(t *testing.T) {
	s := &Service{payments: &mockPaymentRepo{payment: &RobokassaPayment{InvID: 4, OutSum: "4.00"}}, password1: "p1"}
	if _, err := s.HandleSuccessCallback(context.Background(), "4.00", 4, "bad", nil, ""); !errors.Is(err, ErrInvalidSignature) {
		t.Fatal(err)
	}
	sig := s.generateSignatureForSuccess("3.00", 4, nil)
	if _, err := s.HandleSuccessCallback(context.Background(), "3.00", 4, sig, nil, ""); !errors.Is(err, ErrAmountMismatch) {
		t.Fatal(err)
	}
	failSig := s.generateSignatureForSuccess("4.00", 4, nil)
	if err := s.HandleFailCallback(context.Background(), "4.00", 4, failSig, nil); err != nil {
		t.Fatal(err)
	}
	if err := s.HandleFailCallback(context.Background(), "4.00", 4, "bad", nil); !errors.Is(err, ErrInvalidSignature) {
		t.Fatal(err)
	}
}

func TestAmountEqualSecondParseFail(t *testing.T) {
	if amountEqual("10", "bad") {
		t.Fatal("expected false")
	}
}
