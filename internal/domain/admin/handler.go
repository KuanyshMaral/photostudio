package admin

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"photostudio/internal/pkg/chicontext"
	"photostudio/internal/pkg/response"
)

type Handler struct {
	service           *Service
	authHandler       *AuthHandler
	managementHandler *ManagementHandler
}

// NewHandler creates a new admin handler
func NewHandler(service *Service, authHandler *AuthHandler, managementHandler *ManagementHandler) *Handler {
	return &Handler{
		service:           service,
		authHandler:       authHandler,
		managementHandler: managementHandler,
	}
}

// GetPendingStudios
//
//	@Summary		Get pending studios
//	@Description	Get a paginated list of studio owners awaiting verification
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			page	query		int	false	"Page number"	default(1)
//	@Param			limit	query		int	false	"Page size"		default(20)
//	@Success		200		{object}	map[string]interface{}
//	@Failure		403		{object}	map[string]interface{}
//	@Failure		500		{object}	map[string]interface{}
//	@Security		BearerAuth
//	@Router			/admin/studios/pending [get]
func (h *Handler) GetPendingStudios(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		response.CustomError(w, r, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	page := parseIntDefault(r.URL.Query().Get("page"), 1)
	limit := parseIntDefault(r.URL.Query().Get("limit"), 20)

	owners, total, err := h.service.GetPendingStudioOwners(r.Context(), page, limit)
	if err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "FETCH_ERROR", err)
		return
	}

	response.Success(w, http.StatusOK, response.H{"pending_studios": owners, "count": total})
}

// ApproveStudio
//
//	@Summary		Approve studio (Legacy)
//	@Description	Approve a studio owner
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			id	path		int	true	"Studio ID"
//	@Success		200	{object}	map[string]interface{}
//	@Failure		400	{object}	map[string]interface{}
//	@Failure		401	{object}	map[string]interface{}
//	@Failure		403	{object}	map[string]interface{}
//	@Security		BearerAuth
//	@Router			/admin/studios/{id}/approve [post]
func (h *Handler) ApproveStudio(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		response.CustomError(w, r, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	adminID := chicontext.UserIDFromCtx(r.Context())
	if adminID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	studioOwnerID, err := parseIDParam(r, "id")
	if err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_ID", "Invalid studio owner ID")
		return
	}

	if err := h.service.ApproveStudioOwner(r.Context(), studioOwnerID, adminID); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "APPROVE_ERROR", err)
		return
	}

	response.Success(w, http.StatusOK, response.H{"message": "Studio verified successfully"})
}

// VerifyStudio
//
//	@Summary		Verify a pending studio
//	@Description	Accept the verification application for a studio
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int					true	"Studio ID"
//	@Param			request	body		VerifyStudioRequest	true	"Notes for verification"
//	@Success		200		{object}	map[string]interface{}
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]interface{}
//	@Failure		403		{object}	map[string]interface{}
//	@Failure		500		{object}	map[string]interface{}
//	@Security		BearerAuth
//	@Router			/admin/studios/{id}/verify [post]
func (h *Handler) VerifyStudio(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		response.CustomError(w, r, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	adminID := chicontext.UserIDFromCtx(r.Context())
	if adminID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	studioID, err := parseIDParam(r, "id")
	if err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_ID", "Invalid studio ID")
		return
	}

	var req VerifyStudioRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err)
		return
	}

	log.Printf("admin action: VerifyStudio admin_id=%d studio_id=%d notes=%q", adminID, studioID, req.AdminNotes)

	studio, err := h.service.VerifyStudio(r.Context(), studioID, adminID, req.AdminNotes)
	if err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", err)
		return
	}

	response.Success(w, http.StatusOK, studio)
}

// RejectStudio
//
//	@Summary		Reject a pending studio
//	@Description	Reject the verification application for a studio
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int					true	"Studio Owner ID"
//	@Param			request	body		RejectStudioRequest	true	"Reason for rejection"
//	@Success		200		{object}	map[string]interface{}
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		401		{object}	map[string]interface{}
//	@Failure		403		{object}	map[string]interface{}
//	@Security		BearerAuth
//	@Router			/admin/studios/{id}/reject [post]
func (h *Handler) RejectStudio(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		response.CustomError(w, r, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	adminID := chicontext.UserIDFromCtx(r.Context())
	if adminID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	studioOwnerID, err := parseIDParam(r, "id")
	if err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_ID", "Invalid studio owner ID")
		return
	}

	var req RejectStudioRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Reason is required")
		return
	}

	if err := h.service.RejectStudioOwner(r.Context(), studioOwnerID, adminID, req.Reason); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "REJECT_ERROR", err)
		return
	}

	response.Success(w, http.StatusOK, response.H{"message": "Application rejected"})
}

