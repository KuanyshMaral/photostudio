package payment

type InitPaymentRequest struct {
	BookingID   int64             `json:"booking_id" binding:"required" example:"123"`
	OutSum      string            `json:"out_sum" binding:"required" example:"2500.00"`
	Description string            `json:"description" example:"Room booking #123"`
	ShpParams   map[string]string `json:"shp_params" example:"{\"booking_id\":\"123\"}"`
}

type InitPaymentResponse struct {
	InvID      int64  `json:"inv_id" example:"1700000000000000000"`
	PaymentURL string `json:"payment_url" example:"https://auth.robokassa.ru/Merchant/Index.aspx?..."`
	Signature  string `json:"signature" example:"ABCDEF1234567890ABCDEF1234567890"`
	Status     string `json:"status" example:"created"`
}

type ErrorResponse struct {
	Error string `json:"error" example:"invalid request"`
}

type SuccessCallbackResponse struct {
	Status    string `json:"status" example:"ok"`
	Validated bool   `json:"validated" example:"true"`
}


