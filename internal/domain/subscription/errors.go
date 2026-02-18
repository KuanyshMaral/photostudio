package subscription

import "errors"

var (
	ErrPlanNotFound         = errors.New("subscription plan not found")
	ErrSubscriptionNotFound = errors.New("subscription not found")
	ErrAlreadySubscribed    = errors.New("already subscribed to this plan")
	ErrCannotCancelFree     = errors.New("cannot cancel free plan")
	ErrInvalidBillingPeriod = errors.New("invalid billing period")
	ErrNotOwner             = errors.New("only studio owners can have subscriptions")

	// Limit errors returned when an owner exceeds their plan
	ErrRoomLimitReached       = errors.New("room limit reached for your current plan — upgrade to add more rooms")
	ErrPhotoLimitReached      = errors.New("photo limit per room reached for your current plan")
	ErrTeamMemberLimitReached = errors.New("team member limit reached for your current plan")
	ErrFeatureNotAvailable    = errors.New("this feature is not available on your current plan")
)

// LimitError carries rich context for UI display
type LimitError struct {
	Err       error
	Current   int
	Limit     int
	PlanName  string
	UpgradeTo string
}

func (e *LimitError) Error() string { return e.Err.Error() }
func (e *LimitError) Unwrap() error { return e.Err }