// GetStatistics
//
//	@Summary		Get admin statistics
//	@Description	Get summary statistics for the admin dashboard
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string]interface{}
//	@Failure		403	{object}	map[string]interface{}
//	@Failure		500	{object}	map[string]interface{}
//	@Security		BearerAuth
//	@Router			/admin/statistics [get]
func (h *Handler) GetStatistics(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		response.CustomError(w, r, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	stats, err := h.service.GetStatistics(r.Context())
	if err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", err)
		return
	}

	response.Success(w, http.StatusOK, stats)
}

// GetStats
//
//	@Summary		Get platform stats
//	@Description	Get platform statistics
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string]interface{}
//	@Failure		403	{object}	map[string]interface{}
//	@Failure		500	{object}	map[string]interface{}
//	@Security		BearerAuth
//	@Router			/admin/stats [get]
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		response.CustomError(w, r, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	stats, err := h.service.GetPlatformStats(r.Context())
	if err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", err)
		return
	}

	response.Success(w, http.StatusOK, stats)
}

// BlockUser
//
//	@Summary		Block a user
//	@Description	Block a user from the platform
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int					true	"User ID"
//	@Param			request	body		BlockUserRequest	true	"Reason for blocking"
//	@Success		200		{object}	map[string]interface{}
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		403		{object}	map[string]interface{}
//	@Failure		500		{object}	map[string]interface{}
//	@Security		BearerAuth
//	@Router			/admin/users/{id}/block [post]
func (h *Handler) BlockUser(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		response.CustomError(w, r, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	userID, err := parseIDParam(r, "id")
	if err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_ID", "Invalid user ID")
		return
	}

	var req BlockUserRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err)
		return
	}

	log.Printf("admin action: BlockUser user_id=%d reason=%q", userID, req.Reason)

	u, err := h.service.BlockUser(r.Context(), userID, req.Reason)
	if err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", err)
		return
	}

	response.Success(w, http.StatusOK, u)
}

// UnblockUser
//
//	@Summary		Unblock a user
//	@Description	Unblock a previously blocked user
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			id	path		int	true	"User ID"
//	@Success		200	{object}	map[string]interface{}
//	@Failure		400	{object}	map[string]interface{}
//	@Failure		403	{object}	map[string]interface{}
//	@Failure		500	{object}	map[string]interface{}
//	@Security		BearerAuth
//	@Router			/admin/users/{id}/unblock [post]
func (h *Handler) UnblockUser(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		response.CustomError(w, r, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	userID, err := parseIDParam(r, "id")
	if err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_ID", "Invalid user ID")
		return
	}

	log.Printf("admin action: UnblockUser user_id=%d", userID)

	u, err := h.service.UnblockUser(r.Context(), userID)
	if err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", err)
		return
	}

	response.Success(w, http.StatusOK, u)
}

// BanUser
//
//	@Summary		Ban a user
//	@Description	Ban a user from the platform permanently
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int					true	"User ID"
//	@Param			request	body		BlockUserRequest	true	"Reason for banning"
//	@Success		200		{object}	map[string]interface{}
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		403		{object}	map[string]interface{}
//	@Failure		500		{object}	map[string]interface{}
//	@Security		BearerAuth
//	@Router			/admin/users/{id}/ban [post]
func (h *Handler) BanUser(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		response.CustomError(w, r, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	userID, err := parseIDParam(r, "id")
	if err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_ID", "Invalid user ID")
		return
	}

	var req BlockUserRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Reason is required")
		return
	}

	if err := h.service.BanUser(r.Context(), userID, req.Reason); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "BAN_ERROR", err)
		return
	}

	response.Success(w, http.StatusOK, response.H{"message": "User banned"})
}

// UnbanUser
//
//	@Summary		Unban a user
//	@Description	Unban a previously banned user
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			id	path		int	true	"User ID"
//	@Success		200	{object}	map[string]interface{}
//	@Failure		400	{object}	map[string]interface{}
//	@Failure		403	{object}	map[string]interface{}
//	@Failure		500	{object}	map[string]interface{}
//	@Security		BearerAuth
//	@Router			/admin/users/{id}/unban [post]
func (h *Handler) UnbanUser(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		response.CustomError(w, r, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	userID, err := parseIDParam(r, "id")
	if err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_ID", "Invalid user ID")
		return
	}

	if err := h.service.UnbanUser(r.Context(), userID); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "UNBAN_ERROR", err)
		return
	}

	response.Success(w, http.StatusOK, response.H{"message": "User unbanned"})
}

