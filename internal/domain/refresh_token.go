package domain

import "time"

// RefreshToken stores refresh tokens for users.
//
// Security notes:
// - We never store the raw token in DB, only its SHA-256 hash (TokenHash).
// - On refresh we rotate tokens: old token is revoked and replaced by a new one.
type RefreshToken struct {
	ID int64 `json:"id" gorm:"primaryKey"`

	UserID int64 `json:"user_id" gorm:"index;not null"`
	User   User  `json:"-" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`

	TokenHash string `json:"-" gorm:"size:64;uniqueIndex;not null"`

	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt time.Time  `json:"expires_at" gorm:"index;not null"`
	RevokedAt *time.Time `json:"revoked_at" gorm:"index"`

	ReplacedByID *int64 `json:"replaced_by_id" gorm:"index"`
}

func (t *RefreshToken) IsExpired(now time.Time) bool {
	return now.After(t.ExpiresAt)
}

func (t *RefreshToken) IsRevoked() bool {
	return t.RevokedAt != nil
}
