package auth

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"strings"

	"photostudio/internal/domain"

	"golang.org/x/crypto/bcrypt"
)

type jwtService interface {
	GenerateToken(userID int64, role string) (string, error)
}

// Service contains all business logic for authentication
type Service struct {
	users        UserRepositoryInterface
	studioOwners StudioOwnerRepositoryInterface
	jwt          jwtService
}

// NewService creates a new auth service with all dependencies
// This is where we inject repositories — clean dependency injection
func NewService(
	users UserRepositoryInterface,
	studioOwners StudioOwnerRepositoryInterface,
	jwt jwtService,
) *Service {
	return &Service{
		users:        users,
		studioOwners: studioOwners,
		jwt:          jwt,
	}
}

// RegisterClient — creates a regular client user
func (s *Service) RegisterClient(ctx context.Context, req RegisterClientRequest) (*domain.User, string, error) {
	if err := s.validateEmailUnique(ctx, req.Email); err != nil {
		return nil, "", err
	}

	hashedPassword, err := s.hashPassword(req.Password)
	if err != nil {
		return nil, "", err
	}

	user := &domain.User{
		Email:         strings.ToLower(strings.TrimSpace(req.Email)),
		PasswordHash:  hashedPassword,
		Name:          req.Name,
		Phone:         req.Phone,
		Role:          domain.RoleClient,
		EmailVerified: false,
	}

	if err := s.users.Create(ctx, user); err != nil {
		return nil, "", err
	}

	token, err := s.jwt.GenerateToken(user.ID, string(user.Role))
	if err != nil {
		return nil, "", err
	}

	user.PasswordHash = "" // never expose hash
	return user, token, nil
}

// RegisterStudioOwner — creates user + studio_owner record atomically
func (s *Service) RegisterStudioOwner(ctx context.Context, req RegisterStudioRequest) (*domain.User, string, error) {
	if err := s.validateEmailUnique(ctx, req.Email); err != nil {
		return nil, "", err
	}

	hashedPassword, err := s.hashPassword(req.Password)
	if err != nil {
		return nil, "", err
	}

	// Start transaction — ensures atomicity
	tx := s.users.DB().WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, "", tx.Error
	}
	// Rollback on panic (safety net)
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	user := &domain.User{
		Email:         strings.ToLower(strings.TrimSpace(req.Email)),
		PasswordHash:  hashedPassword,
		Name:          req.Name,
		Phone:         req.Phone,
		Role:          domain.RoleStudioOwner,
		StudioStatus:  domain.StatusPending, // "pending"
		EmailVerified: false,
	}

	if err := tx.Create(user).Error; err != nil {
		tx.Rollback()
		return nil, "", err
	}

	studioOwner := &domain.StudioOwner{
		UserID:           user.ID,
		CompanyName:      req.CompanyName,
		BIN:              req.BIN,
		LegalAddress:     req.LegalAddress,
		ContactPerson:    req.ContactPerson,
		ContactPosition:  req.ContactPosition,
		VerificationDocs: []string{},
	}

	if err := tx.Create(studioOwner).Error; err != nil {
		tx.Rollback()
		return nil, "", err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, "", err
	}

	token, err := s.jwt.GenerateToken(user.ID, string(user.Role))
	if err != nil {
		return nil, "", err
	}

	user.PasswordHash = ""
	return user, token, nil
}

// Login — authenticates user and returns JWT
func (s *Service) Login(ctx context.Context, req LoginRequest) (*domain.User, string, error) {
	user, err := s.users.GetByEmail(ctx, strings.ToLower(strings.TrimSpace(req.Email)))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", ErrInvalidCredentials
		}
		return nil, "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, "", ErrInvalidCredentials
	}

	token, err := s.jwt.GenerateToken(user.ID, string(user.Role))
	if err != nil {
		return nil, "", err
	}

	user.PasswordHash = ""
	return user, token, nil
}

// GetCurrentUser — used by /users/me endpoint
func (s *Service) GetCurrentUser(ctx context.Context, userID int64) (*domain.User, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	user.PasswordHash = ""
	return user, nil
}

// UpdateProfile — allows user to update name/phone
func (s *Service) UpdateProfile(ctx context.Context, userID int64, req UpdateProfileRequest) (*domain.User, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		user.Name = req.Name
	}
	if req.Phone != "" {
		user.Phone = req.Phone
	}

	if err := s.users.Update(ctx, user); err != nil {
		return nil, err
	}

	user.PasswordHash = ""
	return user, nil
}

// AppendVerificationDocs — called after file upload
func (s *Service) AppendVerificationDocs(ctx context.Context, userID int64, urls []string) error {
	if len(urls) == 0 {
		return nil
	}
	return s.studioOwners.AppendVerificationDocs(ctx, userID, urls)
}

// ———— Private helper methods ————

// validateEmailUnique — DRY: used by both registration methods
func (s *Service) validateEmailUnique(ctx context.Context, email string) error {
	exists, err := s.users.ExistsByEmail(ctx, email)
	if err != nil {
		return err
	}
	if exists {
		return ErrEmailAlreadyExists
	}
	return nil
}

// hashPassword — centralize bcrypt logic
func (s *Service) hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}


