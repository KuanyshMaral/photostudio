package subscription

import (
	"net/http"

	"photostudio/internal/pkg/chicontext"
	"photostudio/internal/pkg/response"
)

// Handler handles HTTP requests for subscription management.
// All owner routes require role='owner' — clients cannot access any of these endpoints.
type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// swaggerListPlansResponse is a wrapper strictly for generating Swagger documentation.
type swaggerListPlansResponse struct {
	Success bool           `json:"success"`
	Data    []PlanResponse `json:"data"`
}

// swaggerSubscriptionResponseWrapper is a wrapper strictly for generating Swagger documentation.
type swaggerSubscriptionResponseWrapper struct {
	Success bool                 `json:"success"`
	Data    SubscriptionResponse `json:"data"`
}

// swaggerUsageResponseWrapper is a wrapper strictly for generating Swagger documentation.
type swaggerUsageResponseWrapper struct {
	Success bool          `json:"success"`
	Data    UsageResponse `json:"data"`
}

// GetPlans получение списка доступных тарифов.
//
//	@Summary		Список тарифов
//	@Description	Возвращает все доступные планы подписки. Эндпоинт публичный, не требует авторизации.
//	@Tags			Subscriptions
//	@Produce		json
//	@Success		200	{object}	swaggerListPlansResponse	"Список тарифов"
//	@Failure		500	{object}	response.ErrorResponse		"Внутренняя ошибка сервера"
//	@Router			/subscriptions/plans [get]
func (h *Handler) GetPlans(w http.ResponseWriter, r *http.Request) {
	plans, err := h.service.GetPlans(r.Context())
	if err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "FETCH_FAILED", "failed to load plans")
		return
	}

	resp := make([]PlanResponse, 0, len(plans))
	for _, p := range plans {
		resp = append(resp, planToResponse(p))
	}
	response.JSON(w, http.StatusOK, response.H{"success": true, "data": resp})
}

// GetMySubscription получение текущей подписки.
//
//	@Summary		Текущая подписка
//	@Description	Возвращает активную подписку владельца студии со всеми лимитами и фичами тарифа.
//	@Tags			Subscriptions
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	swaggerSubscriptionResponseWrapper	"Подписка"
//	@Failure		401	{object}	response.ErrorResponse				"Не авторизован"
//	@Failure		500	{object}	response.ErrorResponse				"Внутренняя ошибка сервера"
//	@Router			/owner/subscription [get]
func (h *Handler) GetMySubscription(w http.ResponseWriter, r *http.Request) {
	ownerID := chicontext.UserIDFromCtx(r.Context())
	if ownerID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}

	sub, plan, err := h.service.GetCurrentSubscription(r.Context(), ownerID)
	if err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "FETCH_FAILED", err.Error())
		return
	}

	response.JSON(w, http.StatusOK, response.H{"success": true, "data": buildSubscriptionResponse(sub, plan)})
}

// Subscribe оформление или обновление подписки.
//
//	@Summary		Оформить или изменить подписку
//	@Description	Владелец переходит на указанный план. Если план платный - создается подписка (возможно, потребуется оплата).
//	@Tags			Subscriptions
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		SubscribeRequest					true	"Данные (ID тарифа и период)"
//	@Success		201		{object}	swaggerSubscriptionResponseWrapper	"Подписка оформлена"
//	@Failure		400		{object}	response.ErrorResponse				"Неверный запрос"
//	@Failure		401		{object}	response.ErrorResponse				"Не авторизован"
//	@Failure		404		{object}	response.ErrorResponse				"Тариф не найден"
//	@Failure		409		{object}	response.ErrorResponse				"Уже подписан на этот тариф"
//	@Failure		500		{object}	response.ErrorResponse				"Внутренняя ошибка сервера"
//	@Router			/owner/subscription [post]
func (h *Handler) Subscribe(w http.ResponseWriter, r *http.Request) {
	ownerID := chicontext.UserIDFromCtx(r.Context())
	if ownerID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}

	var req SubscribeRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}

	sub, err := h.service.Subscribe(r.Context(), ownerID, &req)
	if err != nil {
		switch err {
		case ErrPlanNotFound:
			response.CustomError(w, r, http.StatusNotFound, "PLAN_NOT_FOUND", err.Error())
		case ErrAlreadySubscribed:
			response.CustomError(w, r, http.StatusConflict, "ALREADY_SUBSCRIBED", err.Error())
		default:
			response.CustomError(w, r, http.StatusInternalServerError, "SUBSCRIBE_FAILED", err.Error())
		}
		return
	}

	plan, _ := h.service.GetPlan(r.Context(), ownerID)
	response.JSON(w, http.StatusCreated, response.H{"success": true, "data": buildSubscriptionResponse(sub, plan)})
}

// Cancel отмена автоматического продления.
//
//	@Summary		Отменить подписку
//	@Description	Отменяет текущую платную подписку владельца (автоматическое продление).
//	@Tags			Subscriptions
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		CancelRequest				false	"Причина отмены (необязательно)"
//	@Success		200		{object}	response.SuccessResponse	"Подписка отменена"
//	@Failure		400		{object}	response.ErrorResponse		"Неверный запрос (например, нельзя отменить бесплатный тариф)"
//	@Failure		401		{object}	response.ErrorResponse		"Не авторизован"
//	@Failure		404		{object}	response.ErrorResponse		"Подписка не найдена"
//	@Failure		500		{object}	response.ErrorResponse		"Внутренняя ошибка сервера"
//	@Router			/owner/subscription/cancel [post]
func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	ownerID := chicontext.UserIDFromCtx(r.Context())
	if ownerID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}

	var req CancelRequest
	_ = response.BindJSON(r, &req) // optional body

	if err := h.service.Cancel(r.Context(), ownerID, req.Reason); err != nil {
		switch err {
		case ErrSubscriptionNotFound:
			response.CustomError(w, r, http.StatusNotFound, "NOT_FOUND", err.Error())
		case ErrCannotCancelFree:
			response.CustomError(w, r, http.StatusBadRequest, "CANNOT_CANCEL_FREE", err.Error())
		default:
			response.CustomError(w, r, http.StatusInternalServerError, "CANCEL_FAILED", err.Error())
		}
		return
	}

	response.JSON(w, http.StatusOK, response.H{"success": true, "message": "subscription cancelled"})
}

// GetUsage получение текущего использования лимитов.
//
//	@Summary		Использование тарифа
//	@Description	Возвращает текущее количество комнат, фото и мест в команде в сравнении с лимитами тарифа.
//	@Tags			Subscriptions
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	swaggerUsageResponseWrapper	"Текущее использование ресурсов"
//	@Failure		401	{object}	response.ErrorResponse		"Не авторизован"
//	@Failure		500	{object}	response.ErrorResponse		"Внутренняя ошибка сервера"
//	@Router			/owner/subscription/usage [get]
func (h *Handler) GetUsage(w http.ResponseWriter, r *http.Request) {
	ownerID := chicontext.UserIDFromCtx(r.Context())
	if ownerID == 0 {
		response.CustomError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}

	usage, err := h.service.GetUsage(r.Context(), ownerID)
	if err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "FETCH_FAILED", err.Error())
		return
	}

	response.JSON(w, http.StatusOK, response.H{"success": true, "data": usage})
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
