package payment

import (
	"io"
	"net/http"
	"photostudio/internal/pkg/response"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
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

// CreatePayment godoc
// @Summary      Create Robokassa payment
// @Description  Creates Robokassa payment link and signature for a booking
// @Tags         Payments
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body InitPaymentRequest true "Payment init payload"
// @Success      200 {object} InitPaymentResponse
// @Failure      400 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Router       /payments/robokassa/create [post]
func (h *Handler) CreatePayment(c *gin.Context) {
	var req InitPaymentRequest
	body, _ := io.ReadAll(c.Request.Body)
	c.Request.Body = io.NopCloser(strings.NewReader(string(body)))
	h.loggerf("level=info msg=robokassa init request request_body=%s", string(body))
	if err := c.ShouldBindJSON(&req); err != nil {
		h.loggerf("level=error msg=invalid robokassa init payload err=%v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.service.InitPayment(c.Request.Context(), req)
	if err != nil {
		h.loggerf("level=error msg=robokassa init failed request=%+v err=%v", req, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.loggerf("level=info msg=robokassa init response response=%+v", resp)
	c.JSON(http.StatusOK, resp)
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
