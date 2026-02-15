package wallet

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"photostudio/internal/domain/auth"
)

const (
	TransactionTypeAdd    = "ADD"
	TransactionTypeSpend  = "SPEND"
	TransactionTypeRefund = "REFUND"
)

// FakeWallet stores user's fake balance.
type FakeWallet struct {
	ID      uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	UserID  int64     `json:"user_id" gorm:"not null;uniqueIndex;index"`
	Balance int64     `json:"balance" gorm:"not null;default:0"`

	User *auth.User `json:"user,omitempty" gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (FakeWallet) TableName() string {
	return "fake_wallets"
}

func (w *FakeWallet) BeforeCreate(_ *gorm.DB) error {
	if w.ID == uuid.Nil {
		w.ID = uuid.New()
	}
	return nil
}

// FakeTransaction records balance operations.
type FakeTransaction struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	WalletID  uuid.UUID `json:"wallet_id" gorm:"type:uuid;not null;index"`
	Amount    int64     `json:"amount" gorm:"not null"`
	Type      string    `json:"type" gorm:"type:varchar(16);not null;index;check:type IN ('ADD','SPEND','REFUND')"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`

	Wallet *FakeWallet `json:"wallet,omitempty" gorm:"foreignKey:WalletID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (FakeTransaction) TableName() string {
	return "fake_transactions"
}

func (t *FakeTransaction) BeforeCreate(_ *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}
