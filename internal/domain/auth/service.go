package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"photostudio/internal/domain/profile"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	maxFailedLoginAttempts = 5
	lockoutDuration        = 15 * time.Minute
)

type jwtService interface {
	GenerateToken(userID int64, role string) (string, error)
}

type ProfileService interface {
	EnsureClientProfile(ctx context.Context, userID int64) (*profile.ClientProfile, error)
	EnsureOwnerProfile(ctx context.Context, userID int64, req *profile.CreateOwnerProfileRequest) (*profile.OwnerProfile, error)
	EnsureAdminProfile(ctx context.Context, userID uuid.UUID, req *profile.CreateAdminProfileRequest) (*profile.AdminProfile, error)
	GetClientProfile(ctx context.Context, userID int64) (*profile.ClientProfile, error)
	GetOwnerProfile(ctx context.Context, userID int64) (*profile.OwnerProfile, error)
	GetAdminProfile(ctx context.Context, userID uuid.UUID) (*profile.AdminProfile, error)
}

// Service contains all business logic for authentication
type Service struct {
	users                  UserRepositoryInterface
	ownerProfiles          OwnerProfileRepositoryInterface
	profileService         ProfileService
	jwt                    jwtService
	mailer                 Mailer
	verificationCodePepper string
	verifyCodeTTL          time.Duration
	verifyResendCooldown   time.Duration
	refreshTokenPepper     string
	refreshTTL             time.Duration
}

type LoginResult struct {
	User         *User
	AccessToken  string
	RefreshToken string
}

type RefreshResult struct {
	AccessToken  string
	RefreshToken string
}

