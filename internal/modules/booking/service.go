package booking

import (
	"context"
	"encoding/json"
	"math"
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
	bookings BookingRepository
	rooms    RoomRepository
	notifs   NotificationSender
}

func NewService(bookings BookingRepository, rooms RoomRepository, notifs NotificationSender) *Service {
	return &Service{bookings: bookings, rooms: rooms, notifs: notifs}
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

	if s.notifs != nil {
		ownerID, _, err := s.bookings.GetStudioOwnerForBooking(ctx, b.ID)
		if err == nil && ownerID > 0 {
			_ = s.notifs.NotifyBookingCreated(ctx, ownerID, b.ID, b.StudioID, b.RoomID, b.StartTime)
		}
	}

	if err := s.bookings.Create(ctx, b); err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok {
			if pgErr.Code == "23505" && pgErr.ConstraintName == "idx_no_overbooking" {
				return nil, ErrOverbooking
			}
		}
		return nil, err
	}

	return b, nil

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
