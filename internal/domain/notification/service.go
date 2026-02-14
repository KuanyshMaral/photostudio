package notification

import (
	"context"
	"fmt"
	"time"
)

type Service struct {
	repo *NotificationRepository
}

func NewService(repo *NotificationRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, userID int64, t NotificationType, title, message string, data map[string]any) error {
	n := &Notification{
		UserID:  userID,
		Type:    t,
		Title:   title,
		Message: message,
		IsRead:  false,
	}
	return s.repo.Create(ctx, n, data)
}

func (s *Service) GetUserNotifications(ctx context.Context, userID int64, limit int) ([]Notification, int64, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	list, err := s.repo.GetByUserID(ctx, userID, limit)
	if err != nil {
		return nil, 0, err
	}

	unread, err := s.repo.CountUnread(ctx, userID)
	if err != nil {
		unread = 0
	}

	return list, unread, nil
}

func (s *Service) MarkAsRead(ctx context.Context, notificationID, userID int64) error {
	return s.repo.MarkAsRead(ctx, notificationID, userID)
}

func (s *Service) MarkAllAsRead(ctx context.Context, userID int64) error {
	return s.repo.MarkAllAsRead(ctx, userID)
}

func (s *Service) NotifyBookingCreated(ctx context.Context, ownerUserID, bookingID, studioID, roomID int64, start time.Time) error {
	return s.Create(
		ctx,
		ownerUserID,
		NotifBookingCreated,
		"Новое бронирование",
		fmt.Sprintf("Поступило новое бронирование на %s", start.Format("02.01.2006 15:04")),
		map[string]any{
			"booking_id": bookingID,
			"studio_id":  studioID,
			"room_id":    roomID,
		},
	)
}

func (s *Service) NotifyBookingConfirmed(ctx context.Context, clientUserID, bookingID, studioID int64) error {
	return s.Create(
		ctx,
		clientUserID,
		NotifBookingConfirmed,
		"Бронирование подтверждено",
		"Ваше бронирование подтверждено владельцем студии",
		map[string]any{
			"booking_id": bookingID,
			"studio_id":  studioID,
		},
	)
}

func (s *Service) NotifyBookingCancelled(ctx context.Context, clientUserID, bookingID, studioID int64, reason string) error {
	msg := "Ваше бронирование отменено владельцем студии"
	if reason != "" {
		msg = msg + ". Причина: " + reason
	}
	return s.Create(
		ctx,
		clientUserID,
		NotifBookingCancelled,
		"Бронирование отменено",
		msg,
		map[string]any{
			"booking_id": bookingID,
			"studio_id":  studioID,
		},
	)
}

func (s *Service) NotifyVerificationApproved(ctx context.Context, ownerUserID, studioID int64) error {
	return s.Create(
		ctx,
		ownerUserID,
		NotifVerificationApproved,
		"Верификация одобрена",
		"Ваша студия успешно прошла верификацию",
		map[string]any{
			"studio_id": studioID,
		},
	)
}

func (s *Service) NotifyVerificationRejected(ctx context.Context, ownerUserID, studioID int64, reason string) error {
	msg := "Ваша заявка на верификацию отклонена"
	if reason != "" {
		msg = msg + ". Причина: " + reason
	}
	return s.Create(
		ctx,
		ownerUserID,
		NotifVerificationRejected,
		"Верификация отклонена",
		msg,
		map[string]any{
			"studio_id": studioID,
		},
	)
}

func (s *Service) NotifyNewReview(ctx context.Context, ownerUserID, studioID, reviewID int64, rating int) error {
	return s.Create(
		ctx,
		ownerUserID,
		NotifNewReview,
		"Новый отзыв",
		fmt.Sprintf("Поступил новый отзыв с оценкой %d", rating),
		map[string]any{
			"studio_id": studioID,
			"review_id": reviewID,
			"rating":    rating,
		},
	)
}
