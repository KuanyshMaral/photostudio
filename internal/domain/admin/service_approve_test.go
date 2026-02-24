package admin

import (
	"context"
	"database/sql"
	"photostudio/internal/domain/auth"
	"photostudio/internal/domain/profile"
	"testing"
	"time"

	"gorm.io/gorm"
)

type mockUserRepo struct {
	user      *auth.User
	getErr    error
	updateErr error
}

func (m *mockUserRepo) DB() *gorm.DB { return nil }

func (m *mockUserRepo) GetByID(ctx context.Context, id int64) (*auth.User, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.user, nil
}

func (m *mockUserRepo) Update(ctx context.Context, u *auth.User) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.user = u
	return nil
}

type mockOwnerProfileRepo struct {
	profile      *profile.OwnerProfile
	getErr       error
	updateErr    error
	verifyStatus string
	verifyReason string
	verifyCalled bool
}

func (m *mockOwnerProfileRepo) GetByID(ctx context.Context, id int64) (*profile.OwnerProfile, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.profile, nil
}

func (m *mockOwnerProfileRepo) GetByUserID(ctx context.Context, userID int64) (*profile.OwnerProfile, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.profile, nil
}

func (m *mockOwnerProfileRepo) Update(ctx context.Context, p *profile.OwnerProfile) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.profile = p
	return nil
}

func (m *mockOwnerProfileRepo) FindPendingPaginated(ctx context.Context, offset, limit int) ([]profile.PendingOwnerProfileRow, int64, error) {
	return nil, 0, nil
}

func (m *mockOwnerProfileRepo) UpdateVerificationStatus(ctx context.Context, userID, adminID int64, status, reason, notes string) error {
	m.verifyCalled = true
	m.verifyStatus = status
	m.verifyReason = reason
	if m.profile != nil {
		m.profile.VerificationStatus = status
		m.profile.VerifiedBy = sql.NullInt64{Int64: adminID, Valid: true}
		if status == "verified" {
			now := time.Now()
			m.profile.VerifiedAt = sql.NullTime{Time: now, Valid: true}
		}
	}
	return nil
}

func TestApproveStudioOwner_Success(t *testing.T) {
	ctx := context.Background()

	adminID := int64(1)
	ownerID := int64(10)
	userID := int64(5)

	u := &auth.User{
		ID:           userID,
		Role:         auth.RoleStudioOwner,
		StudioStatus: auth.StatusPending,
	}
	// ownerID here acts as ProfileID passed from frontend
	prof := &profile.OwnerProfile{
		ID:                 ownerID,
		UserID:             userID,
		VerificationStatus: "pending",
	}

	userRepo := &mockUserRepo{user: u}
	ownerRepo := &mockOwnerProfileRepo{profile: prof}

	svc := NewService(
		userRepo,
		nil,
		nil,
		nil,
		ownerRepo,
		nil,
		nil,
		nil,
		nil,
	)

	// Since implementation logic fetches using ownerRepo.GetByUserID OR GetByID.
	// We mocked GetByUserID to return same profile.
	if err := svc.ApproveStudioOwner(ctx, ownerID, adminID); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if userRepo.user.StudioStatus != auth.StatusVerified {
		t.Fatalf("expected user studio_status = verified, got %v", userRepo.user.StudioStatus)
	}

	if !ownerRepo.verifyCalled {
		t.Fatal("expected UpdateVerificationStatus to be called")
	}
	if ownerRepo.verifyStatus != "verified" {
		t.Fatalf("expected status verified, got %s", ownerRepo.verifyStatus)
	}
}

func TestApproveStudioOwner_NotPending(t *testing.T) {
	ctx := context.Background()

	adminID := int64(1)
	ownerID := int64(10)
	userID := int64(5)

	u := &auth.User{
		ID:           userID,
		Role:         auth.RoleStudioOwner,
		StudioStatus: auth.StatusVerified,
	}
	prof := &profile.OwnerProfile{
		ID:                 ownerID,
		UserID:             userID,
		VerificationStatus: "verified",
	}

	userRepo := &mockUserRepo{user: u}
	ownerRepo := &mockOwnerProfileRepo{profile: prof}

	svc := NewService(
		userRepo,
		nil,
		nil,
		nil,
		ownerRepo,
		nil,
		nil,
		nil,
		nil,
	)

	if err := svc.ApproveStudioOwner(ctx, ownerID, adminID); err == nil {
		t.Fatalf("expected error, got nil")
	}
}
