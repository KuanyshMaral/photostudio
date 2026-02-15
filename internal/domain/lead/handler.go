package lead

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"photostudio/internal/domain/profile"
	"photostudio/internal/pkg/response"
	"photostudio/internal/pkg/validator"
)

// Handler handles lead HTTP requests
type Handler struct {
	service        *Service
	profileService *profile.Service
}

// NewHandler creates lead handler
func NewHandler(service *Service, profileService *profile.Service) *Handler {
	return &Handler{
		service:        service,
		profileService: profileService,
	}
}

// SubmitLead handles POST /api/v1/leads/submit (public)
// @Summary Submit studio owner lead
// @Description Public endpoint for potential studio owners to submit their application
// @Tags Leads
// @Accept json
// @Produce json
// @Param request body SubmitLeadRequest true "Lead submission data"
// @Success 201 {object} response.Response{data=OwnerLead}
// @Failure 400 {object} response.Response
// @Failure 422 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /leads/submit [post]
func (h *Handler) SubmitLead(c *gin.Context) {
	var req SubmitLeadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.CustomError(c, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body")
		return
	}

	if errors := validator.Validate(&req); errors != nil {
		response.CustomError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", errors)
		return
	}

	// Get client IP
	ip := c.ClientIP()
	userAgent := c.Request.UserAgent()

	lead, err := h.service.SubmitLead(c.Request.Context(), &req, ip, userAgent)
	if err != nil {
		if err == ErrEmailExists {
			response.CustomError(c, http.StatusConflict, "EMAIL_EXISTS", "Email already registered")
			return
		}
		response.CustomError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err)
		return
	}

	response.Success(c, http.StatusCreated, lead)
}

// GetLead handles GET /api/v1/admin/leads/:id
// @Summary Get lead by ID
// @Description Admin endpoint to view lead details
// @Tags Admin Leads
// @Produce json
// @Security BearerAuth
// @Param id path int true "Lead ID"
// @Success 200 {object} response.Response{data=OwnerLead}
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/leads/{id} [get]
func (h *Handler) GetLead(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.CustomError(c, http.StatusBadRequest, "INVALID_ID", "Invalid lead ID")
		return
	}

	lead, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == ErrLeadNotFound {
			response.CustomError(c, http.StatusNotFound, "LEAD_NOT_FOUND", "Lead not found")
			return
		}
		response.CustomError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err)
		return
	}

	response.Success(c, http.StatusOK, lead)
}

// ListLeads handles GET /api/v1/admin/leads
// @Summary List leads
// @Description Admin endpoint to list all leads with optional filtering
// @Tags Admin Leads
// @Produce json
// @Security BearerAuth
// @Param status query string false "Filter by status" Enums(new, contacted, qualified, converted, rejected, lost)
// @Param limit query int false "Limit" default(50)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} response.Response{data=LeadListResponse}
// @Failure 500 {object} response.Response
// @Router /admin/leads [get]
func (h *Handler) ListLeads(c *gin.Context) {
	var status *Status
	if s := c.Query("status"); s != "" {
		statusVal := Status(s)
		status = &statusVal
	}

	limit := 50
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}

	offset := 0
	if o := c.Query("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}

	leads, total, err := h.service.ListLeads(c.Request.Context(), status, limit, offset)
	if err != nil {
		response.CustomError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err)
		return
	}

	response.Success(c, http.StatusOK, LeadListResponse{
		Leads: convertLeads(leads),
		Total: total,
	})
}

// UpdateStatus handles PATCH /api/v1/admin/leads/:id/status
// @Summary Update lead status
// @Description Admin endpoint to update lead status
// @Tags Admin Leads
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Lead ID"
// @Param request body UpdateLeadStatusRequest true "Status update"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 422 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/leads/{id}/status [patch]
func (h *Handler) UpdateStatus(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.CustomError(c, http.StatusBadRequest, "INVALID_ID", "Invalid lead ID")
		return
	}

	var req UpdateLeadStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.CustomError(c, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body")
		return
	}

	if errors := validator.Validate(&req); errors != nil {
		response.CustomError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", errors)
		return
	}

	if err := h.service.UpdateStatus(c.Request.Context(), id, req.Status, req.Notes, req.Reason); err != nil {
		if err == ErrLeadNotFound {
			response.CustomError(c, http.StatusNotFound, "LEAD_NOT_FOUND", "Lead not found")
			return
		}
		if err == ErrAlreadyConverted {
			response.CustomError(c, http.StatusConflict, "ALREADY_CONVERTED", "Lead already converted")
			return
		}
		response.CustomError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err)
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Status updated"})
}

// AssignLead handles PATCH /api/v1/admin/leads/:id/assign
// @Summary Assign lead to admin
// @Description Admin endpoint to assign lead to an admin user
// @Tags Admin Leads
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Lead ID"
// @Param request body AssignLeadRequest true "Assignment data"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 422 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/leads/{id}/assign [patch]
func (h *Handler) AssignLead(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.CustomError(c, http.StatusBadRequest, "INVALID_ID", "Invalid lead ID")
		return
	}

	var req AssignLeadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.CustomError(c, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body")
		return
	}

	if errors := validator.Validate(&req); errors != nil {
		response.CustomError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", errors)
		return
	}

	if err := h.service.Assign(c.Request.Context(), id, req.AdminID, req.Priority); err != nil {
		if err == ErrLeadNotFound {
			response.CustomError(c, http.StatusNotFound, "LEAD_NOT_FOUND", "Lead not found")
			return
		}
		response.CustomError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err)
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Lead assigned"})
}

