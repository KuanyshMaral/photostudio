package catalog

import (
	"context"
	"errors"
	"time"

	"photostudio/internal/domain"
	"photostudio/internal/repository"
)

var (
	ErrForbidden       = errors.New("forbidden")
	ErrInvalidRoomType = errors.New("invalid room type")
)

type Service struct {
	studioRepo             *repository.StudioRepository
	roomRepo               *repository.RoomRepository
	equipmentRepo          *repository.EquipmentRepository
	studioWorkingHoursRepo repository.StudioWorkingHoursRepository
}

func NewService(
	studioRepo *repository.StudioRepository,
	roomRepo *repository.RoomRepository,
	equipmentRepo *repository.EquipmentRepository,
	studioWorkingHoursRepo repository.StudioWorkingHoursRepository,
) *Service {
	return &Service{studioRepo, roomRepo, equipmentRepo, studioWorkingHoursRepo}
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

func (s *Service) UpdateRoom(ctx context.Context, roomID int64, req UpdateRoomRequest) (*domain.Room, error) {
	room, err := s.roomRepo.GetByID(ctx, roomID)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		room.Name = *req.Name
	}
	if req.Description != nil {
		room.Description = *req.Description
	}
	if req.AreaSqm != nil && *req.AreaSqm > 0 {
		room.AreaSqm = *req.AreaSqm
	}
	if req.Capacity != nil && *req.Capacity > 0 {
		room.Capacity = *req.Capacity
	}
	if req.RoomType != nil {
		rt, err := domain.ParseRoomType(*req.RoomType)
		if err != nil {
			return nil, ErrInvalidRoomType
		}
		room.RoomType = rt
	}
	if req.PricePerHourMin != nil && *req.PricePerHourMin >= 0 {
		room.PricePerHourMin = *req.PricePerHourMin
	}
	if req.PricePerHourMax != nil {
		room.PricePerHourMax = req.PricePerHourMax
	}
	if req.Amenities != nil {
		room.Amenities = *req.Amenities
	}
	if req.Photos != nil {
		room.Photos = *req.Photos
	}

	if err := s.roomRepo.Update(ctx, room); err != nil {
		return nil, err
	}
	return room, nil
}

