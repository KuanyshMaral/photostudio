package subscription

// SubscribeRequest is sent by a Studio Owner to subscribe to a plan
type SubscribeRequest struct {
	PlanID        string `json:"plan_id" binding:"required"`
	BillingPeriod string `json:"billing_period" binding:"required,oneof=monthly yearly"`
}

// CancelRequest is sent by a Studio Owner to cancel their subscription
type CancelRequest struct {
	Reason string `json:"reason"`
}

// PlanResponse is the public representation of a plan
type PlanResponse struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Description  string       `json:"description"`
	PriceMonthly float64      `json:"price_monthly"`
	PriceYearly  *float64     `json:"price_yearly,omitempty"`
	Limits       PlanLimits   `json:"limits"`
	Features     PlanFeatures `json:"features"`
}

// PlanLimits groups numeric limits for display
type PlanLimits struct {
	MaxRooms         int `json:"max_rooms"` // -1 = unlimited
	MaxPhotosPerRoom int `json:"max_photos_per_room"`
	MaxTeamMembers   int `json:"max_team_members"` // 0 = solo only
}

// PlanFeatures groups boolean feature flags for display
type PlanFeatures struct {
	AnalyticsAdvanced bool `json:"analytics_advanced"`
	PrioritySearch    bool `json:"priority_search"`
	PrioritySupport   bool `json:"priority_support"`
	CRMAccess         bool `json:"crm_access"`
}

// SubscriptionResponse is the public representation of an active subscription
type SubscriptionResponse struct {
	ID            string       `json:"id"`
	PlanID        string       `json:"plan_id"`
	PlanName      string       `json:"plan_name"`
	Status        string       `json:"status"`
	BillingPeriod string       `json:"billing_period"`
	StartedAt     string       `json:"started_at"`
	ExpiresAt     *string      `json:"expires_at,omitempty"`
	DaysRemaining int          `json:"days_remaining"`
	AutoRenew     bool         `json:"auto_renew"`
	Limits        PlanLimits   `json:"limits"`
	Features      PlanFeatures `json:"features"`
}

// UsageResponse shows current usage vs plan limits for a Studio Owner
type UsageResponse struct {
	PlanID   string       `json:"plan_id"`
	PlanName string       `json:"plan_name"`
	Limits   PlanLimits   `json:"limits"`
	Features PlanFeatures `json:"features"`
	Usage    CurrentUsage `json:"usage"`
}

// CurrentUsage tracks actual resource usage
type CurrentUsage struct {
	RoomsCount       int `json:"rooms_count"`
	TeamMembersCount int `json:"team_members_count"`
}

func planToResponse(p *Plan) PlanResponse {
	return PlanResponse{
		ID:           string(p.ID),
		Name:         p.Name,
		Description:  p.Description,
		PriceMonthly: p.PriceMonthly,
		PriceYearly:  p.PriceYearly,
		Limits: PlanLimits{
			MaxRooms:         p.MaxRooms,
			MaxPhotosPerRoom: p.MaxPhotosPerRoom,
			MaxTeamMembers:   p.MaxTeamMembers,
		},
		Features: PlanFeatures{
			AnalyticsAdvanced: p.AnalyticsAdvanced,
			PrioritySearch:    p.PrioritySearch,
			PrioritySupport:   p.PrioritySupport,
			CRMAccess:         p.CRMAccess,
		},
	}
}
