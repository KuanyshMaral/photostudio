package subscription

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RoomCounter is implemented by the catalog repository to count rooms per studio
type RoomCounter interface {
	CountRoomsByOwnerID(ctx context.Context, ownerID int64) (int, error)
}

// Service handles subscription business logic for Studio Owners.
// Clients (role='client') are NEVER passed to this service.
type Service struct {
	repo        Repository
	roomCounter RoomCounter
}

func NewService(repo Repository, roomCounter RoomCounter) *Service {
	return &Service{repo: repo, roomCounter: roomCounter}
}

// defaultFreePlan returns a fallback free plan when DB is unavailable
func defaultFreePlan() *Plan {
	return &Plan{
		ID:               PlanFree,
		Name:             "Бесплатный",
		MaxRooms:         1,
		MaxPhotosPerRoom: 5,
		MaxTeamMembers:   0,
		IsActive:         true,
	}
}

// GetPlans returns all active plans (public, no auth required)
func (s *Service) GetPlans(ctx context.Context) ([]*Plan, error) {
	return s.repo.ListPlans(ctx)
}

// GetCurrentSubscription returns the owner's active subscription and plan.
// If no subscription exists, returns a virtual free-tier subscription.
func (s *Service) GetCurrentSubscription(ctx context.Context, ownerID int64) (*Subscription, *Plan, error) {
	sub, err := s.repo.GetActiveByOwnerID(ctx, ownerID)
	if err != nil {
		return nil, nil, err
	}

	if sub == nil || sub.IsExpired() {
		// No active subscription → virtual free tier
		freePlan, _ := s.repo.GetPlanByID(ctx, PlanFree)
		if freePlan == nil {
			freePlan = defaultFreePlan()
		}
		return &Subscription{
			OwnerID:       ownerID,
			PlanID:        PlanFree,
			Status:        StatusActive,
			BillingPeriod: BillingMonthly,
			StartedAt:     time.Now(),
		}, freePlan, nil
	}

	plan, err := s.repo.GetPlanByID(ctx, sub.PlanID)
	if err != nil || plan == nil {
		plan = defaultFreePlan()
	}
	return sub, plan, nil
}

