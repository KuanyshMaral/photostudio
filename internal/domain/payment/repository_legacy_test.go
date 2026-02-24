package payment

import (
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupLegacyRepoDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:legacy_repo?mode=memory&cache=private"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&RobokassaPayment{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestLegacyRepositoryMethods(t *testing.T) {
	db := setupLegacyRepoDB(t)
	r := NewRobokassaPaymentRepository(db)
	ctx := context.Background()

	p := &RobokassaPayment{BookingID: 1, OutSum: "100.00", InvID: 101, Status: PaymentStatusCreated}
	if err := r.Create(ctx, p); err != nil {
		t.Fatal(err)
	}
	if _, err := r.GetByInvID(ctx, 101); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	if err := r.UpdateStatus(ctx, 101, PaymentStatusFailed, "raw", "bad", &now); err != nil {
		t.Fatal(err)
	}
	if err := r.SaveSuccessRawBody(ctx, 101, "ok"); err != nil {
		t.Fatal(err)
	}
	if err := r.UpdateStatusPendingIfNotPaid(ctx, 101, "pending"); err != nil {
		t.Fatal(err)
	}
	if changed, err := r.MarkPaidIdempotent(ctx, 101, "paid", now); err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	if changed, err := r.MarkPaidIdempotent(ctx, 101, "paid", now); err != nil || changed {
		t.Fatalf("idempotent changed=%v err=%v", changed, err)
	}
}

func TestLegacyRepositoryNotFoundBranches(t *testing.T) {
	db := setupLegacyRepoDB(t)
	r := NewRobokassaPaymentRepository(db)
	ctx := context.Background()
	if _, err := r.GetByInvID(ctx, 999); err == nil {
		t.Fatal("expected not found")
	}
	if err := r.UpdateStatusPendingIfNotPaid(ctx, 999, "raw"); err == nil {
		t.Fatal("expected not found")
	}
	if _, err := r.MarkPaidIdempotent(ctx, 999, "x", time.Now()); err == nil {
		t.Fatal("expected error")
	}
}
