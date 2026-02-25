package payment

import (
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupRepoV2DB(t *testing.T) *Repository {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:repo_v2?mode=memory&cache=private"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Payment{}, &RecurringSubscription{}); err != nil {
		t.Fatal(err)
	}
	return NewRepository(db)
}

func TestRepoV2Branches(t *testing.T) {
	r := setupRepoV2DB(t)
	ctx := context.Background()
	if err := r.CreatePayment(ctx, &Payment{UserID: 1, RobokassaInvoiceID: 11, Amount: "1.00", Status: "created"}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.GetPaymentByInvoiceID(ctx, 11); err != nil {
		t.Fatal(err)
	}
	if _, err := r.GetPaymentByInvoiceID(ctx, 999); err == nil {
		t.Fatal("expected missing")
	}
	now := time.Now()
	changed, err := r.MarkPaymentStatus(ctx, 11, "paid", &now)
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	changed, err = r.MarkPaymentStatus(ctx, 11, "paid", &now)
	if err != nil || changed {
		t.Fatalf("idempotent changed=%v err=%v", changed, err)
	}
	if _, err := r.MarkPaymentStatus(ctx, 777, "paid", &now); err == nil {
		t.Fatal("expected err")
	}

	s := &RecurringSubscription{ID: "sub-1", UserID: 1, Status: "pending"}
	if err := r.CreateSubscription(ctx, s); err != nil {
		t.Fatal(err)
	}
	if _, err := r.GetSubscriptionByUserID(ctx, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := r.GetSubscriptionByUserID(ctx, 2); err == nil {
		t.Fatal("expected missing")
	}
	if err := r.UpdateSubscription(ctx, "sub-1", map[string]interface{}{"status": "active"}); err != nil {
		t.Fatal(err)
	}
}
