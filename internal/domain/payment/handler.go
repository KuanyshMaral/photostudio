package payment

import (
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"photostudio/internal/pkg/response"
	"strconv"
	"strings"
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

func (h *Handler) CreatePayment(c *gin.Context) {
	var req CreatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.CustomError(c, http.StatusBadRequest, "INVALID_REQUEST", err)
		return
	}
	uid := c.GetInt64("user_id")
	resp, err := h.service.CreatePayment(c.Request.Context(), uid, req.BookingID, req.Amount, req.Description, req.Recurring, req.PreviousInvoiceID, req.SubscriptionID)
	if err != nil {
		code := http.StatusInternalServerError
		if err == ErrAmountMismatch {
			code = http.StatusBadRequest
		}
		response.CustomError(c, code, "PAYMENT_CREATE_FAILED", err)
		return
	}
	response.Success(c, http.StatusOK, resp)
}

func (h *Handler) CreateSubscription(c *gin.Context) {
	var req CreateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.CustomError(c, http.StatusBadRequest, "INVALID_REQUEST", err)
		return
	}
	resp, err := h.service.CreateSubscription(c.Request.Context(), c.GetInt64("user_id"), req.Amount)
	if err != nil {
		response.CustomError(c, http.StatusInternalServerError, "SUBSCRIPTION_CREATE_FAILED", err)
		return
	}
	response.Success(c, http.StatusOK, resp)
}

func (h *Handler) MySubscription(c *gin.Context) {
	sub, err := h.service.GetMySubscription(c.Request.Context(), c.GetInt64("user_id"))
	if err != nil {
		response.CustomError(c, http.StatusNotFound, "SUBSCRIPTION_NOT_FOUND", err)
		return
	}
	response.Success(c, http.StatusOK, sub)
}

func (h *Handler) CancelSubscription(c *gin.Context) {
	if err := h.service.CancelSubscription(c.Request.Context(), c.GetInt64("user_id")); err != nil {
		response.CustomError(c, http.StatusBadRequest, "SUBSCRIPTION_CANCEL_FAILED", err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{"status": "canceled"})
}

func (h *Handler) ResultCallback(c *gin.Context) {
	rawBody, _ := io.ReadAll(c.Request.Body)
	c.Request.Body = io.NopCloser(strings.NewReader(string(rawBody)))
	if err := c.Request.ParseForm(); err != nil {
		c.String(http.StatusBadRequest, "bad request")
		return
	}
	outSum := c.PostForm("OutSum")
	invID, err := strconv.ParseInt(c.PostForm("InvId"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "bad request")
		return
	}
	ack, err := h.service.HandleResultCallback(c.Request.Context(), outSum, invID, c.PostForm("SignatureValue"), collectShp(c), string(rawBody))
	if err != nil {
		if err == ErrInvalidSignature || err == ErrAmountMismatch || err == ErrReplayDetected {
			c.String(http.StatusForbidden, "forbidden")
			return
		}
		c.String(http.StatusInternalServerError, "internal error")
		return
	}
	c.String(http.StatusOK, ack)
}

func (h *Handler) SuccessCallback(c *gin.Context) {
	var req PaymentCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.CustomError(c, http.StatusBadRequest, "INVALID_REQUEST", err)
		return
	}
	ok, err := h.service.HandleSuccessCallback(c.Request.Context(), req.OutSum, req.InvID, req.SignatureValue, req.ShpParams, "")
	if err != nil {
		response.CustomError(c, http.StatusForbidden, "PAYMENT_VALIDATION_FAILED", err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{"validated": ok, "redirect_url": h.service.frontSuccess})
}

func (h *Handler) FailCallback(c *gin.Context) {
	var req PaymentFailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.CustomError(c, http.StatusBadRequest, "INVALID_REQUEST", err)
		return
	}
	if err := h.service.FailPayment(c.Request.Context(), req.InvID); err != nil {
		response.CustomError(c, http.StatusInternalServerError, "PAYMENT_FAIL_UPDATE_FAILED", err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{"redirect_url": h.service.frontFail})
}

func collectShp(c *gin.Context) map[string]string {
	res := map[string]string{}
	for k, v := range c.Request.Form {
		if strings.HasPrefix(strings.ToLower(k), "shp_") && len(v) > 0 {
			res[trimShpKey(k)] = v[0]
		}
	}
	for k, v := range c.Request.URL.Query() {
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
