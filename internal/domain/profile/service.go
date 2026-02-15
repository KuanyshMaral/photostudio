package profile

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Service handles profile business logic
type Service struct {
	clientRepo *ClientRepository
	ownerRepo  *OwnerRepository
	adminRepo  *AdminRepository
}

// NewService creates profile service
func NewService(clientRepo *ClientRepository, ownerRepo *OwnerRepository, adminRepo *AdminRepository) *Service {
	return &Service{
		clientRepo: clientRepo,
		ownerRepo:  ownerRepo,
		adminRepo:  adminRepo,
	}
}

// EnsureClientProfile creates an empty client profile if it doesn't exist
func (s *Service) EnsureClientProfile(ctx context.Context, userID int64) (*ClientProfile, error) {
	existing, err := s.clientRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	now := time.Now()
	profile := &ClientProfile{
		UserID:    userID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.clientRepo.Create(ctx, profile); err != nil {
		return nil, err
	}

	return profile, nil
}

// EnsureOwnerProfile creates an owner profile if it doesn't exist
func (s *Service) EnsureOwnerProfile(ctx context.Context, userID int64, req *CreateOwnerProfileRequest) (*OwnerProfile, error) {
	existing, err := s.ownerRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	now := time.Now()
	profile := &OwnerProfile{
		UserID:             userID,
		CompanyName:        req.CompanyName,
		Bin:                nullString(req.Bin),
		LegalAddress:       nullString(req.LegalAddress),
		ContactPerson:      nullString(req.ContactPerson),
		ContactPosition:    nullString(req.ContactPosition),
		Phone:              nullString(req.Phone),
		Email:              nullString(req.Email),
		Website:            nullString(req.Website),
		VerificationStatus: "pending",
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if err := s.ownerRepo.Create(ctx, profile); err != nil {
		return nil, err
	}

	return profile, nil
}

// EnsureAdminProfile creates an admin profile if it doesn't exist
func (s *Service) EnsureAdminProfile(ctx context.Context, userID uuid.UUID, req *CreateAdminProfileRequest) (*AdminProfile, error) {
	existing, err := s.adminRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	now := time.Now()
	profile := &AdminProfile{
		UserID:    userID,
		FullName:  req.FullName,
		Position:  nullString(req.Position),
		Phone:     nullString(req.Phone),
		IsActive:  true,
		CreatedBy: nullUUID(req.CreatedBy),
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.adminRepo.Create(ctx, profile); err != nil {
		return nil, err
	}

	return profile, nil
}

// GetClientProfile retrieves client profile
func (s *Service) GetClientProfile(ctx context.Context, userID int64) (*ClientProfile, error) {
	profile, err := s.clientRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, ErrProfileNotFound
	}
	return profile, nil
}

// GetOwnerProfile retrieves owner profile
func (s *Service) GetOwnerProfile(ctx context.Context, userID int64) (*OwnerProfile, error) {
	profile, err := s.ownerRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, ErrProfileNotFound
	}
	return profile, nil
}

// GetAdminProfile retrieves admin profile
func (s *Service) GetAdminProfile(ctx context.Context, userID uuid.UUID) (*AdminProfile, error) {
	profile, err := s.adminRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, ErrProfileNotFound
	}
	return profile, nil
}

// UpdateClientProfile updates client profile
func (s *Service) UpdateClientProfile(ctx context.Context, userID int64, req *UpdateClientProfileRequest) (*ClientProfile, error) {
	profile, err := s.clientRepo.GetByUserID(ctx, userID)
	if err != nil || profile == nil {
		return nil, ErrProfileNotFound
	}

	// Update fields if provided
	if req.Name != nil {
		profile.Name = nullString(*req.Name)
	}
	if req.Nickname != nil {
		profile.Nickname = nullString(*req.Nickname)
	}
	if req.Phone != nil {
		profile.Phone = nullString(*req.Phone)
	}
	if req.AvatarURL != nil {
		profile.AvatarURL = nullString(*req.AvatarURL)
	}

	profile.UpdatedAt = time.Now()

	if err := s.clientRepo.Update(ctx, profile); err != nil {
		return nil, err
	}

	return profile, nil
}

// UpdateOwnerProfile updates owner profile
func (s *Service) UpdateOwnerProfile(ctx context.Context, userID int64, req *UpdateOwnerProfileRequest) (*OwnerProfile, error) {
	profile, err := s.ownerRepo.GetByUserID(ctx, userID)
	if err != nil || profile == nil {
		return nil, ErrProfileNotFound
	}

	// Update fields if provided
	if req.CompanyName != nil {
		profile.CompanyName = *req.CompanyName
	}
	if req.Bin != nil {
		profile.Bin = nullString(*req.Bin)
	}
	if req.LegalAddress != nil {
		profile.LegalAddress = nullString(*req.LegalAddress)
	}
	if req.ContactPerson != nil {
		profile.ContactPerson = nullString(*req.ContactPerson)
	}
	if req.ContactPosition != nil {
		profile.ContactPosition = nullString(*req.ContactPosition)
	}
	if req.Phone != nil {
		profile.Phone = nullString(*req.Phone)
	}
	if req.Email != nil {
		profile.Email = nullString(*req.Email)
	}
	if req.Website != nil {
		profile.Website = nullString(*req.Website)
	}

	profile.UpdatedAt = time.Now()

	if err := s.ownerRepo.Update(ctx, profile); err != nil {
		return nil, err
	}

	return profile, nil
}

// UpdateAdminProfile updates admin profile
func (s *Service) UpdateAdminProfile(ctx context.Context, userID uuid.UUID, req *UpdateAdminProfileRequest) (*AdminProfile, error) {
	profile, err := s.adminRepo.GetByUserID(ctx, userID)
	if err != nil || profile == nil {
		return nil, ErrProfileNotFound
	}

	// Update fields if provided
	if req.FullName != nil {
		profile.FullName = *req.FullName
	}
	if req.Position != nil {
		profile.Position = nullString(*req.Position)
	}
	if req.Phone != nil {
		profile.Phone = nullString(*req.Phone)
	}

	profile.UpdatedAt = time.Now()

	if err := s.adminRepo.Update(ctx, profile); err != nil {
		return nil, err
	}

	return profile, nil
}