// GetUsers
//
//	@Summary		List users
//	@Description	Get a paginated list of users
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			page	query		int		false	"Page number"	default(1)
//	@Param			limit	query		int		false	"Page size"		default(20)
//	@Param			role	query		string	false	"Filter by role"
//	@Param			search	query		string	false	"Search query"
//	@Success		200		{object}	map[string]interface{}
//	@Failure		403		{object}	map[string]interface{}
//	@Failure		500		{object}	map[string]interface{}
//	@Security		BearerAuth
//	@Router			/admin/users [get]
func (h *Handler) GetUsers(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		response.CustomError(w, r, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	page := parseIntDefault(r.URL.Query().Get("page"), 1)
	limit := parseIntDefault(r.URL.Query().Get("limit"), 20)

	filter := UserListFilter{
		Role:  r.URL.Query().Get("role"),
		Query: r.URL.Query().Get("search"),
	}

	log.Printf("admin action: GetUsers page=%d limit=%d", page, limit)

	users, total, err := h.service.ListUsers(r.Context(), filter, page, limit)
	if err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", err)
		return
	}

	response.Success(w, http.StatusOK, UserListResponse{Users: users, Total: total, Page: page, Limit: limit})
}

// GetReviews
//
//	@Summary		List reviews
//	@Description	Get a paginated list of reviews (for moderation)
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			page	query		int		false	"Page number"	default(1)
//	@Param			limit	query		int		false	"Page size"		default(20)
//	@Param			hidden	query		bool	false	"Filter by hidden status"
//	@Success		200		{object}	map[string]interface{}
//	@Failure		403		{object}	map[string]interface{}
//	@Failure		500		{object}	map[string]interface{}
//	@Security		BearerAuth
//	@Router			/admin/reviews [get]
func (h *Handler) GetReviews(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		response.CustomError(w, r, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	page := parseIntDefault(r.URL.Query().Get("page"), 1)
	limit := parseIntDefault(r.URL.Query().Get("limit"), 20)

	hiddenVal := r.URL.Query().Get("hidden") == "true"
	filter := ReviewListFilter{
		Hidden: &hiddenVal,
	}

	log.Printf("admin action: GetReviews page=%d limit=%d", page, limit)

	reviews, total, err := h.service.ListReviews(r.Context(), filter, page, limit)
	if err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", err)
		return
	}

	response.Success(w, http.StatusOK, ReviewListResponse{Reviews: reviews, Total: total, Page: page, Limit: limit})
}

// HideReview
//
//	@Summary		Hide review
//	@Description	Hide a review from public view
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			id	path		int	true	"Review ID"
//	@Success		200	{object}	map[string]interface{}
//	@Failure		400	{object}	map[string]interface{}
//	@Failure		403	{object}	map[string]interface{}
//	@Failure		500	{object}	map[string]interface{}
//	@Security		BearerAuth
//	@Router			/admin/reviews/{id}/hide [post]
func (h *Handler) HideReview(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		response.CustomError(w, r, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	reviewID, err := parseIDParam(r, "id")
	if err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_ID", "Invalid review ID")
		return
	}

	log.Printf("admin action: HideReview review_id=%d", reviewID)

	rv, err := h.service.HideReview(r.Context(), reviewID)
	if err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", err)
		return
	}

	response.Success(w, http.StatusOK, rv)
}

// ShowReview
//
//	@Summary		Show review
//	@Description	Restore a previously hidden review
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			id	path		int	true	"Review ID"
//	@Success		200	{object}	map[string]interface{}
//	@Failure		400	{object}	map[string]interface{}
//	@Failure		403	{object}	map[string]interface{}
//	@Failure		500	{object}	map[string]interface{}
//	@Security		BearerAuth
//	@Router			/admin/reviews/{id}/show [post]
func (h *Handler) ShowReview(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		response.CustomError(w, r, http.StatusForbidden, "FORBIDDEN", "Admin access required")
		return
	}

	reviewID, err := parseIDParam(r, "id")
	if err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_ID", "Invalid review ID")
		return
	}

	log.Printf("admin action: ShowReview review_id=%d", reviewID)

	rv, err := h.service.ShowReview(r.Context(), reviewID)
	if err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", err)
		return
	}

	response.Success(w, http.StatusOK, rv)
}

