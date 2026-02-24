package subscription

import (
	"database/sql"
	"time"
)

// PlanID identifies a subscription tier
type PlanID string

const (
	PlanFree    PlanID = "free"
	PlanStarter PlanID = "starter"
	PlanPro     PlanID = "pro"
)

// Status of a subscription
type Status string

const (
	StatusActive    Status = "active"
	StatusCancelled Status = "cancelled"
	StatusExpired   Status = "expired"
	StatusPastDue   Status = "past_due"
	StatusPending   Status = "pending"
)

// BillingPeriod for subscription cycle
type BillingPeriod string

const (
	BillingMonthly BillingPeriod = "monthly"
	BillingYearly  BillingPeriod = "yearly"
)

// Plan defines a subscription tier available to Studio Owners.
// Clients (role='client') are NEVER assigned plans — they are pure consumers.
type Plan struct {
	ID          PlanID `gorm:"column:id;primaryKey" json:"id"`
	Name        string `gorm:"column:name" json:"name"`
	Description string `gorm:"column:description" json:"description"`

	PriceMonthly float64  `gorm:"column:price_monthly" json:"price_monthly"`
	PriceYearly  *float64 `gorm:"column:price_yearly" json:"price_yearly,omitempty"`

	// Numeric limits — applied only to Studio Owners
	MaxRooms         int `gorm:"column:max_rooms" json:"max_rooms"` // -1 = unlimited
	MaxPhotosPerRoom int `gorm:"column:max_photos_per_room" json:"max_photos_per_room"`
	MaxTeamMembers   int `gorm:"column:max_team_members" json:"max_team_members"` // 0 = solo only

	// Feature flags — applied only to Studio Owners
	AnalyticsAdvanced bool `gorm:"column:analytics_advanced" json:"analytics_advanced"`
	PrioritySearch    bool `gorm:"column:priority_search" json:"priority_search"`
	PrioritySupport   bool `gorm:"column:priority_support" json:"priority_support"`
	CRMAccess         bool `gorm:"column:crm_access" json:"crm_access"`

	IsActive  bool      `gorm:"column:is_active" json:"is_active"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
}

func (Plan) TableName() string { return "subscription_plans" }

// Subscription tracks an active plan for a Studio Owner.
// owner_id always references a user with role='owner'.
type Subscription struct {
	ID              string         `gorm:"column:id;primaryKey" json:"id"`
	OwnerID         int64          `gorm:"column:owner_id" json:"owner_id"`
	PlanID          PlanID         `gorm:"column:plan_id" json:"plan_id"`
	Status          Status         `gorm:"column:status" json:"status"`
	BillingPeriod   BillingPeriod  `gorm:"column:billing_period" json:"billing_period"`
	StartedAt       time.Time      `gorm:"column:started_at" json:"started_at"`
	ExpiresAt       sql.NullTime   `gorm:"column:expires_at" json:"expires_at,omitempty"`
	AutoRenew       bool           `gorm:"column:auto_renew" json:"auto_renew"`
	CancelReason    sql.NullString `gorm:"column:cancel_reason" json:"cancel_reason,omitempty"`
	CancelledAt     sql.NullTime   `gorm:"column:cancelled_at" json:"cancelled_at,omitempty"`
	PaymentMethodID sql.NullString `gorm:"column:payment_method_id" json:"payment_method_id,omitempty"`
	CreatedAt       time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt       time.Time      `gorm:"column:updated_at" json:"updated_at"`
}

func (Subscription) TableName() string { return "subscriptions" }

// IsExpired checks if the subscription has passed its expiry date
func (s *Subscription) IsExpired() bool {
	if !s.ExpiresAt.Valid {
		return false
	}
	return time.Now().After(s.ExpiresAt.Time)
}

// IsActive checks if subscription is currently usable
func (s *Subscription) IsActive() bool {
	return s.Status == StatusActive && !s.IsExpired()
}

// DaysRemaining returns days until expiry (-1 = unlimited)
func (s *Subscription) DaysRemaining() int {
	if !s.ExpiresAt.Valid {
		return -1
	}
	remaining := time.Until(s.ExpiresAt.Time)
	if remaining < 0 {
		return 0
	}
	return int(remaining.Hours() / 24)
}
