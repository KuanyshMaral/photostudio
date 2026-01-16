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
	// 1. Проверяем completed booking
	hasCompleted, err := s.bookings.HasCompletedBookingForStudio(ctx, userID, req.StudioID)
	if err != nil {
		return nil, err
	}
	if !hasCompleted {
		return nil, errors.New("you must have a completed booking to leave a review")
	}

	// 2. Проверяем что отзыв ещё не оставлен
	exists, err := s.reviews.ExistsByUserAndStudio(ctx, userID, req.StudioID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("you have already reviewed this studio")
	}

	// 3. Валидация рейтинга
	if req.Rating < 1 || req.Rating > 5 {
		return nil, errors.New("rating must be between 1 and 5")
	}

	// 4. Создаём отзыв
	review := &domain.Review{
		UserID:   userID,
		StudioID: req.StudioID,
		Rating:   req.Rating,
		Comment:  req.Comment,
	}

	if err := s.reviews.Create(ctx, review); err != nil {
		return nil, err
	}

	return review, nil
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
