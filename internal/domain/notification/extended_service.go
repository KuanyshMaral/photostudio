package notification

import (
	"context"
	"fmt"
	"log"
)

// ExternalServices holds references to email, push and other services
type ExternalServices struct {
	EmailService interface{} // Email service implementation
	PushService  interface{} // Push notification service implementation
}

// ExtendedService handles notifications with external integrations
type ExtendedService struct {
	service *Service
	external *ExternalServices
}

// NewExtendedService creates an extended notification service with integrations
func NewExtendedService(service *Service, external *ExternalServices) *ExtendedService {
	return &ExtendedService{
		service:  service,
		external: external,
	}
}

// SendNotificationWithChannels sends notification through specified channels
func (s *ExtendedService) SendNotificationWithChannels(
	ctx context.Context,
	userID int64,
	notifType Type,
	title, body string,
	data *NotificationData,
	channels []string,
) (*Notification, error) {
	// Create in-app notification
	n, err := s.service.Create(ctx, userID, notifType, title, body, data)
	if err != nil {
		return nil, err
	}

	// Get user preferences
	prefs, err := s.service.GetPreferences(ctx, userID)
	if err != nil {
		log.Printf("Failed to get preferences for user %d: %v", userID, err)
	}

	// Send through requested channels if enabled
	for _, ch := range channels {
		switch ch {
		case "email":
			if prefs != nil && prefs.EmailEnabled {
				// Email sending would be implemented here
				// s.external.EmailService.Send(...)
				log.Printf("Would send email notification to user %d: %s", userID, title)
			}
		case "push":
			if prefs != nil && prefs.PushEnabled {
				// Push sending would be implemented here
				// s.external.PushService.Send(...)
				log.Printf("Would send push notification to user %d: %s", userID, title)
			}
		case "in_app":
			// Already created above
			log.Printf("In-app notification created for user %d: %s", userID, title)
		}
	}

	return n, nil
}

// NotifyBookingCreatedWithChannels sends booking created notification through configured channels
func (s *ExtendedService) NotifyBookingCreatedWithChannels(
	ctx context.Context,
	ownerID int64,
	bookingID, studioID, roomID int64,
	startTime string,
) error {
	prefs, err := s.service.GetPreferences(ctx, ownerID)
	if err != nil {
		return err
	}

	channels := []string{}
	if prefs.InAppEnabled {
		channels = append(channels, "in_app")
	}
	if prefs.EmailEnabled {
		channels = append(channels, "email")
	}
	if prefs.PushEnabled {
		channels = append(channels, "push")
	}

	startTimeStr := startTime
	_, err = s.SendNotificationWithChannels(
		ctx,
		ownerID,
		TypeBookingCreated,
		"Новое бронирование",
		fmt.Sprintf("Поступило новое бронирование на %s", startTime),
		&NotificationData{
			BookingID: &bookingID,
			StudioID:  &studioID,
			RoomID:    &roomID,
			StartTime: &startTimeStr,
		},
		channels,
	)

	return err
}

// NotifyBookingConfirmedWithChannels sends booking confirmed notification
func (s *ExtendedService) NotifyBookingConfirmedWithChannels(
	ctx context.Context,
	clientID int64,
	bookingID, studioID int64,
) error {
	prefs, err := s.service.GetPreferences(ctx, clientID)
	if err != nil {
		return err
	}

	channels := []string{}
	if prefs.InAppEnabled {
		channels = append(channels, "in_app")
	}
	if prefs.EmailEnabled {
		channels = append(channels, "email")
	}
	if prefs.PushEnabled {
		channels = append(channels, "push")
	}

	_, err = s.SendNotificationWithChannels(
		ctx,
		clientID,
		TypeBookingConfirmed,
		"Бронирование подтверждено",
		"Ваше бронирование подтверждено владельцем студии",
		&NotificationData{
			BookingID: &bookingID,
			StudioID:  &studioID,
		},
		channels,
	)

	return err
}

// NotifyBookingCancelledWithChannels sends booking cancelled notification
func (s *ExtendedService) NotifyBookingCancelledWithChannels(
	ctx context.Context,
	clientID int64,
	bookingID, studioID int64,
	reason string,
) error {
	prefs, err := s.service.GetPreferences(ctx, clientID)
	if err != nil {
		return err
	}

	channels := []string{}
	if prefs.InAppEnabled {
		channels = append(channels, "in_app")
	}
	if prefs.EmailEnabled {
		channels = append(channels, "email")
	}
	if prefs.PushEnabled {
		channels = append(channels, "push")
	}

	msg := "Ваше бронирование отменено владельцем студии"
	if reason != "" {
		msg = msg + ". Причина: " + reason
	}

	_, err = s.SendNotificationWithChannels(
		ctx,
		clientID,
		TypeBookingCancelled,
		"Бронирование отменено",
		msg,
		&NotificationData{
			BookingID:          &bookingID,
			StudioID:           &studioID,
			CancellationReason: &reason,
		},
		channels,
	)

	return err
}

