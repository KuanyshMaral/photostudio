package wallet

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrInvalidAmount     = errors.New("amount must be positive")
	ErrInsufficientFunds = errors.New("insufficient balance")
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) GetOrCreateWallet(ctx context.Context, userID int64) (*FakeWallet, error) {
	wallet, err := s.getWalletByUserID(ctx, userID)
	if err == nil {
		return wallet, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	wallet = &FakeWallet{UserID: userID, Balance: 0}
	if err := s.db.WithContext(ctx).Create(wallet).Error; err != nil {
		if isUniqueConstraintError(err) {
			return s.getWalletByUserID(ctx, userID)
		}
		return nil, err
	}
	return wallet, nil
}

func (s *Service) Add(ctx context.Context, userID int64, amount int64) (*FakeWallet, *FakeTransaction, error) {
	if amount <= 0 {
		return nil, nil, ErrInvalidAmount
	}

	var wallet FakeWallet
	var txn FakeTransaction

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := getOrCreateWalletForUpdate(tx, userID, &wallet); err != nil {
			return err
		}

		wallet.Balance += amount
		if err := tx.Model(&FakeWallet{}).Where("id = ?", wallet.ID).Update("balance", wallet.Balance).Error; err != nil {
			return err
		}

		txn = FakeTransaction{WalletID: wallet.ID, Amount: amount, Type: TransactionTypeAdd}
		if err := tx.Create(&txn).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return &wallet, &txn, nil
}

func (s *Service) Spend(ctx context.Context, userID int64, amount int64) (*FakeWallet, *FakeTransaction, error) {
	if amount <= 0 {
		return nil, nil, ErrInvalidAmount
	}

	var wallet FakeWallet
	var txn FakeTransaction

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := getOrCreateWalletForUpdate(tx, userID, &wallet); err != nil {
			return err
		}

		if wallet.Balance < amount {
			return ErrInsufficientFunds
		}

		wallet.Balance -= amount
		if err := tx.Model(&FakeWallet{}).Where("id = ?", wallet.ID).Update("balance", wallet.Balance).Error; err != nil {
			return err
		}

		txn = FakeTransaction{WalletID: wallet.ID, Amount: amount, Type: TransactionTypeSpend}
		if err := tx.Create(&txn).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return &wallet, &txn, nil
}

func (s *Service) ListTransactions(ctx context.Context, userID int64) ([]FakeTransaction, error) {
	wallet, err := s.GetOrCreateWallet(ctx, userID)
	if err != nil {
		return nil, err
	}

	var txns []FakeTransaction
	if err := s.db.WithContext(ctx).Where("wallet_id = ?", wallet.ID).Order("created_at desc").Find(&txns).Error; err != nil {
		return nil, err
	}

	return txns, nil
}

func (s *Service) getWalletByUserID(ctx context.Context, userID int64) (*FakeWallet, error) {
	var wallet FakeWallet
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		return nil, err
	}
	return &wallet, nil
}

func getOrCreateWalletForUpdate(tx *gorm.DB, userID int64, wallet *FakeWallet) error {
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).First(wallet).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		*wallet = FakeWallet{UserID: userID, Balance: 0}
		if err := tx.Create(wallet).Error; err != nil {
			if isUniqueConstraintError(err) {
				return tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).First(wallet).Error
			}
			return err
		}
	}
	return nil
}

func isUniqueConstraintError(err error) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique constraint") || strings.Contains(msg, "unique failed")
}
