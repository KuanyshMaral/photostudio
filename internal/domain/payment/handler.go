package payment

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"photostudio/internal/pkg/chicontext"
	"photostudio/internal/pkg/response"
)

type Handler struct {
	service *Service
	loggerf func(format string, args ...interface{})
}

func NewHandler(service *Service, loggerf func(format string, args ...interface{})) *Handler {
	if loggerf == nil {
		loggerf = func(string, ...interface{}) {}
	}
	return &Handler{service: service, loggerf: loggerf}
}

// swaggerPaymentSubscriptionResponse is a wrapper strictly for generating Swagger documentation.
type swaggerPaymentSubscriptionResponse struct {
	Success bool `json:"success"`
	Data    any  `json:"data"` // subscription.SubscriptionResponse
}

// CreatePayment создание нового платежа (Robokassa).
//
//	@Summary		Создать платеж
//	@Description	Инициирует процесс оплаты (бронирование, услуга).
//	@Tags			Payment
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		CreatePaymentRequest								true	"Данные платежа"
//	@Success		200		{object}	response.SuccessResponse{data=InitPaymentResponse}	"Ссылка на оплату"
//	@Failure		400		{object}	response.ErrorResponse								"Неверные параметры"
//	@Failure		401		{object}	response.ErrorResponse								"Не авторизован"
//	@Failure		500		{object}	response.ErrorResponse								"Ошибка сервера"
//	@Router			/payments [post]
func (h *Handler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	var req CreatePaymentRequest
	body, _ := io.ReadAll(r.Body)
	r.Body = io.NopCloser(strings.NewReader(string(body)))
	h.loggerf("level=info msg=robokassa create request request_body=%s", string(body))
	if err := response.BindJSON(r, &req); err != nil {
		h.loggerf("level=error msg=invalid robokassa create payload err=%v", err)
		response.JSON(w, http.StatusBadRequest, response.H{"error": err.Error()})
		return
	}
	amount := strings.TrimSpace(req.Amount)
	if amount == "" {
		amount = strings.TrimSpace(req.OutSum)
	}
	if amount == "" {
		response.JSON(w, http.StatusBadRequest, response.H{"error": "amount or out_sum is required"})
		return
	}

	subscriptionID := req.SubscriptionID
	if subscriptionID != nil && strings.EqualFold(strings.TrimSpace(*subscriptionID), "string") {
		subscriptionID = nil
	}

	userID := chicontext.UserIDFromCtx(r.Context())
	resp, err := h.service.CreatePayment(r.Context(), userID, req.BookingID, amount, req.Description, req.ShpParams, req.Recurring, req.PreviousInvoiceID, subscriptionID)
	if err != nil {
		h.loggerf("level=error msg=robokassa create failed request=%+v err=%v", req, err)
		status := http.StatusInternalServerError
		switch err {
		case ErrAmountMismatch, ErrInvalidAmount, ErrInvalidSubscriptionID:
			status = http.StatusBadRequest
		}
		response.JSON(w, status, response.H{"error": err.Error()})
		return
	}
	h.loggerf("level=info msg=robokassa create response response=%+v", resp)
	response.Success(w, http.StatusOK, resp)
}

