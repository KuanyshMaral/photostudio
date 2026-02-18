package relationship

import (
	"context"
)

// Service handles user blocking logic
type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Block blocks a user. Returns ErrCannotBlockSelf or ErrAlreadyBlocked.
func (s *Service) Block(ctx context.Context, blockerID, blockedID int64) error {
	if blockerID == blockedID {
		return ErrCannotBlockSelf
	}
	return s.repo.Block(ctx, blockerID, blockedID)
}

// Unblock removes a block. Returns ErrNotBlocked if not blocked.
func (s *Service) Unblock(ctx context.Context, blockerID, blockedID int64) error {
	return s.repo.Unblock(ctx, blockerID, blockedID)
}

// IsBlocked returns true if either user has blocked the other.
func (s *Service) IsBlocked(ctx context.Context, userA, userB int64) (bool, error) {
	return s.repo.IsBlocked(ctx, userA, userB)
}

// ListBlocked returns all users blocked by the given user.
func (s *Service) ListBlocked(ctx context.Context, userID int64) ([]*BlockRelation, error) {
	return s.repo.ListBlocked(ctx, userID)
}
