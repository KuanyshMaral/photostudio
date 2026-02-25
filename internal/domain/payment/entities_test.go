package payment

import "testing"

func TestTableNames(t *testing.T) {
	if (Payment{}).TableName() != "payments" {
		t.Fatal("bad")
	}
	if (RecurringSubscription{}).TableName() != "recurring_subscriptions" {
		t.Fatal("bad")
	}
	if (RobokassaPayment{}).TableName() != "robokassa_payments" {
		t.Fatal("bad")
	}
}
