package review

type CreateReviewRequest struct {
	StudioID  int64    `json:"studio_id" validate:"required,gt=0"`
	BookingID *int64   `json:"booking_id,omitempty"`
	Rating    int      `json:"rating" validate:"required,gte=1,lte=5"`
	Comment   string   `json:"comment,omitempty"`
	Photos    []string `json:"photos,omitempty"`
}

type OwnerResponseRequest struct {
	Response string `json:"response" validate:"required"`
}