// InitPayment универсальная инициализация платежа (старый API, если используется).
//
//	@Summary		Инициализировать платеж
//	@Description	Создает платеж в БД и возвращает ссылку.
//	@Tags			Payment
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		InitPaymentRequest									true	"Данные платежа"
//	@Success		200		{object}	response.SuccessResponse{data=InitPaymentResponse}	"Платеж"
//	@Failure		400		{object}	response.ErrorResponse								"Ошибка запроса"
//	@Failure		401		{object}	response.ErrorResponse								"Не авторизован"
//	@Failure		500		{object}	response.ErrorResponse								"Ошибка сервера"
//	@Router			/payments/init [post]
func (h *Handler) InitPayment(w http.ResponseWriter, r *http.Request) {
	var req InitPaymentRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.JSON(w, http.StatusBadRequest, response.H{"error": err.Error()})
		return
	}
	resp, err := h.service.InitPayment(r.Context(), req)
	if err != nil {
		response.JSON(w, http.StatusInternalServerError, response.H{"error": err.Error()})
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

// CreateSubscription создание подписки.
//
//	@Summary		Создать подписку
//	@Description	Инициирует платеж для рекуррентной подписки.
//	@Tags			Payment
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		CreateSubscriptionRequest							true	"Сумма"
//	@Success		200		{object}	response.SuccessResponse{data=InitPaymentResponse}	"Ссылка на оплату"
//	@Failure		400		{object}	response.ErrorResponse								"Ошибка запроса"
//	@Failure		401		{object}	response.ErrorResponse								"Не авторизован"
//	@Failure		500		{object}	response.ErrorResponse								"Ошибка сервера"
//	@Router			/payments/subscriptions [post]
func (h *Handler) CreateSubscription(w http.ResponseWriter, r *http.Request) {
	var req CreateSubscriptionRequest
	if err := response.BindJSON(r, &req); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_REQUEST", err)
		return
	}
	userID := chicontext.UserIDFromCtx(r.Context())
	resp, err := h.service.CreateSubscription(r.Context(), userID, req.Amount)
	if err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "SUBSCRIPTION_CREATE_FAILED", err)
		return
	}
	response.Success(w, http.StatusOK, resp)
}

// MySubscription получить статус подписки пользователя.
//
//	@Summary		Моя подписка
//	@Description	Возвращает текущую подписку пользователя.
//	@Tags			Payment
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	swaggerPaymentSubscriptionResponse	"Подписка"
//	@Failure		401	{object}	response.ErrorResponse				"Не авторизован"
//	@Failure		404	{object}	response.ErrorResponse				"Подписка не найдена"
//	@Failure		500	{object}	response.ErrorResponse				"Ошибка сервера"
//	@Router			/payments/subscriptions/my [get]
func (h *Handler) MySubscription(w http.ResponseWriter, r *http.Request) {
	userID := chicontext.UserIDFromCtx(r.Context())
	sub, err := h.service.GetMySubscription(r.Context(), userID)
	if err != nil {
		response.CustomError(w, r, http.StatusNotFound, "SUBSCRIPTION_NOT_FOUND", err)
		return
	}
	response.Success(w, http.StatusOK, sub)
}

// CancelSubscription отменить подписку.
//
//	@Summary		Отменить подписку
//	@Description	Отменяет активную подписку пользователя (отмена рекуррента).
//	@Tags			Payment
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.SuccessResponse	"Готово"
//	@Failure		400	{object}	response.ErrorResponse		"Ошибка отмены"
//	@Failure		401	{object}	response.ErrorResponse		"Не авторизован"
//	@Failure		500	{object}	response.ErrorResponse		"Ошибка сервера"
//	@Router			/payments/subscriptions/my/cancel [post]
func (h *Handler) CancelSubscription(w http.ResponseWriter, r *http.Request) {
	userID := chicontext.UserIDFromCtx(r.Context())
	if err := h.service.CancelSubscription(r.Context(), userID); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "SUBSCRIPTION_CANCEL_FAILED", err)
		return
	}
	response.Success(w, http.StatusOK, response.H{"status": "canceled"})
}

