package catalog

import (
	"context"
	"errors"

	"photostudio/internal/domain"
	"photostudio/internal/repository"
)

var ErrForbidden = errors.New("forbidden")

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

func (s *Service) CreateStudio(ctx context.Context, user *domain.User, req CreateStudioRequest) error {
	if user.Role != domain.RoleStudioOwner || user.StudioStatus != domain.StatusVerified {
		return ErrForbidden
	}

	studio := &domain.Studio{
		OwnerID:      user.ID,
		Name:         req.Name,
		Address:      req.Address,
		City:         req.City,
		WorkingHours: req.WorkingHours,
	}

	return s.studioRepo.Create(ctx, studio)
}

func (s *Service) UpdateStudio(ctx context.Context, userID, studioID int64, req UpdateStudioRequest) error {
	studio, err := s.studioRepo.GetByID(ctx, studioID)
	if err != nil {
		return err
	}

	if studio.OwnerID != userID {
		return ErrForbidden
	}

	studio.Name = req.Name
	studio.Address = req.Address
	studio.WorkingHours = req.WorkingHours

	return s.studioRepo.Update(ctx, studio)
}

/* ---------- ROOMS ---------- */

func (s *Service) CreateRoom(ctx context.Context, userID, studioID int64, req CreateRoomRequest) error {
	studio, err := s.studioRepo.GetByID(ctx, studioID)
	if err != nil {
		return err
	}

	if studio.OwnerID != userID {
		return ErrForbidden
	}

	room := &domain.Room{
		StudioID:        studioID,
		Name:            req.Name,
		AreaSqm:         req.AreaSqm,
		Capacity:        req.Capacity,
		RoomType:        domain.RoomType(req.RoomType),
		PricePerHourMin: req.PricePerHourMin,
		PricePerHourMax: req.PricePerHourMax,
		Amenities:       req.Amenities,
		IsActive:        true,
	}

	return s.roomRepo.Create(ctx, room)
}

/* ---------- EQUIPMENT ---------- */

func (s *Service) AddEquipment(ctx context.Context, userID, roomID int64, req CreateEquipmentRequest) error {
	// 1. Find the room
	room, err := s.roomRepo.GetByID(ctx, roomID)
	if err != nil {
		return err
	}

	// 2. Find the studio to check ownership
	studio, err := s.studioRepo.GetByID(ctx, room.StudioID)
	if err != nil {
		return err
	}

	// 3. Security Check: Only the owner can add equipment to their rooms
	if studio.OwnerID != userID {
		return ErrForbidden
	}

	eq := &domain.Equipment{
		RoomID:      roomID,
		Name:        req.Name,
		Category:    req.Category,
		Brand:       req.Brand,
		Model:       req.Model,
		Quantity:    req.Quantity,
		RentalPrice: req.RentalPrice,
	}

	return s.equipmentRepo.Create(ctx, eq)
}
