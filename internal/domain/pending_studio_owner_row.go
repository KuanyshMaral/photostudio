package domain

import "time"

// PendingStudioOwnerRow Row DTO for queries that join studio_owners + users (no cycles with admin package)
type PendingStudioOwnerRow struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	BIN         string    `json:"bin"`
	CompanyName string    `json:"company_name"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}