// NotifyVerificationApprovedWithChannels sends verification approved notification
func (s *ExtendedService) NotifyVerificationApprovedWithChannels(
	ctx context.Context,
	ownerID int64,
	studioID int64,
) error {
	prefs, err := s.service.GetPreferences(ctx, ownerID)
	if err != nil {
		return err
	}

	channels := []string{}
	if prefs.InAppEnabled {
		channels = append(channels, "in_app")
	}
	if prefs.EmailEnabled {
		channels = append(channels, "email")
	}
	if prefs.PushEnabled {
		channels = append(channels, "push")
	}

	_, err = s.SendNotificationWithChannels(
		ctx,
		ownerID,
		TypeVerificationApproved,
		"Верификация одобрена",
		"Ваша студия успешно прошла верификацию",
		&NotificationData{
			StudioID: &studioID,
		},
		channels,
	)

	return err
}

// NotifyVerificationRejectedWithChannels sends verification rejected notification
func (s *ExtendedService) NotifyVerificationRejectedWithChannels(
	ctx context.Context,
	ownerID int64,
	studioID int64,
	reason string,
) error {
	prefs, err := s.service.GetPreferences(ctx, ownerID)
	if err != nil {
		return err
	}

	channels := []string{}
	if prefs.InAppEnabled {
		channels = append(channels, "in_app")
	}
	if prefs.EmailEnabled {
		channels = append(channels, "email")
	}

	msg := "Ваша заявка на верификацию отклонена"
	if reason != "" {
		msg = msg + ". Причина: " + reason
	}

	_, err = s.SendNotificationWithChannels(
		ctx,
		ownerID,
		TypeVerificationRejected,
		"Верификация отклонена",
		msg,
		&NotificationData{
			StudioID: &studioID,
			Reason:   &reason,
		},
		channels,
	)

	return err
}

// NotifyNewReviewWithChannels sends new review notification
func (s *ExtendedService) NotifyNewReviewWithChannels(
	ctx context.Context,
	ownerID int64,
	studioID, reviewID int64,
	rating int,
) error {
	prefs, err := s.service.GetPreferences(ctx, ownerID)
	if err != nil {
		return err
	}

	channels := []string{}
	if prefs.InAppEnabled {
		channels = append(channels, "in_app")
	}
	if prefs.EmailEnabled {
		channels = append(channels, "email")
	}
	if prefs.PushEnabled {
		channels = append(channels, "push")
	}

	_, err = s.SendNotificationWithChannels(
		ctx,
		ownerID,
		TypeNewReview,
		"Новый отзыв",
		fmt.Sprintf("Поступил новый отзыв с оценкой %d ⭐", rating),
		&NotificationData{
			StudioID: &studioID,
			ReviewID: &reviewID,
			Rating:   &rating,
		},
		channels,
	)

	return err
}

// NotifyNewMessageWithChannels sends new message notification
func (s *ExtendedService) NotifyNewMessageWithChannels(
	ctx context.Context,
	userID int64,
	senderName, preview string,
	chatRoomID, messageID int64,
) error {
	prefs, err := s.service.GetPreferences(ctx, userID)
	if err != nil {
		return err
	}

	// For messages, typically don't send email, only in-app and push
	channels := []string{}
	if prefs.InAppEnabled {
		channels = append(channels, "in_app")
	}
	if prefs.PushEnabled {
		channels = append(channels, "push")
	}

	_, err = s.SendNotificationWithChannels(
		ctx,
		userID,
		TypeNewMessage,
		fmt.Sprintf("Новое сообщение от %s", senderName),
		preview,
		&NotificationData{
			ChatRoomID:     &chatRoomID,
			MessageID:      &messageID,
			SenderName:     &senderName,
			MessagePreview: &preview,
		},
		channels,
	)

	return err
}

// BulkNotify sends notification to multiple users
func (s *ExtendedService) BulkNotify(
	ctx context.Context,
	userIDs []int64,
	notifType Type,
	title, body string,
	data *NotificationData,
) (successful, failed int) {
	for _, userID := range userIDs {
		if _, err := s.service.Create(ctx, userID, notifType, title, body, data); err != nil {
			failed++
			log.Printf("Failed to notify user %d: %v", userID, err)
		} else {
			successful++
		}
	}
	return
}

// GetService returns underlying service
func (s *ExtendedService) GetService() *Service {
	return s.service
}
