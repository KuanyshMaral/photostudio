package review

type CreateReviewRequest struct {
	TargetType  string         `json:"target_type" validate:"required"`
	TargetID    int64          `json:"target_id" validate:"required,gt=0"`
	ContextType *string        `json:"context_type,omitempty"`
	ContextID   *int64         `json:"context_id,omitempty"`
	Rating      int            `json:"rating" validate:"required,gte=1,lte=5"`
	Comment     string         `json:"comment,omitempty"`
	Photos      []string       `json:"photos,omitempty"`
	Criteria    map[string]int `json:"criteria,omitempty"`
}

type OwnerResponseRequest struct {
	Response string `json:"response" validate:"required"`
}
