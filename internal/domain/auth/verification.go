package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"log"
	"math/big"
	"regexp"
	"strings"
	"time"
)

var codeRegex = regexp.MustCompile(`^\d{6}$`)

type Mailer interface {
	SendVerificationCode(ctx context.Context, email, code string) error
}

type DevConsoleMailer struct {
	enabled bool
}

func NewDevConsoleMailer(enabled bool) *DevConsoleMailer {
	return &DevConsoleMailer{enabled: enabled}
}

func (m *DevConsoleMailer) SendVerificationCode(_ context.Context, email, code string) error {
	if m.enabled {
		log.Printf("[DEV-EMAIL] verification code email=%s code=%s", email, code)
	}
	return nil
}

type verificationCodeRow struct {
	UserID      int64      `gorm:"column:user_id"`
	CodeHash    string     `gorm:"column:code_hash"`
	Attempts    int        `gorm:"column:attempts"`
	ResendCount int        `gorm:"column:resend_count"`
	LastSentAt  time.Time  `gorm:"column:last_sent_at"`
	ExpiresAt   time.Time  `gorm:"column:expires_at"`
	UsedAt      *time.Time `gorm:"column:used_at"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
}

func (verificationCodeRow) TableName() string { return "email_verification_codes" }

type VerifyRequestResult struct {
	Status string
}

func (s *Service) RequestEmailVerification(ctx context.Context, email string) (*VerifyRequestResult, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	user, err := s.users.GetByEmail(ctx, normalizedEmail)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("verify/request: email not found (masked)")
			return &VerifyRequestResult{Status: "accepted"}, nil
		}
		return nil, err
	}

	if isUserEmailVerified(user) {
		log.Printf("verify/request: already verified user_id=%d", user.ID)
		return &VerifyRequestResult{Status: "accepted"}, nil
	}

	now := time.Now()
	var current verificationCodeRow
	err = s.users.DB().WithContext(ctx).Where("user_id = ?", user.ID).First(&current).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if err == nil {
		cooldownUntil := current.LastSentAt.Add(s.verifyResendCooldown)
		if cooldownUntil.After(now) {
			return nil, ErrRateLimitExceeded
		}
	}

	code, genErr := generateVerificationCode()
	if genErr != nil {
		return nil, genErr
	}
	codeHash := hashVerificationCode(code, s.verificationCodePepper)
	expiresAt := now.Add(s.verifyCodeTTL)

	if errors.Is(err, gorm.ErrRecordNotFound) {
		row := verificationCodeRow{
			UserID:      user.ID,
			CodeHash:    codeHash,
			Attempts:    0,
			ResendCount: 1,
			LastSentAt:  now,
			ExpiresAt:   expiresAt,
		}
		if createErr := s.users.DB().WithContext(ctx).Create(&row).Error; createErr != nil {
			return nil, createErr
		}
	} else {
		if updateErr := s.users.DB().WithContext(ctx).
			Model(&verificationCodeRow{}).
			Where("user_id = ?", user.ID).
			Updates(map[string]any{
				"code_hash":    codeHash,
				"last_sent_at": now,
				"expires_at":   expiresAt,
				"resend_count": gorm.Expr("resend_count + 1"),
				"used_at":      nil,
			}).Error; updateErr != nil {
			return nil, updateErr
		}
	}

	if mailErr := s.mailer.SendVerificationCode(ctx, user.Email, code); mailErr != nil {
		return nil, mailErr
	}

	return &VerifyRequestResult{Status: "accepted"}, nil
}

func (s *Service) ConfirmEmailVerification(ctx context.Context, email, code string) error {
	if !codeRegex.MatchString(code) {
		return ErrInvalidVerificationCodeFormat
	}

	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	user, err := s.users.GetByEmail(ctx, normalizedEmail)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrInvalidVerificationCode
		}
		return err
	}

	now := time.Now()
	var row verificationCodeRow
	if err := s.users.DB().WithContext(ctx).Where("user_id = ?", user.ID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrInvalidVerificationCode
		}
		return err
	}

	if row.UsedAt != nil || !row.ExpiresAt.After(now) {
		return ErrInvalidVerificationCode
	}

	inputHash := hashVerificationCode(code, s.verificationCodePepper)
	if inputHash != row.CodeHash {
		attempts := row.Attempts + 1
		if updateErr := s.users.DB().WithContext(ctx).
			Model(&verificationCodeRow{}).
			Where("user_id = ?", user.ID).
			Update("attempts", attempts).Error; updateErr != nil {
			return updateErr
		}
		if attempts >= 5 {
			return ErrTooManyAttempts
		}
		return ErrInvalidVerificationCode
	}

	return s.users.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("users").Where("id = ?", user.ID).Updates(map[string]any{
			"email_verified":    true,
			"email_verified_at": now,
			"updated_at":        now,
		}).Error; err != nil {
			return err
		}

		if err := tx.Model(&verificationCodeRow{}).Where("user_id = ?", user.ID).Updates(map[string]any{
			"used_at": now,
		}).Error; err != nil {
			return err
		}

		return nil
	})
}

func generateVerificationCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func hashVerificationCode(code, pepper string) string {
	h := sha256.Sum256([]byte(code + pepper))
	return hex.EncodeToString(h[:])
}