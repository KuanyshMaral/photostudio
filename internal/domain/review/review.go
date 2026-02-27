package review

import "time"

type TargetType string

const (
	TargetTypeStudio TargetType = "studio"
)

type Review struct {
	ID            int64      `json:"id"`
	AuthorID      int64      `json:"author_id"`
	TargetType    TargetType `json:"target_type"`
	TargetID      int64      `json:"target_id"`
	ContextType   *string    `json:"context_type,omitempty"`
	ContextID     *int64     `json:"context_id,omitempty"`
	Rating        int        `json:"rating"`
	Comment       string     `json:"comment,omitempty"`
	Photos        []string   `json:"photos,omitempty" gorm:"-"`      // DEPRECATED: use attachments WHERE target_type='review_photos' AND target_id=id
	Attachments   []string   `json:"attachments,omitempty" gorm:"-"` // Populated from attachment service (URLs)
	Criteria      []byte     `json:"criteria,omitempty"`
	OwnerResponse *string    `json:"owner_response,omitempty"`
	RespondedAt   *time.Time `json:"responded_at,omitempty"`
	IsVerified    bool       `json:"is_verified"`
	IsHidden      bool       `json:"is_hidden"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}
