package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"gorm.io/gorm"
	"strings"
	"time"

	"photostudio/internal/domain"

	"golang.org/x/crypto/bcrypt"
)

type jwtService interface {
	GenerateToken(userID int64, role string) (string, error)
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // seconds
}

// Service contains all business logic for authentication
type Service struct {
	users         UserRepositoryInterface
	studioOwners  StudioOwnerRepositoryInterface
	refreshTokens RefreshTokenRepositoryInterface
	jwt           jwtService
	accessTTL     time.Duration
	refreshTTL    time.Duration
}

// NewService creates a new auth service with all dependencies
// This is where we inject repositories — clean dependency injection
func NewService(
	users UserRepositoryInterface,
	studioOwners StudioOwnerRepositoryInterface,
	refreshTokens RefreshTokenRepositoryInterface,
	jwt jwtService,
	accessTTL time.Duration,
	refreshTTL time.Duration,
) *Service {
	return &Service{
		users:         users,
		studioOwners:  studioOwners,
		refreshTokens: refreshTokens,
		jwt:           jwt,
		accessTTL:     accessTTL,
		refreshTTL:    refreshTTL,
	}
}

// RegisterClient — creates a regular client user
func (s *Service) RegisterClient(ctx context.Context, req RegisterClientRequest) (*domain.User, TokenPair, error) {
	if err := s.validateEmailUnique(ctx, req.Email); err != nil {
		return nil, TokenPair{}, err
	}

	hashedPassword, err := s.hashPassword(req.Password)
	if err != nil {
		return nil, TokenPair{}, err
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
		return nil, TokenPair{}, err
	}

	tokens, err := s.issueTokens(ctx, user.ID, string(user.Role))
	if err != nil {
		return nil, TokenPair{}, err
	}

	user.PasswordHash = "" // never expose hash
	return user, tokens, nil
}

// RegisterStudioOwner — creates user + studio_owner record atomically
func (s *Service) RegisterStudioOwner(ctx context.Context, req RegisterStudioRequest) (*domain.User, TokenPair, error) {
	if err := s.validateEmailUnique(ctx, req.Email); err != nil {
		return nil, TokenPair{}, err
	}

	hashedPassword, err := s.hashPassword(req.Password)
	if err != nil {
		return nil, TokenPair{}, err
	}

	// Start transaction — ensures atomicity
	tx := s.users.DB().WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, TokenPair{}, tx.Error
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
		return nil, TokenPair{}, err
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
		return nil, TokenPair{}, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, TokenPair{}, err
	}

	tokens, err := s.issueTokens(ctx, user.ID, string(user.Role))
	if err != nil {
		return nil, TokenPair{}, err
	}

	user.PasswordHash = ""
	return user, tokens, nil
}

// Login — authenticates user and returns JWT
func (s *Service) Login(ctx context.Context, req LoginRequest) (*domain.User, TokenPair, error) {
	user, err := s.users.GetByEmail(ctx, strings.ToLower(strings.TrimSpace(req.Email)))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, TokenPair{}, ErrInvalidCredentials
		}
		return nil, TokenPair{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, TokenPair{}, ErrInvalidCredentials
	}

	tokens, err := s.issueTokens(ctx, user.ID, string(user.Role))
	if err != nil {
		return nil, TokenPair{}, err
	}

	user.PasswordHash = ""
	return user, tokens, nil
}

// Refresh rotates refresh token and returns new token pair.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (*domain.User, TokenPair, error) {
	if refreshToken == "" {
		return nil, TokenPair{}, ErrUnauthorized
	}

	// Cleanup old tokens best-effort
	_ = s.refreshTokens.DeleteExpired(ctx)

	hash := sha256Hex(refreshToken)
	stored, err := s.refreshTokens.GetByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, TokenPair{}, ErrUnauthorized
		}
		return nil, TokenPair{}, err
	}

	now := time.Now().UTC()
	if stored.IsRevoked() || stored.IsExpired(now) {
		return nil, TokenPair{}, ErrUnauthorized
	}

	user, err := s.users.GetByID(ctx, stored.UserID)
	if err != nil {
		return nil, TokenPair{}, ErrUnauthorized
	}

	// Issue a new refresh token (rotation)
	newRefreshRaw, newRefreshHash, err := generateRefreshToken()
	if err != nil {
		return nil, TokenPair{}, err
	}
	newRec := &domain.RefreshToken{
		UserID:    user.ID,
		TokenHash: newRefreshHash,
		ExpiresAt: now.Add(s.refreshTTL),
	}
	if err := s.refreshTokens.Create(ctx, newRec); err != nil {
		return nil, TokenPair{}, err
	}
	// Revoke old and link replacement (best-effort)
	_ = s.refreshTokens.Revoke(ctx, stored.ID, &newRec.ID)

	access, err := s.jwt.GenerateToken(user.ID, string(user.Role))
	if err != nil {
		return nil, TokenPair{}, err
	}

	user.PasswordHash = ""
	return user, TokenPair{
		AccessToken:  access,
		RefreshToken: newRefreshRaw,
		ExpiresIn:    int64(s.accessTTL.Seconds()),
	}, nil
}

// Logout revokes current refresh token (if provided)
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}
	hash := sha256Hex(refreshToken)
	stored, err := s.refreshTokens.GetByHash(ctx, hash)
	if err != nil {
		return nil // don't leak
	}
	return s.refreshTokens.Revoke(ctx, stored.ID, nil)
}

func (s *Service) AccessTTLSeconds() int {
	return int(s.accessTTL.Seconds())
}

func (s *Service) RefreshTTLSeconds() int {
	return int(s.refreshTTL.Seconds())
}

func (s *Service) issueTokens(ctx context.Context, userID int64, role string) (TokenPair, error) {
	access, err := s.jwt.GenerateToken(userID, role)
	if err != nil {
		return TokenPair{}, err
	}

	refreshRaw, refreshHash, err := generateRefreshToken()
	if err != nil {
		return TokenPair{}, err
	}

	now := time.Now().UTC()
	rec := &domain.RefreshToken{
		UserID:    userID,
		TokenHash: refreshHash,
		ExpiresAt: now.Add(s.refreshTTL),
	}
	if err := s.refreshTokens.Create(ctx, rec); err != nil {
		return TokenPair{}, err
	}

	return TokenPair{
		AccessToken:  access,
		RefreshToken: refreshRaw,
		ExpiresIn:    int64(s.accessTTL.Seconds()),
	}, nil
}

func generateRefreshToken() (raw string, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	raw = hex.EncodeToString(b)
	hash = sha256Hex(raw)
	return raw, hash, nil
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
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
