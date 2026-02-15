package lead

import (
	"context"
	"database/sql"
	"time"

	"photostudio/internal/domain/auth"
)

// AuthRepository defines authentication data access
type AuthRepository interface {
	GetByEmail(ctx context.Context, email string) (*auth.User, error)
	CreateWithPassword(ctx context.Context, user *auth.User, password string) error
}

// Service handles lead business logic
type Service struct {
	repo     *Repository
	authRepo AuthRepository
}

// NewService creates lead service
func NewService(repo *Repository, authRepo AuthRepository) *Service {
	return &Service{
		repo:     repo,
		authRepo: authRepo,
	}
}

// SubmitLead creates a new owner lead (public endpoint)
func (s *Service) SubmitLead(ctx context.Context, req *SubmitLeadRequest, ip string, userAgent string) (*OwnerLead, error) {
	// Check if email already exists as user
	existingUser, _ := s.authRepo.GetByEmail(ctx, req.ContactEmail)
	if existingUser != nil {
		return nil, ErrEmailExists
	}

	// Check if lead with this email already exists
	existingLead, _ := s.repo.GetByEmail(ctx, req.ContactEmail)
	if existingLead != nil && !existingLead.IsConverted() {
		return existingLead, nil // Return existing lead
	}

	now := time.Now()
	lead := &OwnerLead{
		ContactName:     req.ContactName,
		ContactEmail:    req.ContactEmail,
		ContactPhone:    req.ContactPhone,
		ContactPosition: sql.NullString{String: req.ContactPosition, Valid: req.ContactPosition != ""},
		CompanyName:     req.CompanyName,
		Bin:             sql.NullString{String: req.Bin, Valid: req.Bin != ""},
		LegalAddress:    sql.NullString{String: req.LegalAddress, Valid: req.LegalAddress != ""},
		Website:         sql.NullString{String: req.Website, Valid: req.Website != ""},
		UseCase:         sql.NullString{String: req.UseCase, Valid: req.UseCase != ""},
		HowFoundUs:      sql.NullString{String: req.HowFoundUs, Valid: req.HowFoundUs != ""},
		Status:          StatusNew,
		Source:          sql.NullString{String: "website", Valid: true},
		UTMSource:       sql.NullString{String: req.UTMSource, Valid: req.UTMSource != ""},
		UTMMedium:       sql.NullString{String: req.UTMMedium, Valid: req.UTMMedium != ""},
		UTMCampaign:     sql.NullString{String: req.UTMCampaign, Valid: req.UTMCampaign != ""},
		IPAddress:       sql.NullString{String: ip, Valid: ip != ""},
		UserAgent:       sql.NullString{String: userAgent, Valid: userAgent != ""},
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.repo.Create(ctx, lead); err != nil {
		return nil, err
	}

	return lead, nil
}

// GetByID returns lead by ID
func (s *Service) GetByID(ctx context.Context, id int64) (*OwnerLead, error) {
	lead, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if lead == nil {
		return nil, ErrLeadNotFound
	}
	return lead, nil
}

// ListLeads returns leads with optional status filter
func (s *Service) ListLeads(ctx context.Context, status *Status, limit, offset int) ([]*OwnerLead, int, error) {
	return s.repo.List(ctx, status, limit, offset)
}

// UpdateStatus updates lead status
func (s *Service) UpdateStatus(ctx context.Context, id int64, status Status, notes, reason string) error {
	lead, err := s.repo.GetByID(ctx, id)
	if err != nil || lead == nil {
		return ErrLeadNotFound
	}

	if lead.IsConverted() {
		return ErrAlreadyConverted
	}

	return s.repo.UpdateStatus(ctx, id, status, notes, reason)
}

// MarkContacted marks lead as contacted
func (s *Service) MarkContacted(ctx context.Context, id int64) error {
	lead, err := s.repo.GetByID(ctx, id)
	if err != nil || lead == nil {
		return ErrLeadNotFound
	}

	// Update status to contacted if it was new
	if lead.Status == StatusNew {
		if err := s.repo.UpdateStatus(ctx, id, StatusContacted, "", ""); err != nil {
			return err
		}
	}

	// Always update last contacted at
	return s.repo.MarkContacted(ctx, id)
}

// RejectLead marks lead as rejected
func (s *Service) RejectLead(ctx context.Context, id int64, reason string) error {
	lead, err := s.repo.GetByID(ctx, id)
	if err != nil || lead == nil {
		return ErrLeadNotFound
	}

	if lead.IsConverted() {
		return ErrAlreadyConverted
	}

	return s.repo.UpdateStatus(ctx, id, StatusRejected, "", reason)
}

// Assign assigns lead to admin
func (s *Service) Assign(ctx context.Context, id int64, adminID string, priority int) error {
	lead, err := s.repo.GetByID(ctx, id)
	if err != nil || lead == nil {
		return ErrLeadNotFound
	}

	return s.repo.Assign(ctx, id, adminID, priority)
}

// ConvertLead converts lead to owner account
// Note: Profile creation is handled by profile service after user creation
func (s *Service) ConvertLead(ctx context.Context, leadID int64, req *ConvertLeadRequest) (*auth.User, error) {
	lead, err := s.repo.GetByID(ctx, leadID)
	if err != nil || lead == nil {
		return nil, ErrLeadNotFound
	}

	if lead.IsConverted() {
		return nil, ErrAlreadyConverted
	}

	// Allow conversion for New, Contacted, and Qualified leads.
	// Only block Rejected or Lost.
	if lead.Status == StatusRejected || lead.Status == StatusLost {
		return nil, ErrCannotConvert
	}

	// Check if email already exists
	existingUser, _ := s.authRepo.GetByEmail(ctx, lead.ContactEmail)
	if existingUser != nil {
		return nil, ErrEmailExists
	}

	// Create user with role studio_owner and email_verified=true
	user := &auth.User{
		Email:           lead.ContactEmail,
		Role:            auth.RoleStudioOwner,
		EmailVerified:   true,
		EmailVerifiedAt: timePtr(time.Now()),
		Name:            req.LegalName,       // Use LegalName as initial user name
		StudioStatus:    auth.StatusVerified, // Auto-verify since admin converted
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// The auth service will handle password hashing
	if err := s.authRepo.CreateWithPassword(ctx, user, req.Password); err != nil {
		return nil, err
	}

	// Mark lead as converted
	_ = s.repo.MarkConverted(ctx, leadID, user.ID)

	// NOTE: Owner profile creation should be handled by the caller (profile service)
	// after this method returns successfully

	return user, nil
}

// GetStats returns lead statistics
func (s *Service) GetStats(ctx context.Context) (map[Status]int, error) {
	return s.repo.CountByStatus(ctx)
}

// Helper
func timePtr(t time.Time) *time.Time {
	return &t
}
