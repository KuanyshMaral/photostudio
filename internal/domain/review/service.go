package review

import (
	"context"
	"encoding/json"
	"errors"
	"photostudio/internal/domain/catalog"

	"gorm.io/gorm"
)

type BookingGate interface {
	HasCompletedBookingForStudio(ctx context.Context, userID, studioID int64) (bool, error)
}

type StudioGate interface {
	GetByID(ctx context.Context, id int64) (*catalog.Studio, error)
}

type Service struct {
	reviews  *ReviewRepository
	bookings BookingGate
	studios  StudioGate
}

func NewService(reviews *ReviewRepository, bookings BookingGate, studios StudioGate) *Service {
	return &Service{reviews: reviews, bookings: bookings, studios: studios}
}

func (s *Service) Create(ctx context.Context, userID int64, req CreateReviewRequest) (*Review, error) {
	targetType := TargetType(req.TargetType)

	if targetType == TargetTypeStudio {
		hasCompleted, err := s.bookings.HasCompletedBookingForStudio(ctx, userID, req.TargetID)
		if err != nil {
			return nil, err
		}
		if !hasCompleted {
			return nil, errors.New("you must have a completed booking to leave a review for a studio")
		}
	}

	exists, err := s.reviews.HasReviewed(ctx, userID, targetType, req.TargetID, req.ContextType, req.ContextID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("you have already reviewed this entity in this context")
	}

	if req.Rating < 1 || req.Rating > 5 {
		return nil, errors.New("rating must be between 1 and 5")
	}

	var criteriaBytes []byte = []byte("{}")
	if len(req.Criteria) > 0 {
		if b, err := json.Marshal(req.Criteria); err == nil {
			criteriaBytes = b
		}
	}

	review := &Review{
		AuthorID:    userID,
		TargetType:  targetType,
		TargetID:    req.TargetID,
		ContextType: req.ContextType,
		ContextID:   req.ContextID,
		Rating:      req.Rating,
		Comment:     req.Comment,
		Photos:      req.Photos,
		Criteria:    criteriaBytes,
	}

	if err := s.reviews.Create(ctx, review); err != nil {
		return nil, err
	}

	return review, nil
}

func (s *Service) GetByTarget(ctx context.Context, targetType string, targetID int64, limit, offset int) ([]Review, error) {
	if targetID <= 0 || targetType == "" {
		return nil, ErrInvalidRequest
	}
	return s.reviews.GetByTarget(ctx, TargetType(targetType), targetID, limit, offset)
}

func (s *Service) AddOwnerResponse(ctx context.Context, reviewID, userID int64, response string) (*Review, error) {
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

	if rv.TargetType == TargetTypeStudio {
		st, err := s.studios.GetByID(ctx, rv.TargetID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrNotFound
			}
			return nil, err
		}

		if st.OwnerID != userID {
			return nil, ErrForbidden
		}
	} else {
		return nil, errors.New("cannot add owner response for this target type currently")
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