// GetPlatformAnalytics
//
//	@Summary		Get platform analytics
//	@Description	Get analytical data for the platform
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			days	query		int	false	"Days back"	default(30)
//	@Success		200		{object}	map[string]interface{}
//	@Failure		500		{object}	map[string]interface{}
//	@Security		BearerAuth
//	@Router			/admin/analytics [get]
func (h *Handler) GetPlatformAnalytics(w http.ResponseWriter, r *http.Request) {
	daysBack := 30
	if d := r.URL.Query().Get("days"); d != "" {
		if v, err := strconv.Atoi(d); err == nil && v > 0 && v <= 365 {
			daysBack = v
		}
	}

	analytics, err := h.service.GetPlatformAnalytics(r.Context(), daysBack)
	if err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "ANALYTICS_FAILED", err)
		return
	}

	response.Success(w, http.StatusOK, response.H{"analytics": analytics})
}

// SetStudioVIP
//
//	@Summary		Set studio VIP status
//	@Description	Update the VIP status of a studio
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int						true	"Studio ID"
//	@Param			body	body		map[string]interface{}	true	"Status (is_vip: bool)"
//	@Success		200		{object}	map[string]interface{}
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		500		{object}	map[string]interface{}
//	@Security		BearerAuth
//	@Router			/admin/studios/{id}/vip [patch]
func (h *Handler) SetStudioVIP(w http.ResponseWriter, r *http.Request) {
	studioID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	var req struct {
		IsVIP bool `json:"is_vip"`
	}
	if err := response.BindJSON(r, &req); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_REQUEST", err)
		return
	}

	if err := h.service.SetStudioVIP(r.Context(), studioID, req.IsVIP); err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "UPDATE_FAILED", err)
		return
	}

	response.Success(w, http.StatusOK, response.H{"message": "VIP status updated"})
}

// SetStudioGold
//
//	@Summary		Set studio Gold status
//	@Description	Update the Gold status of a studio
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int						true	"Studio ID"
//	@Param			body	body		map[string]interface{}	true	"Status (is_gold: bool)"
//	@Success		200		{object}	map[string]interface{}
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		500		{object}	map[string]interface{}
//	@Security		BearerAuth
//	@Router			/admin/studios/{id}/gold [patch]
func (h *Handler) SetStudioGold(w http.ResponseWriter, r *http.Request) {
	studioID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	var req struct {
		IsGold bool `json:"is_gold"`
	}
	if err := response.BindJSON(r, &req); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_REQUEST", err)
		return
	}

	if err := h.service.SetStudioGold(r.Context(), studioID, req.IsGold); err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "UPDATE_FAILED", err)
		return
	}

	response.Success(w, http.StatusOK, response.H{"message": "Gold status updated"})
}

// SetStudioPromo
//
//	@Summary		Set studio Promo status
//	@Description	Update the Promo status of a studio
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int						true	"Studio ID"
//	@Param			body	body		map[string]interface{}	true	"Status (in_promo_slider: bool)"
//	@Success		200		{object}	map[string]interface{}
//	@Failure		400		{object}	map[string]interface{}
//	@Failure		500		{object}	map[string]interface{}
//	@Security		BearerAuth
//	@Router			/admin/studios/{id}/promo [patch]
func (h *Handler) SetStudioPromo(w http.ResponseWriter, r *http.Request) {
	studioID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	var req struct {
		InPromo bool `json:"in_promo_slider"`
	}
	if err := response.BindJSON(r, &req); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_REQUEST", err)
		return
	}

	if err := h.service.SetStudioPromo(r.Context(), studioID, req.InPromo); err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "UPDATE_FAILED", err)
		return
	}

	response.Success(w, http.StatusOK, response.H{"message": "Promo status updated"})
}

// GetAds
//
//	@Summary		Get ads
//	@Description	Get a list of ads for the platform
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			placement	query		string	false	"Ad placement"
//	@Param			active_only	query		bool	false	"Show active only"
//	@Success		200			{object}	map[string]interface{}
//	@Failure		500			{object}	map[string]interface{}
//	@Security		BearerAuth
//	@Router			/admin/ads [get]
func (h *Handler) GetAds(w http.ResponseWriter, r *http.Request) {
	placement := r.URL.Query().Get("placement")
	activeOnly := r.URL.Query().Get("active_only") == "true"

	ads, err := h.service.GetAds(r.Context(), placement, activeOnly)
	if err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "FETCH_FAILED", err)
		return
	}

	response.Success(w, http.StatusOK, response.H{"ads": ads})
}

