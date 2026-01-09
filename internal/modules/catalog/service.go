package catalog

import (
	"context"
	"errors"

	"photostudio/internal/domain"
	"photostudio/internal/repository"
)

var (
	ErrForbidden       = errors.New("forbidden")
	ErrInvalidRoomType = errors.New("invalid room type")
)

type Service struct {
	studioRepo    *repository.StudioRepository
	roomRepo      *repository.RoomRepository
	equipmentRepo *repository.EquipmentRepository
}

func NewService(
	studioRepo *repository.StudioRepository,
	roomRepo *repository.RoomRepository,
	equipmentRepo *repository.EquipmentRepository,
) *Service {
	return &Service{studioRepo, roomRepo, equipmentRepo}
}

/* ---------- STUDIO ---------- */

func (s *Service) CreateStudio(ctx context.Context, user *domain.User, req CreateStudioRequest) (*domain.Studio, error) {
	// Check if user has permission
	if user.Role != domain.RoleStudioOwner || user.StudioStatus != domain.StatusVerified {
		return nil, ErrForbidden
	}

	studio := &domain.Studio{
		OwnerID:      user.ID,
		Name:         req.Name,
		Description:  req.Description,
		Address:      req.Address,
		District:     req.District,
		City:         req.City,
		Phone:        req.Phone,
		Email:        req.Email,
		Website:      req.Website,
		WorkingHours: req.WorkingHours,
	}

	if err := s.studioRepo.Create(ctx, studio); err != nil {
		return nil, err
	}

	return studio, nil
}

func (s *Service) UpdateStudio(ctx context.Context, userID, studioID int64, req UpdateStudioRequest) (*domain.Studio, error) {
	studio, err := s.studioRepo.GetByID(ctx, studioID)
	if err != nil {
		return nil, err
	}

	// Check ownership
	if studio.OwnerID != userID {
		return nil, ErrForbidden
	}

	// Update fields
	studio.Name = req.Name
	studio.Description = req.Description
	studio.Address = req.Address
	studio.City = req.City
	studio.District = req.District
	studio.Phone = req.Phone
	studio.Email = req.Email
	studio.Website = req.Website
	studio.WorkingHours = req.WorkingHours

	if err := s.studioRepo.Update(ctx, studio); err != nil {
		return nil, err
	}

	return studio, nil
}

func (s *Service) GetStudiosByOwner(ctx context.Context, ownerID int64) ([]domain.Studio, error) {
	return s.studioRepo.GetByOwnerID(ctx, ownerID)
}

/* ---------- ROOMS ---------- */

func (s *Service) CreateRoom(ctx context.Context, userID, studioID int64, req CreateRoomRequest) (*domain.Room, error) {
	// Verify studio exists and user owns it
	studio, err := s.studioRepo.GetByID(ctx, studioID)
	if err != nil {
		return nil, err
	}

	if studio.OwnerID != userID {
		return nil, ErrForbidden
	}

	roomType, err := domain.ParseRoomType(req.RoomType)
	if err != nil {
		return nil, ErrInvalidRoomType
	}

	room := &domain.Room{
		StudioID:        studioID,
		Name:            req.Name,
		Description:     req.Description,
		AreaSqm:         req.AreaSqm,
		Capacity:        req.Capacity,
		RoomType:        roomType,
		PricePerHourMin: req.PricePerHourMin,
		PricePerHourMax: req.PricePerHourMax,
		Amenities:       req.Amenities,
		Photos:          req.Photos,
		IsActive:        true,
	}

	if err := s.roomRepo.Create(ctx, room); err != nil {
		return nil, err
	}

	return room, nil
}

/* ---------- EQUIPMENT ---------- */

func (s *Service) AddEquipment(ctx context.Context, userID, roomID int64, req CreateEquipmentRequest) (*domain.Equipment, error) {
	// 1. Find the room
	room, err := s.roomRepo.GetByID(ctx, roomID)
	if err != nil {
		return nil, err
	}

	// 2. Find the studio to check ownership
	studio, err := s.studioRepo.GetByID(ctx, room.StudioID)
	if err != nil {
		return nil, err
	}

	// 3. Security Check: Only the owner can add equipment to their rooms
	if studio.OwnerID != userID {
		return nil, ErrForbidden
	}

	equipment := &domain.Equipment{
		RoomID:      roomID,
		Name:        req.Name,
		Category:    req.Category,
		Brand:       req.Brand,
		Model:       req.Model,
		Quantity:    req.Quantity,
		RentalPrice: req.RentalPrice,
	}

	if err := s.equipmentRepo.Create(ctx, equipment); err != nil {
		return nil, err
	}

	return equipment, nil
}
