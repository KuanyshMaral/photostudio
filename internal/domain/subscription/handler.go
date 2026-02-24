package subscription

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests for subscription management.
// All routes require role='owner' — clients cannot access any of these endpoints.
type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// GetPlans godoc
// @Summary List all subscription plans
// @Description Returns all available plans. Public endpoint — no auth required.
// @Tags Subscriptions
// @Produce json
// @Success 200 {array} PlanResponse
// @Router /subscriptions/plans [get]
func (h *Handler) GetPlans(c *gin.Context) {
	plans, err := h.service.GetPlans(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "failed to load plans"})
		return
	}

	resp := make([]PlanResponse, 0, len(plans))
	for _, p := range plans {
		resp = append(resp, planToResponse(p))
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": resp})
}

// GetMySubscription godoc
// @Summary Get current subscription for the authenticated Studio Owner
// @Tags Subscriptions
// @Security BearerAuth
// @Produce json
// @Success 200 {object} SubscriptionResponse
// @Router /owner/subscription [get]
func (h *Handler) GetMySubscription(c *gin.Context) {
	ownerID := mustOwnerID(c)
	if ownerID == 0 {
		return
	}

	sub, plan, err := h.service.GetCurrentSubscription(c.Request.Context(), ownerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	resp := buildSubscriptionResponse(sub, plan)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": resp})
}

// Subscribe godoc
// @Summary Subscribe or upgrade to a plan
// @Tags Subscriptions
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body SubscribeRequest true "Plan and billing period"
// @Success 201 {object} SubscriptionResponse
// @Router /owner/subscription [post]
func (h *Handler) Subscribe(c *gin.Context) {
	ownerID := mustOwnerID(c)
	if ownerID == 0 {
		return
	}

	var req SubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	sub, err := h.service.Subscribe(c.Request.Context(), ownerID, &req)
	if err != nil {
		switch err {
		case ErrPlanNotFound:
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": err.Error()})
		case ErrAlreadySubscribed:
			c.JSON(http.StatusConflict, gin.H{"success": false, "error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		}
		return
	}

	plan, _ := h.service.GetPlan(c.Request.Context(), ownerID)
	resp := buildSubscriptionResponse(sub, plan)
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": resp})
}

// Cancel godoc
// @Summary Cancel current subscription
// @Tags Subscriptions
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body CancelRequest false "Optional cancel reason"
// @Success 200 {object} map[string]interface{}
// @Router /owner/subscription/cancel [post]
func (h *Handler) Cancel(c *gin.Context) {
	ownerID := mustOwnerID(c)
	if ownerID == 0 {
		return
	}

	var req CancelRequest
	_ = c.ShouldBindJSON(&req)

	if err := h.service.Cancel(c.Request.Context(), ownerID, req.Reason); err != nil {
		switch err {
		case ErrSubscriptionNotFound:
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": err.Error()})
		case ErrCannotCancelFree:
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "subscription cancelled"})
}

// GetUsage godoc
// @Summary Get current usage vs plan limits for the Studio Owner
// @Tags Subscriptions
// @Security BearerAuth
// @Produce json
// @Success 200 {object} UsageResponse
// @Router /owner/subscription/usage [get]
func (h *Handler) GetUsage(c *gin.Context) {
	ownerID := mustOwnerID(c)
	if ownerID == 0 {
		return
	}

	usage, err := h.service.GetUsage(c.Request.Context(), ownerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": usage})
}

// mustOwnerID extracts the owner's user ID from the JWT context.
// Returns 0 and writes 401 if not found.
func mustOwnerID(c *gin.Context) int64 {
	id, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "unauthorized"})
		return 0
	}
	switch v := id.(type) {
	case int64:
		return v
	case float64:
		return int64(v)
	}
	c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "invalid user id"})
	return 0
}

func buildSubscriptionResponse(sub *Subscription, plan *Plan) SubscriptionResponse {
	resp := SubscriptionResponse{
		ID:            sub.ID,
		PlanID:        string(sub.PlanID),
		Status:        string(sub.Status),
		BillingPeriod: string(sub.BillingPeriod),
		StartedAt:     sub.StartedAt.Format("2006-01-02T15:04:05Z"),
		AutoRenew:     sub.AutoRenew,
		DaysRemaining: sub.DaysRemaining(),
	}
	if sub.ExpiresAt.Valid {
		s := sub.ExpiresAt.Time.Format("2006-01-02T15:04:05Z")
		resp.ExpiresAt = &s
	}
	if plan != nil {
		resp.PlanName = plan.Name
		resp.Limits = PlanLimits{
			MaxRooms:         plan.MaxRooms,
			MaxPhotosPerRoom: plan.MaxPhotosPerRoom,
			MaxTeamMembers:   plan.MaxTeamMembers,
		}
		resp.Features = PlanFeatures{
			AnalyticsAdvanced: plan.AnalyticsAdvanced,
			PrioritySearch:    plan.PrioritySearch,
			PrioritySupport:   plan.PrioritySupport,
			CRMAccess:         plan.CRMAccess,
		}
	}
	return resp
}
