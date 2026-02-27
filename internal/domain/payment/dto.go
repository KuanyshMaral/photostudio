package payment

type InitPaymentRequest struct {
	BookingID   int64             `json:"booking_id" binding:"required"`
	OutSum      string            `json:"out_sum" binding:"required"`
	Description string            `json:"description"`
	ShpParams   map[string]string `json:"shp_params"`
}

type CreatePaymentRequest struct {
	BookingID         int64             `json:"booking_id" binding:"required" example:"2"`
	Amount            string            `json:"amount" example:"25000"`
	OutSum            string            `json:"out_sum" example:"25000"`
	Description       string            `json:"description" example:"Booking payment"`
	ShpParams         map[string]string `json:"shp_params"`
	Recurring         bool              `json:"recurring"`
	PreviousInvoiceID *int64            `json:"previous_invoice_id"`
	SubscriptionID    *string           `json:"subscription_id" format:"uuid" example:"3fa85f64-5717-4562-b3fc-2c963f66afa6"`
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
