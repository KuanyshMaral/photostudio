package review

import (
	"context"
	"errors"

	"photostudio/internal/domain"
	"photostudio/internal/repository"

	"gorm.io/gorm"
)

type BookingGate interface {
	HasCompletedBookingForStudio(ctx context.Context, userID, studioID int64) (bool, error)
}

type StudioGate interface {
	GetByID(ctx context.Context, id int64) (*domain.Studio, error)
}

type Service struct {
	reviews  *repository.ReviewRepository
	bookings BookingGate
	studios  StudioGate
}

func NewService(reviews *repository.ReviewRepository, bookings BookingGate, studios StudioGate) *Service {
	return &Service{reviews: reviews, bookings: bookings, studios: studios}
}

func (s *Service) Create(ctx context.Context, userID int64, req CreateReviewRequest) (*domain.Review, error) {
	if userID <= 0 || req.StudioID <= 0 || req.Rating < 1 || req.Rating > 5 {
		return nil, ErrInvalidRequest
	}

	ok, err := s.bookings.HasCompletedBookingForStudio(ctx, userID, req.StudioID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrReviewNotAllowed
	}

	rv := &domain.Review{
		StudioID:  req.StudioID,
		UserID:    userID,
		BookingID: req.BookingID,
		Rating:    req.Rating,
		Comment:   req.Comment,
		Photos:    req.Photos,
	}

	if err := s.reviews.Create(ctx, rv); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, err
	}
	return rv, nil
}

func (s *Service) GetByStudio(ctx context.Context, studioID int64, limit, offset int) ([]domain.Review, error) {
	if studioID <= 0 {
		return nil, ErrInvalidRequest
	}
	return s.reviews.GetByStudio(ctx, studioID, limit, offset)
}

func (s *Service) AddOwnerResponse(ctx context.Context, reviewID, userID int64, response string) (*domain.Review, error) {
	if reviewID <= 0 || userID <= 0 || response == "" {
		return nil, ErrInvalidRequest
	}

	rv, err := s.reviews.GetByID(ctx, reviewID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	st, err := s.studios.GetByID(ctx, rv.StudioID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if st.OwnerID != userID {
		return nil, ErrForbidden
	}

	updated, err := s.reviews.SetOwnerResponse(ctx, reviewID, response)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return updated, nil
}

func isUniqueViolation(err error) bool {
	s := err.Error()
	return contains(s, "duplicate key value violates unique constraint") ||
		contains(s, "SQLSTATE 23505") ||
		contains(s, "23505")
}

func contains(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