// Subscribe creates or upgrades a Studio Owner's subscription.
func (s *Service) Subscribe(ctx context.Context, ownerID int64, req *SubscribeRequest) (*Subscription, error) {
	planID := PlanID(req.PlanID)
	plan, err := s.repo.GetPlanByID(ctx, planID)
	if err != nil || plan == nil {
		return nil, ErrPlanNotFound
	}

	existing, err := s.repo.GetActiveByOwnerID(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.PlanID == planID {
		return nil, ErrAlreadySubscribed
	}

	period := BillingPeriod(req.BillingPeriod)
	var expiresAt time.Time
	switch period {
	case BillingMonthly:
		expiresAt = time.Now().AddDate(0, 1, 0)
	case BillingYearly:
		expiresAt = time.Now().AddDate(1, 0, 0)
	default:
		return nil, ErrInvalidBillingPeriod
	}

	// Cancel existing subscription if upgrading/changing
	if existing != nil {
		_ = s.repo.Cancel(ctx, existing.ID, fmt.Sprintf("Upgraded to %s", planID))
	}

	now := time.Now()
	sub := &Subscription{
		ID:            uuid.New().String(),
		OwnerID:       ownerID,
		PlanID:        planID,
		Status:        StatusActive,
		BillingPeriod: period,
		StartedAt:     now,
		ExpiresAt:     sql.NullTime{Time: expiresAt, Valid: true},
		AutoRenew:     true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.repo.Create(ctx, sub); err != nil {
		return nil, err
	}
	return sub, nil
}

// Cancel cancels a Studio Owner's active subscription.
func (s *Service) Cancel(ctx context.Context, ownerID int64, reason string) error {
	sub, err := s.repo.GetActiveByOwnerID(ctx, ownerID)
	if err != nil || sub == nil {
		return ErrSubscriptionNotFound
	}
	if sub.PlanID == PlanFree {
		return ErrCannotCancelFree
	}
	return s.repo.Cancel(ctx, sub.ID, reason)
}

// GetPlan returns the owner's current plan (falls back to free)
func (s *Service) GetPlan(ctx context.Context, ownerID int64) (*Plan, error) {
	_, plan, err := s.GetCurrentSubscription(ctx, ownerID)
	return plan, err
}

// ---- Limit Checkers (called by other services/handlers) ----

// CanAddRoom checks if the owner can create another room.
// Returns nil if allowed, LimitError if blocked.
func (s *Service) CanAddRoom(ctx context.Context, ownerID int64) error {
	plan, err := s.GetPlan(ctx, ownerID)
	if err != nil {
		return err
	}
	if plan.MaxRooms == -1 {
		return nil // unlimited
	}
	count, err := s.roomCounter.CountRoomsByOwnerID(ctx, ownerID)
	if err != nil {
		return err
	}
	if count >= plan.MaxRooms {
		return &LimitError{
			Err:       ErrRoomLimitReached,
			Current:   count,
			Limit:     plan.MaxRooms,
			PlanName:  string(plan.ID),
			UpgradeTo: nextPlan(plan.ID),
		}
	}
	return nil
}

// CanUploadPhoto checks if the owner can upload more photos to a room.
func (s *Service) CanUploadPhoto(ctx context.Context, ownerID int64, currentPhotoCount int) error {
	plan, err := s.GetPlan(ctx, ownerID)
	if err != nil {
		return err
	}
	if currentPhotoCount >= plan.MaxPhotosPerRoom {
		return &LimitError{
			Err:       ErrPhotoLimitReached,
			Current:   currentPhotoCount,
			Limit:     plan.MaxPhotosPerRoom,
			PlanName:  string(plan.ID),
			UpgradeTo: nextPlan(plan.ID),
		}
	}
	return nil
}

// HasFeature checks if the owner's plan includes a boolean feature.
func (s *Service) HasFeature(ctx context.Context, ownerID int64, feature string) (bool, error) {
	plan, err := s.GetPlan(ctx, ownerID)
	if err != nil {
		return false, err
	}
	switch feature {
	case "analytics_advanced":
		return plan.AnalyticsAdvanced, nil
	case "priority_search":
		return plan.PrioritySearch, nil
	case "priority_support":
		return plan.PrioritySupport, nil
	case "crm_access":
		return plan.CRMAccess, nil
	}
	return false, nil
}

// GetUsage returns current usage vs plan limits for a Studio Owner
func (s *Service) GetUsage(ctx context.Context, ownerID int64) (*UsageResponse, error) {
	_, plan, err := s.GetCurrentSubscription(ctx, ownerID)
	if err != nil {
		return nil, err
	}

	roomsCount := 0
	if s.roomCounter != nil {
		roomsCount, _ = s.roomCounter.CountRoomsByOwnerID(ctx, ownerID)
	}

	return &UsageResponse{
		PlanID:   string(plan.ID),
		PlanName: plan.Name,
		Limits: PlanLimits{
			MaxRooms:         plan.MaxRooms,
			MaxPhotosPerRoom: plan.MaxPhotosPerRoom,
			MaxTeamMembers:   plan.MaxTeamMembers,
		},
		Features: PlanFeatures{
			AnalyticsAdvanced: plan.AnalyticsAdvanced,
			PrioritySearch:    plan.PrioritySearch,
			PrioritySupport:   plan.PrioritySupport,
			CRMAccess:         plan.CRMAccess,
		},
		Usage: CurrentUsage{
			RoomsCount: roomsCount,
		},
	}, nil
}

// ExpireOldSubscriptions is called by a background job
func (s *Service) ExpireOldSubscriptions(ctx context.Context) (int, error) {
	return s.repo.ExpireOldSubscriptions(ctx)
}

func nextPlan(current PlanID) string {
	switch current {
	case PlanFree:
		return string(PlanStarter)
	case PlanStarter:
		return string(PlanPro)
	default:
		return ""
	}
}

// Ensure gorm.ErrRecordNotFound is accessible
var _ = gorm.ErrRecordNotFound