func (s *Service) DeleteRoom(ctx context.Context, roomID int64) error {
	return s.roomRepo.SetActive(ctx, roomID, false)
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

func (s *Service) AddStudioPhotos(ctx context.Context, userID, studioID int64, urls []string) error {
	studio, err := s.studioRepo.GetByID(ctx, studioID)
	if err != nil {
		return err
	}
	if studio.OwnerID != userID {
		return ErrForbidden
	}

	// LIMIT
	existing := len(studio.Photos)
	space := 10 - existing
	if space <= 0 {
		return errors.New("photo limit reached (max 10)")
	}
	if len(urls) > space {
		urls = urls[:space]
	}

	return s.studioRepo.AddPhotos(ctx, studioID, urls)
}

// WorkingStatusResponse представляет статус работы студии
type WorkingStatusResponse struct {
	IsOpen       bool                   `json:"is_open"`
	Message      string                 `json:"message"`
	OpenTime     string                 `json:"open_time,omitempty"`
	CloseTime    string                 `json:"close_time,omitempty"`
	WorkingHours domain.WorkingHoursMap `json:"working_hours,omitempty"`
}

// GetStudioWorkingStatus возвращает статус работы студии (открыта/закрыта)
func (s *Service) GetStudioWorkingStatus(ctx context.Context, studioID int64) (*WorkingStatusResponse, error) {
	studio, err := s.studioRepo.GetByID(ctx, studioID)
	if err != nil {
		return nil, err
	}

	return s.calculateWorkingStatus(studio), nil
}

func (s *Service) calculateWorkingStatus(studio *domain.Studio) *WorkingStatusResponse {
	response := &WorkingStatusResponse{
		IsOpen:       false,
		WorkingHours: studio.WorkingHours,
	}

	// Если нет рабочих часов, считаем закрытым
	if len(studio.WorkingHours) == 0 {
		response.Message = "Часы работы не указаны"
		return response
	}

	// Получаем текущее время в Алматы (UTC+5 для Казахстана)
	now := time.Now()

	// Определяем день недели
	weekday := now.Weekday().String()
	schedule, exists := studio.WorkingHours[weekday]
	if !exists {
		response.Message = "Сегодня выходной"
		return response
	}

	// Парсим время открытия и закрытия
	openTime, err1 := time.Parse("15:04", schedule.Open)
	closeTime, err2 := time.Parse("15:04", schedule.Close)
	if err1 != nil || err2 != nil {
		response.Message = "Ошибка в формате рабочих часов"
		return response
	}

	// Проверяем, открыта ли студия сейчас
	currentTime := now.Format("15:04")
	currentParsed, _ := time.Parse("15:04", currentTime)

	if currentParsed.After(openTime) && currentParsed.Before(closeTime) {
		response.IsOpen = true
		response.Message = "Открыто"
		response.OpenTime = schedule.Open
		response.CloseTime = schedule.Close
	} else {
		response.Message = "Закрыто"
		response.OpenTime = schedule.Open
		response.CloseTime = schedule.Close
	}

	return response
}

/* ---------- WORKING HOURS NEW ---------- */

// GetStudioWorkingHours возвращает полную информацию о часах работы
func (s *Service) GetStudioWorkingHours(ctx context.Context, studioID int64) (*WorkingHoursResponse, error) {
	// Проверяем существование студии, но не используем результат
	_, err := s.studioRepo.GetByID(ctx, studioID)
	if err != nil {
		return nil, err
	}

	// Получаем структурированные часы работы
	hours, err := s.studioWorkingHoursRepo.GetHoursForStudio(studioID)
	if err != nil {
		return nil, err
	}

	// Рассчитываем live статус
	isOpen, statusText, nextOpen := CalculateLiveStatus(hours)

	response := &WorkingHoursResponse{
		StudioID:     studioID,
		Hours:        hours,
		CompactText:  FormatCompactHours(hours),
		IsOpenNow:    isOpen,
		StatusText:   statusText,
		NextOpenTime: nextOpen,
	}

	return response, nil
}

// UpdateStudioWorkingHours обновляет часы работы студии
func (s *Service) UpdateStudioWorkingHours(ctx context.Context, userID, studioID int64, hours []domain.WorkingHours) error {
	studio, err := s.studioRepo.GetByID(ctx, studioID)
	if err != nil {
		return err
	}

	// Проверка прав
	if studio.OwnerID != userID {
		return ErrForbidden
	}

	// Валидация часов
	for _, h := range hours {
		if h.DayOfWeek < 0 || h.DayOfWeek > 6 {
			return errors.New("invalid day of week")
		}
		if h.OpenTime == "" || h.CloseTime == "" {
			return errors.New("open and close times are required")
		}
	}

	// Сохраняем (не нужно преобразовывать в JSON, GORM сериализует автоматически)
	studioHours := &domain.StudioWorkingHours{
		StudioID: studioID,
		Hours:    hours, // напрямую массив WorkingHours
	}

	return s.studioWorkingHoursRepo.CreateOrUpdate(studioHours)
}

// GetWorkingHoursForDate возвращает часы работы на конкретную дату
func (s *Service) GetWorkingHoursForDate(ctx context.Context, studioID int64, date time.Time) (*domain.WorkingHours, error) {
	hours, err := s.studioWorkingHoursRepo.GetHoursForStudio(studioID)
	if err != nil {
		return nil, err
	}

	dayOfWeek := int(date.Weekday())
	for _, h := range hours {
		if h.DayOfWeek == dayOfWeek {
			return &h, nil
		}
	}

	// Дефолтные часы
	return &domain.WorkingHours{
		DayOfWeek: dayOfWeek,
		OpenTime:  "09:00",
		CloseTime: "21:00",
		IsClosed:  false,
	}, nil
}
