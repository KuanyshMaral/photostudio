package booking

import (
	"context"
	"encoding/json"
	"math"
	"photostudio/internal/repository"
	"sort"
	"time"

	"photostudio/internal/domain"
	_ "photostudio/internal/repository"

	"github.com/jackc/pgx/v5/pgconn"
)

type TimeSlot struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type BookingDetails struct {
	ID         int64     `json:"id"`
	Status     string    `json:"status"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	TotalPrice float64   `json:"total_price"`

	RoomID   int64  `json:"room_id"`
	RoomName string `json:"room_name"`

	StudioID   int64  `json:"studio_id"`
	StudioName string `json:"studio_name"`
}

type Service struct {
	bookings               BookingRepository
	rooms                  RoomRepository
	notifs                 NotificationSender
	studioWorkingHoursRepo repository.StudioWorkingHoursRepository // Добавляем поле
}

func NewService(
	bookings BookingRepository,
	rooms RoomRepository,
	notifs NotificationSender,
	studioWorkingHoursRepo repository.StudioWorkingHoursRepository, // Добавляем параметр
) *Service {
	return &Service{
		bookings:               bookings,
		rooms:                  rooms,
		notifs:                 notifs,
		studioWorkingHoursRepo: studioWorkingHoursRepo, // Инициализируем
	}
}

func (s *Service) CreateBooking(ctx context.Context, req CreateBookingRequest) (*domain.Booking, error) {
	if req.EndTime.Before(req.StartTime) || req.EndTime.Equal(req.StartTime) {
		return nil, ErrValidation
	}

	now := time.Now()
	if req.StartTime.Before(now) {
		return nil, ErrValidation
	}

	ok, err := s.bookings.CheckAvailability(ctx, req.RoomID, req.StartTime, req.EndTime)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotAvailable
	}

	pricePerHour, err := s.rooms.GetPriceByID(ctx, req.RoomID)
	if err != nil {
		return nil, err
	}

	durationHours := req.EndTime.Sub(req.StartTime).Hours()
	total := durationHours * pricePerHour
	total = math.Round(total*100) / 100

	b := &domain.Booking{
		RoomID:        req.RoomID,
		StudioID:      req.StudioID,
		UserID:        req.UserID,
		StartTime:     req.StartTime,
		EndTime:       req.EndTime,
		TotalPrice:    total,
		Status:        domain.BookingPending,
		PaymentStatus: domain.PaymentUnpaid,
		Notes:         req.Notes,
	}

	if err := s.bookings.Create(ctx, b); err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok {
			if pgErr.Code == "23505" && pgErr.ConstraintName == "idx_no_overbooking" {
				return nil, ErrOverbooking
			}
		}
		return nil, err
	}

	// уведомление владельцу студии о новом бронировании (после Create, когда b.ID уже известен)
	if s.notifs != nil {
		ownerID, _, err := s.bookings.GetStudioOwnerForBooking(ctx, b.ID)
		if err == nil && ownerID > 0 {
			_ = s.notifs.NotifyBookingCreated(ctx, ownerID, b.ID, b.StudioID, b.RoomID, b.StartTime)
		}
	}

	return b, nil
}

func (s *Service) GetBusySlots(ctx context.Context, roomID int64, from, to time.Time) ([]repository.BusySlot, error) {
	rows, err := s.bookings.GetBusySlotsForRoom(ctx, roomID, from, to)
	if err != nil {
		return nil, err
	}

	out := make([]repository.BusySlot, 0, len(rows))
	for _, r := range rows {
		out = append(out, repository.BusySlot{Start: r.Start, End: r.End})
	}
	return out, nil
}

func (s *Service) GetRoomAvailability(ctx context.Context, roomID int64, dateStr string) ([]TimeSlot, error) {
	day, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, ErrValidation
	}
	day = time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
	whRaw, err := s.rooms.GetStudioWorkingHoursByRoomID(ctx, roomID)
	if err != nil {
		return nil, err
	}
	// ДОБАВИТЬ: Fallback если working_hours пустой
	if len(whRaw) == 0 {
		// Дефолтный график: 09:00-21:00
		open := time.Date(day.Year(), day.Month(), day.Day(), 9, 0, 0, 0, time.UTC)
		close := time.Date(day.Year(), day.Month(), day.Day(), 21, 0, 0, 0, time.UTC)

		busyRepo, err := s.bookings.GetBusySlotsForRoom(ctx, roomID, open, close)
		if err != nil {
			return nil, err
		}

		busy := make([]TimeSlot, 0, len(busyRepo))
		for _, b := range busyRepo {
			busy = append(busy, TimeSlot{Start: b.Start, End: b.End})
		}

		return subtractBusy(open, close, busy), nil
	}
	// Остальной код без изменений...
	open, close, ok, err := extractOpenCloseUTC(whRaw, day)
	if err != nil {
		return nil, err
	}
	if !ok || !close.After(open) {
		return []TimeSlot{}, nil
	}
	busyRepo, err := s.bookings.GetBusySlotsForRoom(ctx, roomID, open, close)
	if err != nil {
		return nil, err
	}
	busy := make([]TimeSlot, 0, len(busyRepo))
	for _, b := range busyRepo {
		busy = append(busy, TimeSlot{Start: b.Start, End: b.End})
	}
	return subtractBusy(open, close, busy), nil
}

// GetRoomAvailabilityV2 returns booked slots format as per Task 3.2
func (s *Service) GetAvailability(ctx context.Context, roomID int64, dateStr string) (*AvailabilityResponse, error) {
	day, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, ErrValidation
	}
	day = time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)

	// Get working hours
	whRaw, err := s.rooms.GetStudioWorkingHoursByRoomID(ctx, roomID)
	if err != nil {
		return nil, err
	}

	var workingHours WorkingHours
	var open, close time.Time

	if len(whRaw) == 0 {
		// Default schedule: 09:00-21:00
		workingHours = WorkingHours{Open: "09:00", Close: "21:00"}
		open = time.Date(day.Year(), day.Month(), day.Day(), 9, 0, 0, 0, time.UTC)
		close = time.Date(day.Year(), day.Month(), day.Day(), 21, 0, 0, 0, time.UTC)
	} else {
		var ok bool
		open, close, ok, err = extractOpenCloseUTC(whRaw, day)
		if err != nil {
			return nil, err
		}
		if !ok || !close.After(open) {
			// Studio is closed on this day
			return &AvailabilityResponse{
				RoomID:       roomID,
				Date:         dateStr,
				WorkingHours: WorkingHours{Open: "", Close: ""},
				BookedSlots:  []BookedSlot{},
			}, nil
		}
		workingHours = WorkingHours{
			Open:  open.Format("15:04"),
			Close: close.Format("15:04"),
		}
	}

	// Get busy slots
	startOfDay := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := startOfDay.Add(24 * time.Hour)

	busyRepo, err := s.bookings.GetBusySlotsForRoom(ctx, roomID, startOfDay, endOfDay)
	if err != nil {
		return nil, err
	}

	// Format booked slots
	bookedSlots := make([]BookedSlot, 0)
	for _, b := range busyRepo {
		// Get the booking to find its status
		// We need to enhance this - for now we'll get all bookings
		bookedSlots = append(bookedSlots, BookedSlot{
			Start:  b.Start.Format("15:04"),
			End:    b.End.Format("15:04"),
			Status: "booked", // This could be enhanced to show actual status
		})
	}

	return &AvailabilityResponse{
		RoomID:       roomID,
		Date:         dateStr,
		WorkingHours: workingHours,
		BookedSlots:  bookedSlots,
	}, nil
}

func (s *Service) GetMyBookings(ctx context.Context, userID int64, limit, offset int) ([]BookingDetails, error) {
	rows, err := s.bookings.GetUserBookingsWithDetails(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}

	out := make([]BookingDetails, 0, len(rows))
	for _, r := range rows {
		out = append(out, BookingDetails{
			ID:         r.ID,
			Status:     r.Status,
			StartTime:  r.StartTime,
			EndTime:    r.EndTime,
			TotalPrice: r.TotalPrice,
			RoomID:     r.RoomID,
			RoomName:   r.RoomName,
			StudioID:   r.StudioID,
			StudioName: r.StudioName,
		})
	}
	return out, nil
}

func (s *Service) UpdateBookingStatus(ctx context.Context, bookingID, actorUserID int64, actorRole, newStatus string) (*domain.Booking, error) {
	if actorRole != string(domain.RoleStudioOwner) {
		return nil, ErrForbidden
	}

	ownerID, currentStatus, err := s.bookings.GetStudioOwnerForBooking(ctx, bookingID)
	if err != nil {
		return nil, err
	}
	if ownerID == 0 && currentStatus == "" {
		return nil, ErrNotFound
	}
	if ownerID != actorUserID {
		return nil, ErrForbidden
	}
	if s.notifs != nil {
		b, err := s.bookings.GetByID(ctx, bookingID)
		if err == nil && b != nil {
			if newStatus == string(domain.BookingConfirmed) {
				_ = s.notifs.NotifyBookingConfirmed(ctx, b.UserID, b.ID, b.StudioID)
			}
			if newStatus == string(domain.BookingCancelled) {
				_ = s.notifs.NotifyBookingCancelled(ctx, b.UserID, b.ID, b.StudioID, "")
			}
		}
	}

	if !(currentStatus == "pending" && newStatus == "confirmed") {
		return nil, ErrInvalidStatusTransition
	}

	if err := s.bookings.UpdateStatus(ctx, bookingID, newStatus); err != nil {
		return nil, err
	}

	b, err := s.bookings.GetByID(ctx, bookingID)
	if err != nil {
		return nil, err
	}
	return b, nil
}

type workingHoursDay struct {
	Open  string `json:"open"`
	Close string `json:"close"`
}

func extractOpenCloseUTC(whRaw json.RawMessage, day time.Time) (time.Time, time.Time, bool, error) {
	if len(whRaw) == 0 {
		return time.Time{}, time.Time{}, false, nil
	}

	var wh map[string]workingHoursDay
	if err := json.Unmarshal(whRaw, &wh); err != nil {
		return time.Time{}, time.Time{}, false, err
	}

	key := weekdayKey(day.Weekday())
	v, ok := wh[key]
	if !ok || v.Open == "" || v.Close == "" {
		return time.Time{}, time.Time{}, false, nil
	}

	openT, err := time.Parse("15:04", v.Open)
	if err != nil {
		return time.Time{}, time.Time{}, false, err
	}
	closeT, err := time.Parse("15:04", v.Close)
	if err != nil {
		return time.Time{}, time.Time{}, false, err
	}

	open := time.Date(day.Year(), day.Month(), day.Day(), openT.Hour(), openT.Minute(), 0, 0, time.UTC)
	close := time.Date(day.Year(), day.Month(), day.Day(), closeT.Hour(), closeT.Minute(), 0, 0, time.UTC)
	return open, close, true, nil
}

func weekdayKey(w time.Weekday) string {
	switch w {
	case time.Monday:
		return "monday"
	case time.Tuesday:
		return "tuesday"
	case time.Wednesday:
		return "wednesday"
	case time.Thursday:
		return "thursday"
	case time.Friday:
		return "friday"
	case time.Saturday:
		return "saturday"
	default:
		return "sunday"
	}
}

func subtractBusy(open, close time.Time, busy []TimeSlot) []TimeSlot {
	if len(busy) == 0 {
		return []TimeSlot{{Start: open, End: close}}
	}

	sort.Slice(busy, func(i, j int) bool { return busy[i].Start.Before(busy[j].Start) })

	merged := make([]TimeSlot, 0, len(busy))
	for _, s := range busy {
		if s.End.Before(open) || !s.Start.Before(close) {
			continue
		}
		if s.Start.Before(open) {
			s.Start = open
		}
		if s.End.After(close) {
			s.End = close
		}
		if !s.End.After(s.Start) {
			continue
		}

		if len(merged) == 0 {
			merged = append(merged, s)
			continue
		}
		last := merged[len(merged)-1]
		if !s.Start.After(last.End) {
			if s.End.After(last.End) {
				last.End = s.End
				merged[len(merged)-1] = last
			}
		} else {
			merged = append(merged, s)
		}
	}

	cur := open
	out := make([]TimeSlot, 0)
	for _, b := range merged {
		if b.Start.After(cur) {
			out = append(out, TimeSlot{Start: cur, End: b.Start})
		}
		if b.End.After(cur) {
			cur = b.End
		}
		if !cur.Before(close) {
			break
		}
	}
	if cur.Before(close) {
		out = append(out, TimeSlot{Start: cur, End: close})
	}
	return out
}

func (s *Service) GetBookingsByStudio(ctx context.Context, studioID int64) ([]domain.Booking, error) {
	return s.bookings.GetByStudioID(ctx, studioID)
}

func (s *Service) UpdatePaymentStatus(ctx context.Context, bookingID, ownerID int64, status domain.PaymentStatus) (*domain.Booking, error) {
	ok, err := s.bookings.IsBookingOwnedByUser(ctx, bookingID, ownerID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrForbidden
	}

	return s.bookings.UpdatePaymentStatus(ctx, bookingID, status)
}

// IsBookingStudioOwner checks if the user is the owner of the studio for this booking
func (s *Service) IsBookingStudioOwner(ctx context.Context, userID, bookingID int64) (bool, error) {
	ownerID, _, err := s.bookings.GetStudioOwnerForBooking(ctx, bookingID)
	if err != nil {
		return false, err
	}
	return ownerID == userID, nil
}

// UpdateStatus updates the booking status
func (s *Service) UpdateStatus(ctx context.Context, bookingID int64, status string) error {
	return s.bookings.UpdateStatus(ctx, bookingID, status)
}

// GetByID retrieves a booking by ID
func (s *Service) GetByID(ctx context.Context, bookingID int64) (*domain.Booking, error) {
	return s.bookings.GetByID(ctx, bookingID)
}

// CancelBooking отменяет бронирование с причиной
// Block 9: Обязательная причина отмены
func (s *Service) CancelBooking(ctx context.Context, bookingID int64, reason string) (*domain.Booking, error) {
	booking, err := s.bookings.GetByID(ctx, bookingID)
	if err != nil {
		return nil, err
	}

	// Проверяем, можно ли отменить
	if booking.Status == domain.BookingCancelled {
		return nil, ErrInvalidStatusTransition
	}

	if booking.Status == domain.BookingCompleted {
		return nil, ErrInvalidStatusTransition
	}

	// Block 9: Обновляем статус и сохраняем причину
	if err := s.bookings.CancelWithReason(ctx, bookingID, reason); err != nil {
		return nil, err
	}

	// Отправляем уведомление
	if s.notifs != nil {
		_ = s.notifs.NotifyBookingCancelled(ctx, booking.UserID, booking.ID, booking.StudioID, reason)
	}

	// Возвращаем обновлённое бронирование
	return s.bookings.GetByID(ctx, bookingID)
}

// UpdateDeposit обновляет предоплату (Block 10)
func (s *Service) UpdateDeposit(ctx context.Context, bookingID int64, amount float64) (*domain.Booking, error) {
	booking, err := s.bookings.GetByID(ctx, bookingID)
	if err != nil {
		return nil, err
	}

	if amount < 0 {
		return nil, ErrValidation
	}

	if amount > booking.TotalPrice {
		return nil, ErrValidation
	}

	// Обновляем deposit
	if err := s.bookings.UpdateDeposit(ctx, bookingID, amount); err != nil {
		return nil, err
	}

	// Если есть предоплата — подтверждаем бронь
	if amount > 0 && booking.Status == domain.BookingPending {
		if err := s.bookings.UpdateStatus(ctx, bookingID, string(domain.BookingConfirmed)); err != nil {
			return nil, err
		}
	}

	return s.bookings.GetByID(ctx, bookingID)
}

// GetRoomByID получает комнату по ID
func (s *Service) GetRoomByID(ctx context.Context, roomID int64) (*domain.Room, error) {
	// Используем поле rooms (RoomRepository)
	return s.rooms.GetByID(ctx, roomID)
}

// GetWorkingHoursForDate получает рабочие часы на конкретную дату
func (s *Service) GetWorkingHoursForDate(ctx context.Context, studioID int64, date time.Time) (*domain.WorkingHours, error) {
	// Используем studioWorkingHoursRepo
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