// CreateAd
//
//	@Summary		Create ad
//	@Description	Create a new ad on the platform
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			ad	body		Ad	true	"Ad fields"
//	@Success		201	{object}	map[string]interface{}
//	@Failure		400	{object}	map[string]interface{}
//	@Failure		500	{object}	map[string]interface{}
//	@Security		BearerAuth
//	@Router			/admin/ads [post]
func (h *Handler) CreateAd(w http.ResponseWriter, r *http.Request) {
	var ad Ad
	if err := response.BindJSON(r, &ad); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_REQUEST", err)
		return
	}

	if err := h.service.CreateAd(r.Context(), &ad); err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "CREATE_FAILED", err)
		return
	}

	response.Success(w, http.StatusCreated, response.H{"ad": ad})
}

// UpdateAd
//
//	@Summary		Update ad
//	@Description	Update an existing ad
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			id	path		int						true	"Ad ID"
//	@Param			ad	body		map[string]interface{}	true	"Updating fields"
//	@Success		200	{object}	map[string]interface{}
//	@Failure		400	{object}	map[string]interface{}
//	@Failure		404	{object}	map[string]interface{}
//	@Failure		500	{object}	map[string]interface{}
//	@Security		BearerAuth
//	@Router			/admin/ads/{id} [patch]
func (h *Handler) UpdateAd(w http.ResponseWriter, r *http.Request) {
	rawID := strings.TrimPrefix(strings.TrimSpace(chi.URLParam(r, "id")), ":")
	adID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || adID <= 0 {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_ID", "invalid ad id")
		return
	}

	var updates map[string]interface{}
	if err := response.BindJSON(r, &updates); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_REQUEST", err)
		return
	}

	if err := h.service.UpdateAd(r.Context(), adID, updates); err != nil {
		switch err.Error() {
		case "ad not found":
			response.CustomError(w, r, http.StatusNotFound, "NOT_FOUND", err)
		case "no valid fields to update":
			response.CustomError(w, r, http.StatusBadRequest, "INVALID_REQUEST", err)
		default:
			response.CustomError(w, r, http.StatusInternalServerError, "UPDATE_FAILED", err)
		}
		return
	}

	response.Success(w, http.StatusOK, response.H{"message": "Ad updated"})
}

// DeleteAd
//
//	@Summary		Delete ad
//	@Description	Delete an ad by ID
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			id	path		int	true	"Ad ID"
//	@Success		200	{object}	map[string]interface{}
//	@Failure		500	{object}	map[string]interface{}
//	@Security		BearerAuth
//	@Router			/admin/ads/{id} [delete]
func (h *Handler) DeleteAd(w http.ResponseWriter, r *http.Request) {
	rawID := strings.TrimPrefix(strings.TrimSpace(chi.URLParam(r, "id")), ":")
	adID, _ := strconv.ParseInt(rawID, 10, 64)

	if err := h.service.DeleteAd(r.Context(), adID); err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "DELETE_FAILED", err)
		return
	}

	response.Success(w, http.StatusOK, response.H{"message": "Ad deleted"})
}

// DeleteReview
//
//	@Summary		Delete review
//	@Description	Permanently delete a review by ID
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			id	path		int	true	"Review ID"
//	@Success		200	{object}	map[string]interface{}
//	@Failure		500	{object}	map[string]interface{}
//	@Security		BearerAuth
//	@Router			/admin/reviews/{id} [delete]
func (h *Handler) DeleteReview(w http.ResponseWriter, r *http.Request) {
	reviewID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	if err := h.service.DeleteReview(r.Context(), reviewID); err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "DELETE_FAILED", err)
		return
	}

	response.Success(w, http.StatusOK, response.H{"message": "Review deleted"})
}

// -------------------- helpers --------------------

func isAdmin(r *http.Request) bool {
	if adminID := chicontext.AdminIDFromCtx(r.Context()); adminID != "" {
		return true
	}
	role := chicontext.RoleFromCtx(r.Context())
	switch role {
	case "admin", "super_admin", "support", "moderator":
		return true
	default:
		return false
	}
}

func parseIDParam(r *http.Request, name string) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, name), 10, 64)
}

func parseIntDefault(v string, def int) int {
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return def
	}
	return n
}