// ResultCallback webhook от Robokassa.
//
//	@Summary		Robokassa Result Webhook
//	@Description	Обрабатывает успешный платеж от Robokassa в фоне. Специфический контент-тип application/x-www-form-urlencoded.
//	@Tags			Payment Webhooks
//	@Accept			x-www-form-urlencoded
//	@Produce		plain
//	@Router			/payments/callback/result [post]
func (h *Handler) ResultCallback(w http.ResponseWriter, r *http.Request) {
	rawBody, _ := io.ReadAll(r.Body)
	r.Body = io.NopCloser(strings.NewReader(string(rawBody)))
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad request"))
		return
	}
	outSum := r.FormValue("OutSum")
	invIDRaw := r.FormValue("InvId")
	signature := r.FormValue("SignatureValue")
	if strings.TrimSpace(outSum) == "" || strings.TrimSpace(invIDRaw) == "" || strings.TrimSpace(signature) == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad request"))
		return
	}
	invID, err := strconv.ParseInt(invIDRaw, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad request"))
		return
	}
	ack, err := h.service.HandleResultCallback(r.Context(), outSum, invID, signature, collectShp(r), string(rawBody))
	if err != nil {
		if err == ErrInvalidSignature || err == ErrAmountMismatch || err == ErrReplayDetected {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte("forbidden"))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
		return
	}
	_, _ = w.Write([]byte(ack))
}

// SuccessCallback webhook перенаправления от Robokassa.
//
//	@Summary		Robokassa Success Redirect
//	@Description	Обрабатывает пользователя, вернувшегося после успешной оплаты.
//	@Tags			Payment Webhooks
//	@Produce		json
//	@Router			/payments/callback/success [get]
func (h *Handler) SuccessCallback(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_REQUEST", err)
		return
	}
	outSum := firstNonEmpty(r.FormValue("OutSum"), r.URL.Query().Get("OutSum"))
	invIDRaw := firstNonEmpty(r.FormValue("InvId"), r.URL.Query().Get("InvId"))
	signature := firstNonEmpty(r.FormValue("SignatureValue"), r.URL.Query().Get("SignatureValue"))
	if strings.TrimSpace(outSum) == "" || strings.TrimSpace(invIDRaw) == "" || strings.TrimSpace(signature) == "" {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "OutSum, InvId and SignatureValue are required")
		return
	}
	invID, err := strconv.ParseInt(invIDRaw, 10, 64)
	if err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "InvId must be an integer")
		return
	}
	ok, err := h.service.HandleSuccessCallback(r.Context(), outSum, invID, signature, collectShp(r), "")
	if err != nil {
		response.CustomError(w, r, http.StatusForbidden, "PAYMENT_VALIDATION_FAILED", err)
		return
	}
	response.Success(w, http.StatusOK, response.H{"validated": ok, "redirect_url": h.service.frontSuccess})
}

// FailCallback webhook ошибки от Robokassa.
//
//	@Summary		Robokassa Fail Redirect
//	@Description	Обрабатывает пользователя, вернувшегося после неудачной оплаты.
//	@Tags			Payment Webhooks
//	@Produce		json
//	@Router			/payments/callback/fail [get]
func (h *Handler) FailCallback(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_REQUEST", err)
		return
	}
	invIDRaw := firstNonEmpty(r.FormValue("InvId"), r.URL.Query().Get("InvId"), r.FormValue("inv_id"), r.URL.Query().Get("inv_id"))
	if strings.TrimSpace(invIDRaw) == "" {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "InvId is required")
		return
	}
	invID, err := strconv.ParseInt(invIDRaw, 10, 64)
	if err != nil {
		response.CustomError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "InvId must be an integer")
		return
	}
	if err := h.service.FailPayment(r.Context(), invID); err != nil {
		response.CustomError(w, r, http.StatusInternalServerError, "PAYMENT_FAIL_UPDATE_FAILED", err)
		return
	}
	response.Success(w, http.StatusOK, response.H{"redirect_url": h.service.frontFail})
}

func collectShp(r *http.Request) map[string]string {
	res := map[string]string{}
	for k, v := range r.Form {
		if strings.HasPrefix(strings.ToLower(k), "shp_") && len(v) > 0 {
			res[trimShpKey(k)] = v[0]
		}
	}
	for k, v := range r.URL.Query() {
		if strings.HasPrefix(strings.ToLower(k), "shp_") && len(v) > 0 {
			res[trimShpKey(k)] = v[0]
		}
	}
	return res
}

func trimShpKey(k string) string {
	if len(k) < 4 {
		return k
	}
	return k[4:]
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
