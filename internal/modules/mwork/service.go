package mwork

import (
	"context"
	"errors"
	"strings"

	"photostudio/internal/domain"
	"photostudio/internal/repository"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type SyncResult string

const (
	ResultCreated SyncResult = "created"
	ResultUpdated SyncResult = "updated"
	ResultLinked  SyncResult = "linked"
)

var ErrConflict = errors.New("mwork sync conflict")

type Service struct {
	users *repository.UserRepository
}

func NewService(users *repository.UserRepository) *Service {
	return &Service{users: users}
}

func (s *Service) SyncUser(ctx context.Context, req SyncUserRequest) (*domain.User, SyncResult, error) {
	normalizedEmail := normalizeEmail(req.Email)
	tx := s.users.DB().WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, "", tx.Error
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	userRepo := repository.NewUserRepository(tx)

	user, err := userRepo.GetByMworkUserID(ctx, req.MworkUserID)
	if err == nil {
		updated := false
		if !emailsEqual(user.Email, normalizedEmail) {
			user.Email = normalizedEmail
			updated = true
		}
		if user.MworkRole != req.Role {
			user.MworkRole = req.Role
			updated = true
		}
		if updated {
			if err := userRepo.Update(ctx, user); err != nil {
				tx.Rollback()
				if isUniqueViolation(err) {
					return nil, "", ErrConflict
				}
				return nil, "", err
			}
		}
		if err := tx.Commit().Error; err != nil {
			return nil, "", err
		}
		return user, ResultUpdated, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		tx.Rollback()
		return nil, "", err
	}

	user, err = userRepo.GetByEmail(ctx, normalizedEmail)
	if err == nil {
		if user.MworkUserID != "" && user.MworkUserID != req.MworkUserID {
			tx.Rollback()
			return nil, "", ErrConflict
		}
		user.MworkUserID = req.MworkUserID
		user.MworkRole = req.Role
		user.Email = normalizedEmail
		if err := userRepo.Update(ctx, user); err != nil {
			tx.Rollback()
			if isUniqueViolation(err) {
				return nil, "", ErrConflict
			}
			return nil, "", err
		}
		if err := tx.Commit().Error; err != nil {
			return nil, "", err
		}
		return user, ResultLinked, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		tx.Rollback()
		return nil, "", err
	}

	password, err := bcrypt.GenerateFromPassword([]byte(uuid.NewString()), bcrypt.DefaultCost)
	if err != nil {
		tx.Rollback()
		return nil, "", err
	}

	user = &domain.User{
		Email:         normalizedEmail,
		PasswordHash:  string(password),
		Role:          domain.RoleClient,
		Name:          normalizedEmail,
		EmailVerified: false,
		MworkUserID:   req.MworkUserID,
		MworkRole:     req.Role,
	}

	if err := userRepo.Create(ctx, user); err != nil {
		tx.Rollback()
		if isUniqueViolation(err) {
			return nil, "", ErrConflict
		}
		return nil, "", err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, "", err
	}
	return user, ResultCreated, nil
}

func emailsEqual(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func isUniqueViolation(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "duplicate key value violates unique constraint") ||
		strings.Contains(msg, "SQLSTATE 23505") ||
		strings.Contains(msg, "23505") ||
		strings.Contains(msg, "UNIQUE constraint failed")
}
