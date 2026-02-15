package lead

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
)

// Repository handles lead data access
type Repository struct {
	db *sqlx.DB
}

// NewRepository creates lead repository
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts a new lead
func (r *Repository) Create(ctx context.Context, lead *OwnerLead) error {
	query := `
		INSERT INTO owner_leads (
			contact_name, contact_email, contact_phone, contact_position,
			company_name, bin, legal_address, website,
			use_case, how_found_us,
			status, priority, notes,
			source, utm_source, utm_medium, utm_campaign, referrer_url,
			ip_address, user_agent,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22
		) RETURNING id, created_at, updated_at
	`

	return r.db.QueryRowContext(
		ctx, query,
		lead.ContactName, lead.ContactEmail, lead.ContactPhone, lead.ContactPosition,
		lead.CompanyName, lead.Bin, lead.LegalAddress, lead.Website,
		lead.UseCase, lead.HowFoundUs,
		lead.Status, lead.Priority, lead.Notes,
		lead.Source, lead.UTMSource, lead.UTMMedium, lead.UTMCampaign, lead.ReferrerURL,
		lead.IPAddress, lead.UserAgent,
		lead.CreatedAt, lead.UpdatedAt,
	).Scan(&lead.ID, &lead.CreatedAt, &lead.UpdatedAt)
}

// GetByID retrieves lead by ID
func (r *Repository) GetByID(ctx context.Context, id int64) (*OwnerLead, error) {
	var lead OwnerLead
	query := `SELECT * FROM owner_leads WHERE id = $1`
	err := r.db.GetContext(ctx, &lead, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &lead, err
}

// GetByEmail retrieves lead by contact email
func (r *Repository) GetByEmail(ctx context.Context, email string) (*OwnerLead, error) {
	var lead OwnerLead
	query := `SELECT * FROM owner_leads WHERE contact_email = $1 ORDER BY created_at DESC LIMIT 1`
	err := r.db.GetContext(ctx, &lead, query, email)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &lead, err
}

// List returns leads with optional status filter
func (r *Repository) List(ctx context.Context, status *Status, limit, offset int) ([]*OwnerLead, int, error) {
	var leads []*OwnerLead
	var total int

	var query string
	var args []interface{}

	if status != nil {
		query = `SELECT * FROM owner_leads WHERE status = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
		args = []interface{}{*status, limit, offset}
		r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM owner_leads WHERE status = $1`, *status)
	} else {
		query = `SELECT * FROM owner_leads ORDER BY created_at DESC LIMIT $1 OFFSET $2`
		args = []interface{}{limit, offset}
		r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM owner_leads`)
	}

	err := r.db.SelectContext(ctx, &leads, query, args...)
	return leads, total, err
}

// UpdateStatus updates lead status
func (r *Repository) UpdateStatus(ctx context.Context, id int64, status Status, notes, reason string) error {
	query := `
		UPDATE owner_leads
		SET status = $2, notes = $3, rejection_reason = $4, updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, id, status, nullString(notes), nullString(reason))
	return err
}

// MarkContacted marks lead as contacted
func (r *Repository) MarkContacted(ctx context.Context, id int64) error {
	query := `
		UPDATE owner_leads
		SET last_contacted_at = NOW(), 
		    follow_up_count = follow_up_count + 1, 
		    updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// Assign assigns lead to admin
func (r *Repository) Assign(ctx context.Context, id int64, adminID string, priority int) error {
	query := `
		UPDATE owner_leads
		SET assigned_to = $2, priority = $3, updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, id, adminID, priority)
	return err
}

// MarkConverted marks lead as converted
func (r *Repository) MarkConverted(ctx context.Context, leadID, userID int64) error {
	query := `
		UPDATE owner_leads
		SET status = $2, converted_at = NOW(), converted_user_id = $3, updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, leadID, StatusConverted, userID)
	return err
}

// CountByStatus returns lead counts by status
func (r *Repository) CountByStatus(ctx context.Context) (map[Status]int, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT status, COUNT(*) FROM owner_leads GROUP BY status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[Status]int)
	for rows.Next() {
		var status Status
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		counts[status] = count
	}
	return counts, nil
}

// Helper functions
func nullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

func nullInt64(i int64) sql.NullInt64 {
	return sql.NullInt64{Int64: i, Valid: i != 0}
}