// RejectLead handles POST /admin/leads/:id/reject
// @Summary Reject lead
// @Description Mark lead as rejected with reason
// @Tags Admin Leads
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Lead ID"
// @Param request body UpdateLeadStatusRequest true "Reason"
// @Success 200 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /admin/leads/{id}/reject [post]
func (h *Handler) RejectLead(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.CustomError(c, http.StatusBadRequest, "INVALID_ID", "Invalid lead ID")
		return
	}

	var req UpdateLeadStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.CustomError(c, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body")
		return
	}

	if err := h.service.RejectLead(c.Request.Context(), id, req.Reason); err != nil {
		if err == ErrLeadNotFound {
			response.CustomError(c, http.StatusNotFound, "LEAD_NOT_FOUND", "Lead not found")
			return
		}
		if err == ErrAlreadyConverted {
			response.CustomError(c, http.StatusBadRequest, "ALREADY_CONVERTED", "Lead already converted")
			return
		}
		response.CustomError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err)
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Lead rejected"})
}

// MarkContacted handles POST /admin/leads/:id/contacted
// @Summary Mark lead as contacted
// @Description Update lead status to contacted and increment follow-up count
// @Tags Admin Leads
// @Produce json
// @Security BearerAuth
// @Param id path int true "Lead ID"
// @Success 200 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /admin/leads/{id}/contacted [post]
func (h *Handler) MarkContacted(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.CustomError(c, http.StatusBadRequest, "INVALID_ID", "Invalid lead ID")
		return
	}

	if err := h.service.MarkContacted(c.Request.Context(), id); err != nil {
		if err == ErrLeadNotFound {
			response.CustomError(c, http.StatusNotFound, "LEAD_NOT_FOUND", "Lead not found")
			return
		}
		response.CustomError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err)
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "Lead marked as contacted"})
}

// ConvertLead handles POST /admin/leads/:id/convert
// @Summary Convert lead to owner
// @Description Create studio owner account from lead
// @Tags Admin Leads
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Lead ID"
// @Param request body ConvertLeadRequest true "Conversion data"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /admin/leads/{id}/convert [post]
func (h *Handler) ConvertLead(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.CustomError(c, http.StatusBadRequest, "INVALID_ID", "Invalid lead ID")
		return
	}

	var req ConvertLeadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.CustomError(c, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body")
		return
	}

	if errors := validator.Validate(&req); errors != nil {
		response.CustomError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", errors)
		return
	}

	user, err := h.service.ConvertLead(c.Request.Context(), id, &req)
	if err != nil {
		switch err {
		case ErrLeadNotFound:
			response.CustomError(c, http.StatusNotFound, "LEAD_NOT_FOUND", "Lead not found")
		case ErrAlreadyConverted:
			response.CustomError(c, http.StatusBadRequest, "ALREADY_CONVERTED", "Lead already converted")
		case ErrCannotConvert:
			response.CustomError(c, http.StatusBadRequest, "CANNOT_CONVERT", "Lead must be qualified or contacted")
		case ErrEmailExists:
			response.CustomError(c, http.StatusConflict, "EMAIL_EXISTS", "User with this email already exists")
		default:
			response.CustomError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err)
		}
		return
	}

	// Create owner profile from lead data
	leadDetails, _ := h.service.GetByID(c.Request.Context(), id)
	if leadDetails != nil {
		ownerProfileReq := &profile.CreateOwnerProfileRequest{
			CompanyName:     leadDetails.CompanyName,
			Bin:             getValue(leadDetails.Bin),
			LegalAddress:    req.LegalAddress,
			ContactPerson:   leadDetails.ContactName,
			ContactPosition: getValue(leadDetails.ContactPosition),
			Phone:           leadDetails.ContactPhone,
			Email:           leadDetails.ContactEmail,
			Website:         getValue(leadDetails.Website),
		}

		_, _ = h.profileService.EnsureOwnerProfile(c.Request.Context(), user.ID, ownerProfileReq)
	}

	response.Success(c, http.StatusOK, gin.H{
		"message": "Lead converted successfully and profile created",
		"user_id": user.ID,
		"email":   user.Email,
	})
}

// Helper to get string value from sql.NullString
func getValue(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

// GetStats handles GET /api/v1/admin/leads/stats
// @Summary Get lead statistics
// @Description Admin endpoint to get lead counts by status
// @Tags Admin Leads
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/leads/stats [get]
func (h *Handler) GetStats(c *gin.Context) {
	stats, err := h.service.GetStats(c.Request.Context())
	if err != nil {
		response.CustomError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err)
		return
	}

	response.Success(c, http.StatusOK, stats)
}

// Helper to convert pointer slice to value slice
func convertLeads(leads []*OwnerLead) []OwnerLead {
	result := make([]OwnerLead, len(leads))
	for i, lead := range leads {
		result[i] = *lead
	}
	return result
}
