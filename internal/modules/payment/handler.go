package payment

import (
	"io"
	"net/http"
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

func (h *Handler) RegisterProtectedRoutes(rg *gin.RouterGroup) {
	rg.POST("/payments/robokassa/init", h.InitPayment)
}

func (h *Handler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	rg.POST("/payments/robokassa/result", h.ResultCallback)
	rg.GET("/payments/robokassa/success", h.SuccessCallback)
}

// InitPayment godoc
// @Summary      Initialize Robokassa payment
// @Description  Creates Robokassa payment link and signature for a booking
// @Tags         Payments
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body InitPaymentRequest true "Payment init payload"
// @Success      200 {object} InitPaymentResponse
// @Failure      400 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Router       /payments/robokassa/init [post]
func (h *Handler) InitPayment(c *gin.Context) {
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

// ResultCallback godoc
// @Summary      Robokassa ResultURL callback
// @Description  Validates callback signature and marks payment as paid (idempotent)
// @Tags         Payments
// @Produce      plain
// @Param        OutSum formData string true "Amount"
// @Param        InvId formData integer true "Invoice ID"
// @Param        SignatureValue formData string true "MD5 signature"
// @Success      200 {string} string "OK{InvId}"
// @Failure      400 {string} string "bad request"
// @Failure      403 {string} string "forbidden"
// @Failure      500 {string} string "internal error"
// @Router       /payments/robokassa/result [post]
func (h *Handler) ResultCallback(c *gin.Context) {
	rawBody, _ := io.ReadAll(c.Request.Body)
	c.Request.Body = io.NopCloser(strings.NewReader(string(rawBody)))
	_ = c.Request.ParseForm()
	h.loggerf("level=info msg=robokassa result callback raw_body=%s form=%v", string(rawBody), c.Request.PostForm)

	outSum := c.PostForm("OutSum")
	invID, err := strconv.ParseInt(c.PostForm("InvId"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "bad request")
		return
	}
	signature := c.PostForm("SignatureValue")
	shp := collectShp(c)

	ack, err := h.service.HandleResultCallback(c.Request.Context(), outSum, invID, signature, shp, string(rawBody))
	if err != nil {
		h.loggerf("level=error msg=robokassa result callback failed inv_id=%d err=%v", invID, err)
		if err == ErrInvalidSignature || err == ErrAmountMismatch {
			c.String(http.StatusForbidden, "forbidden")
			return
		}
		c.String(http.StatusInternalServerError, "internal error")
		return
	}
	h.loggerf("level=info msg=robokassa result callback handled inv_id=%d ack=%s", invID, ack)
	c.String(http.StatusOK, ack)
}

// SuccessCallback godoc
// @Summary      Robokassa SuccessURL callback
// @Description  Validates customer return callback signature
// @Tags         Payments
// @Produce      json
// @Param        OutSum query string true "Amount"
// @Param        InvId query integer true "Invoice ID"
// @Param        SignatureValue query string true "MD5 signature"
// @Success      200 {object} SuccessCallbackResponse
// @Failure      400 {object} ErrorResponse
// @Failure      403 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Router       /payments/robokassa/success [get]
func (h *Handler) SuccessCallback(c *gin.Context) {
	raw := c.Request.URL.RawQuery
	h.loggerf("level=info msg=robokassa success callback raw_query=%s query=%v", raw, c.Request.URL.Query())
	outSum := c.Query("OutSum")
	invID, err := strconv.ParseInt(c.Query("InvId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid InvId"})
		return
	}
	signature := c.Query("SignatureValue")
	shp := collectShp(c)

	ok, err := h.service.HandleSuccessCallback(c.Request.Context(), outSum, invID, signature, shp, raw)
	if err != nil {
		h.loggerf("level=error msg=robokassa success callback failed inv_id=%d err=%v", invID, err)
		if err == ErrInvalidSignature || err == ErrAmountMismatch {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "validated": ok})
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
