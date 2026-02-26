package payment

type InitPaymentRequest struct {
	BookingID   int64             `json:"booking_id" binding:"required"`
	OutSum      string            `json:"out_sum" binding:"required"`
	Description string            `json:"description"`
	ShpParams   map[string]string `json:"shp_params"`
}

type CreatePaymentRequest struct {
	BookingID         int64   `json:"booking_id" binding:"required"`
	Amount            string  `json:"amount"`
	OutSum            string  `json:"out_sum"`
	Description       string  `json:"description"`
	Recurring         bool    `json:"recurring"`
	PreviousInvoiceID *int64  `json:"previous_invoice_id"`
	SubscriptionID    *string `json:"subscription_id"`
}

type CreateSubscriptionRequest struct {
	Amount string `json:"amount" binding:"required"`
}

type PaymentCallbackRequest struct {
	OutSum         string            `json:"OutSum" binding:"required"`
	InvID          int64             `json:"InvId" binding:"required"`
	SignatureValue string            `json:"SignatureValue" binding:"required"`
	ShpParams      map[string]string `json:"shp_params"`
}

type PaymentFailRequest struct {
	InvID int64 `json:"inv_id" binding:"required"`
}

type InitPaymentResponse struct {
	InvID      int64  `json:"inv_id"`
	PaymentURL string `json:"payment_url"`
	Signature  string `json:"signature"`
	Status     string `json:"status"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
type SuccessCallbackResponse struct {
	Status    string `json:"status"`
	Validated bool   `json:"validated"`
}
