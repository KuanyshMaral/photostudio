package admin

import (
	"context"
	"testing"
	"time"

	"photostudio/internal/domain"

	"gorm.io/gorm"
)

type mockUserRepo struct {
	user      *domain.User
	getErr    error
	updateErr error
}

func (m *mockUserRepo) DB() *gorm.DB { return nil }

func (m *mockUserRepo) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.user, nil
}

func (m *mockUserRepo) Update(ctx context.Context, u *domain.User) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.user = u
	return nil
}

type mockStudioOwnerRepo struct {
	owner     *domain.StudioOwner
	getErr    error
	updateErr error
}

func (m *mockStudioOwnerRepo) DB() *gorm.DB { return nil }

func (m *mockStudioOwnerRepo) FindByID(ctx context.Context, id int64) (*domain.StudioOwner, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.owner, nil
}

func (m *mockStudioOwnerRepo) Update(ctx context.Context, o *domain.StudioOwner) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.owner = o
	return nil
}

func (m *mockStudioOwnerRepo) FindPendingPaginated(ctx context.Context, offset, limit int) ([]domain.PendingStudioOwnerRow, int64, error) {
	return nil, 0, nil
}

func TestApproveStudioOwner_Success(t *testing.T) {
	ctx := context.Background()

	adminID := int64(1)
	ownerID := int64(10)
	userID := int64(5)

	u := &domain.User{
		ID:           userID,
		Role:         domain.RoleStudioOwner,
		StudioStatus: domain.StatusPending,
	}
	so := &domain.StudioOwner{
		ID:     ownerID,
		UserID: userID,
	}

	userRepo := &mockUserRepo{user: u}
	ownerRepo := &mockStudioOwnerRepo{owner: so}

	svc := NewService(
		userRepo,
		nil,
		nil,
		nil,
		ownerRepo,
		nil,
	)

	if err := svc.ApproveStudioOwner(ctx, ownerID, adminID); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if userRepo.user.StudioStatus != domain.StatusVerified {
		t.Fatalf("expected user studio_status = verified, got %v", userRepo.user.StudioStatus)
	}

	if ownerRepo.owner.VerifiedAt == nil {
		t.Fatalf("expected verified_at to be set")
	}
	if time.Since(*ownerRepo.owner.VerifiedAt) > 10*time.Second {
		t.Fatalf("expected verified_at to be recent, got %v", ownerRepo.owner.VerifiedAt)
	}

	if ownerRepo.owner.VerifiedBy == nil || *ownerRepo.owner.VerifiedBy != adminID {
		t.Fatalf("expected verified_by = %d, got %v", adminID, ownerRepo.owner.VerifiedBy)
	}

	if ownerRepo.owner.RejectedReason != "" {
		t.Fatalf("expected rejected_reason empty, got %q", ownerRepo.owner.RejectedReason)
	}
}

func TestApproveStudioOwner_NotPending(t *testing.T) {
	ctx := context.Background()

	adminID := int64(1)
	ownerID := int64(10)
	userID := int64(5)

	u := &domain.User{
		ID:           userID,
		Role:         domain.RoleStudioOwner,
		StudioStatus: domain.StatusVerified,
	}
	so := &domain.StudioOwner{
		ID:     ownerID,
		UserID: userID,
	}

	userRepo := &mockUserRepo{user: u}
	ownerRepo := &mockStudioOwnerRepo{owner: so}

	svc := NewService(
		userRepo,
		nil,
		nil,
		nil,
		ownerRepo,
		nil,
	)

	if err := svc.ApproveStudioOwner(ctx, ownerID, adminID); err == nil {
		t.Fatalf("expected error, got nil")
	}
}