type refreshTokenRow struct {
	UserAgent       *string    `gorm:"column:user_agent"`
	IP              *string    `gorm:"column:ip"`
	ID              int64      `gorm:"column:id"`
	UserID          int64      `gorm:"column:user_id"`
	TokenHash       string     `gorm:"column:token_hash"`
	JTI             string     `gorm:"column:jti"`
	FamilyID        string     `gorm:"column:family_id"`
	RotatedFrom     *int64     `gorm:"column:rotated_from"`
	ExpiresAt       time.Time  `gorm:"column:expires_at"`
	UsedAt          *time.Time `gorm:"column:used_at"`
	RevokedAt       *time.Time `gorm:"column:revoked_at"`
	ReuseDetectedAt *time.Time `gorm:"column:reuse_detected_at"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
}

func (refreshTokenRow) TableName() string { return "refresh_tokens" }

func NewService(
	users UserRepositoryInterface,
	ownerProfiles OwnerProfileRepositoryInterface,
	profileService ProfileService,
	jwt jwtService,
	mailer Mailer,
	verificationCodePepper string,
	verifyCodeTTL time.Duration,
	verifyResendCooldown time.Duration,
	refreshTokenPepper string,
	refreshTTL time.Duration,
) *Service {
	return &Service{
		users:                  users,
		ownerProfiles:          ownerProfiles,
		profileService:         profileService,
		jwt:                    jwt,
		mailer:                 mailer,
		verificationCodePepper: verificationCodePepper,
		verifyCodeTTL:          verifyCodeTTL,
		verifyResendCooldown:   verifyResendCooldown,
		refreshTokenPepper:     refreshTokenPepper,
		refreshTTL:             refreshTTL,
	}
}

func (s *Service) RegisterClient(ctx context.Context, req RegisterClientRequest) (*User, bool, error) {
	if err := s.validateEmailUnique(ctx, req.Email); err != nil {
		return nil, false, err
	}

	hashedPassword, err := s.hashPassword(req.Password)
	if err != nil {
		return nil, false, err
	}

	user := &User{
		Email:         strings.ToLower(strings.TrimSpace(req.Email)),
		PasswordHash:  hashedPassword,
		Role:          RoleClient,
		EmailVerified: false,
	}

	if err := s.users.Create(ctx, user); err != nil {
		return nil, false, err
	}

	if _, err := s.RequestEmailVerification(ctx, user.Email); err != nil {
		return nil, false, err
	}

	user.PasswordHash = ""
	return user, true, nil
}

func (s *Service) Login(ctx context.Context, req LoginRequest, userAgent, ip string) (*LoginResult, error) {
	user, err := s.users.GetByEmail(ctx, strings.ToLower(strings.TrimSpace(req.Email)))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	now := time.Now()
	if user.StudioStatus == StatusBlocked && !user.IsBanned {
		_ = s.users.DB().WithContext(ctx).Table("users").Where("id = ?", user.ID).Updates(map[string]any{"is_banned": true, "banned_at": now, "ban_reason": "studio_status_blocked"}).Error
		user.IsBanned = true
	}
	if user.IsBanned || user.StudioStatus == StatusBlocked {
		return nil, ErrAccountBanned
	}
	if user.LockedUntil != nil && user.LockedUntil.After(now) {
		return nil, ErrAccountLocked
	}
	if !isUserEmailVerified(user) {
		return nil, ErrEmailNotVerified
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		failedAttempts := user.FailedLoginAttempts + 1
		updates := map[string]any{"failed_login_attempts": failedAttempts}
		if failedAttempts >= maxFailedLoginAttempts {
			updates["locked_until"] = now.Add(lockoutDuration)
		}
		if updateErr := s.users.DB().WithContext(ctx).Table("users").Where("id = ?", user.ID).Updates(updates).Error; updateErr != nil {
			return nil, updateErr
		}
		if failedAttempts >= maxFailedLoginAttempts {
			return nil, ErrAccountLocked
		}
		return nil, ErrInvalidCredentials
	}

	if user.FailedLoginAttempts > 0 || user.LockedUntil != nil {
		if err := s.users.DB().WithContext(ctx).Table("users").Where("id = ?", user.ID).Updates(map[string]any{
			"failed_login_attempts": 0,
			"locked_until":          nil,
		}).Error; err != nil {
			return nil, err
		}
	}

	accessToken, err := s.jwt.GenerateToken(user.ID, string(user.Role))
	if err != nil {
		return nil, err
	}

	refreshTokenRaw, refreshHash, err := generateOpaqueRefreshToken(s.refreshTokenPepper)
	if err != nil {
		return nil, err
	}

	familyID := uuid.NewString()
	uaPtr := nullableString(userAgent)
	ipPtr := nullableString(ip)
	if err := s.users.DB().WithContext(ctx).Create(&refreshTokenRow{
		UserID:    user.ID,
		TokenHash: refreshHash,
		JTI:       uuid.NewString(),
		FamilyID:  familyID,
		ExpiresAt: now.Add(s.refreshTTL),
		UserAgent: uaPtr,
		IP:        ipPtr,
	}).Error; err != nil {
		return nil, err
	}

	var keepIDs []int64
	_ = s.users.DB().WithContext(ctx).
		Table("refresh_tokens").
		Select("id").
		Where("user_id = ? AND revoked_at IS NULL", user.ID).
		Order("created_at DESC").
		Limit(10).
		Pluck("id", &keepIDs).Error

	if len(keepIDs) > 0 {
		_ = s.users.DB().WithContext(ctx).
			Table("refresh_tokens").
			Where("user_id = ? AND revoked_at IS NULL AND id NOT IN ?", user.ID, keepIDs).
			Updates(map[string]interface{}{"revoked_at": time.Now()}).Error
	}

	user.PasswordHash = ""

	// Lazy profile generation: ensure the user has a profile row immediately on login.
	// This guarantees the profile exists before the frontend makes any profile API calls or opens sockets.
	if s.profileService != nil {
		switch user.Role {
		case RoleClient:
			_, _ = s.profileService.EnsureClientProfile(ctx, user.ID)
		case RoleStudioOwner:
			_, _ = s.profileService.EnsureOwnerProfile(ctx, user.ID, &profile.CreateOwnerProfileRequest{
				CompanyName: "", // Skeleton — owner fills this in their profile settings
			})
		}
	}

	return &LoginResult{User: user, AccessToken: accessToken, RefreshToken: refreshTokenRaw}, nil
}

func (s *Service) RefreshSession(ctx context.Context, refreshRaw, userAgent, ip string) (*RefreshResult, error) {
	now := time.Now()
	hash := hashTokenWithPepper(refreshRaw, s.refreshTokenPepper)
	var result *RefreshResult

	err := s.users.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current refreshTokenRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("token_hash = ?", hash).First(&current).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInvalidRefreshToken
			}
			return err
		}

		if !current.ExpiresAt.After(now) {
			return ErrInvalidRefreshToken
		}

		if current.UsedAt != nil || current.RevokedAt != nil {
			if err := tx.Model(&refreshTokenRow{}).Where("id = ?", current.ID).Updates(map[string]any{
				"reuse_detected_at": now,
			}).Error; err != nil {
				return err
			}
			if err := tx.Model(&refreshTokenRow{}).Where("family_id = ? AND revoked_at IS NULL", current.FamilyID).Updates(map[string]any{
				"revoked_at": now,
			}).Error; err != nil {
				return err
			}
			return ErrRefreshTokenReused
		}

		var user User
		if err := tx.Table("users").Where("id = ?", current.UserID).First(&user).Error; err != nil {
			return err
		}
		if user.StudioStatus == StatusBlocked || user.IsBanned {
			if err := tx.Model(&refreshTokenRow{}).Where("family_id = ? AND revoked_at IS NULL", current.FamilyID).Updates(map[string]any{"revoked_at": now}).Error; err != nil {
				return err
			}
			return ErrAccountBanned
		}
		if !isUserEmailVerified(&user) {
			return ErrEmailNotVerified
		}

		accessToken, err := s.jwt.GenerateToken(user.ID, string(user.Role))
		if err != nil {
			return err
		}
		newRaw, newHash, err := generateOpaqueRefreshToken(s.refreshTokenPepper)
		if err != nil {
			return err
		}

		if err := tx.Model(&refreshTokenRow{}).Where("id = ?", current.ID).Updates(map[string]any{
			"used_at":    now,
			"revoked_at": now,
		}).Error; err != nil {
			return err
		}
		rotatedFrom := current.ID
		uaPtr := nullableString(userAgent)
		ipPtr := nullableString(ip)
		if err := tx.Create(&refreshTokenRow{
			UserID:      current.UserID,
			TokenHash:   newHash,
			JTI:         uuid.NewString(),
			FamilyID:    current.FamilyID,
			RotatedFrom: &rotatedFrom,
			ExpiresAt:   now.Add(s.refreshTTL),
			UserAgent:   uaPtr,
			IP:          ipPtr,
		}).Error; err != nil {
			return err
		}
		result = &RefreshResult{AccessToken: accessToken, RefreshToken: newRaw}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Service) Logout(ctx context.Context, refreshRaw string) error {
	hash := hashTokenWithPepper(refreshRaw, s.refreshTokenPepper)
	now := time.Now()

	var token refreshTokenRow
	if err := s.users.DB().WithContext(ctx).Where("token_hash = ?", hash).First(&token).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	return s.users.DB().WithContext(ctx).Model(&refreshTokenRow{}).Where("id = ?", token.ID).Updates(map[string]any{
		"revoked_at": now,
	}).Error
}

func (s *Service) GetCurrentUser(ctx context.Context, userID int64) (*User, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	user.PasswordHash = ""
	return user, nil
}

// UpdateProfile has been removed - profile updates happen via the profile domain
// (PUT /profiles/client, PUT /profiles/owner, etc.)
// This function is intentionally left empty.

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

func (s *Service) hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func generateOpaqueRefreshToken(pepper string) (raw string, hash string, err error) {
	buf := make([]byte, 32)
	if _, err = rand.Read(buf); err != nil {
		return "", "", err
	}
	raw = hex.EncodeToString(buf)
	hash = hashTokenWithPepper(raw, pepper)
	return raw, hash, nil
}

func hashTokenWithPepper(raw, pepper string) string {
	sum := sha256.Sum256([]byte(raw + pepper))
	return hex.EncodeToString(sum[:])
}

func isUserEmailVerified(user *User) bool {
	return user.EmailVerifiedAt != nil || user.EmailVerified
}

func nullableString(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}
