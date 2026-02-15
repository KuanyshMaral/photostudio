package wallet

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestService(t *testing.T) *Service {
	t.Helper()
	dsn := fmt.Sprintf("file:wallet_test_%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&FakeWallet{}, &FakeTransaction{}); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}
	return NewService(db)
}

func TestGetOrCreateWalletCreatesOnFirstRequest(t *testing.T) {
	svc := setupTestService(t)

	wallet, err := svc.GetOrCreateWallet(context.Background(), 1001)
	if err != nil {
		t.Fatalf("GetOrCreateWallet returned error: %v", err)
	}
	if wallet.Balance != 0 {
		t.Fatalf("expected zero initial balance, got %d", wallet.Balance)
	}

	again, err := svc.GetOrCreateWallet(context.Background(), 1001)
	if err != nil {
		t.Fatalf("GetOrCreateWallet second call returned error: %v", err)
	}
	if wallet.ID != again.ID {
		t.Fatalf("expected same wallet id, got %s and %s", wallet.ID, again.ID)
	}
}

func TestAddAndSpendFlow(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()

	wallet, addTxn, err := svc.Add(ctx, 101, 150)
	if err != nil {
		t.Fatalf("Add returned error: %v", err)
	}
	if wallet.Balance != 150 {
		t.Fatalf("expected balance 150, got %d", wallet.Balance)
	}
	if addTxn.Type != TransactionTypeAdd {
		t.Fatalf("expected txn type %s, got %s", TransactionTypeAdd, addTxn.Type)
	}

	wallet, spendTxn, err := svc.Spend(ctx, 101, 40)
	if err != nil {
		t.Fatalf("Spend returned error: %v", err)
	}
	if wallet.Balance != 110 {
		t.Fatalf("expected balance 110, got %d", wallet.Balance)
	}
	if spendTxn.Type != TransactionTypeSpend {
		t.Fatalf("expected txn type %s, got %s", TransactionTypeSpend, spendTxn.Type)
	}

	txns, err := svc.ListTransactions(ctx, 101)
	if err != nil {
		t.Fatalf("ListTransactions returned error: %v", err)
	}
	if len(txns) != 2 {
		t.Fatalf("expected 2 transactions, got %d", len(txns))
	}
}

func TestListTransactionsCreatesEmptyWallet(t *testing.T) {
	svc := setupTestService(t)

	txns, err := svc.ListTransactions(context.Background(), 999)
	if err != nil {
		t.Fatalf("ListTransactions returned error: %v", err)
	}
	if len(txns) != 0 {
		t.Fatalf("expected 0 transactions, got %d", len(txns))
	}
}

func TestAddRejectsNonPositiveAmount(t *testing.T) {
	svc := setupTestService(t)
	_, _, err := svc.Add(context.Background(), 102, 0)
	if !errors.Is(err, ErrInvalidAmount) {
		t.Fatalf("expected ErrInvalidAmount, got %v", err)
	}
}

func TestSpendRejectsNonPositiveAmount(t *testing.T) {
	svc := setupTestService(t)
	_, _, err := svc.Spend(context.Background(), 103, -1)
	if !errors.Is(err, ErrInvalidAmount) {
		t.Fatalf("expected ErrInvalidAmount, got %v", err)
	}
}

func TestSpendInsufficientFunds(t *testing.T) {
	svc := setupTestService(t)
	_, _, err := svc.Spend(context.Background(), 104, 10)
	if !errors.Is(err, ErrInsufficientFunds) {
		t.Fatalf("expected ErrInsufficientFunds, got %v", err)
	}
}
