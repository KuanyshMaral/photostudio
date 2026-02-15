package notification

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Service handles notification business logic
type Service struct {
	notifRepo       Repository
	prefRepo        PreferencesRepository
	deviceTokenRepo DeviceTokenRepository
}

// NewService creates notification service
func NewService(repo Repository, prefRepo PreferencesRepository, deviceTokenRepo DeviceTokenRepository) *Service {
	return &Service{
		notifRepo:       repo,
		prefRepo:        prefRepo,
		deviceTokenRepo: deviceTokenRepo,
	}
}

// NewServiceLegacy creates legacy service for backward compatibility
func NewServiceLegacy(repo *NotificationRepository) *Service {
	return &Service{
		notifRepo: &legacyRepositoryAdapter{repo},
	}
}

// Create creates a notification
func (s *Service) Create(ctx context.Context, userID int64, notifType Type, title, body string, data *NotificationData) (*Notification, error) {
	n := &Notification{
		UserID:    userID,
		Type:      notifType,
		Title:     title,
		IsRead:    false,
		CreatedAt: time.Now(),
	}

	if body != "" {
		n.Body = sql.NullString{String: body, Valid: true}
	}
	if err := n.SetData(data); err != nil {
		return nil, err
	}

	if err := s.notifRepo.Create(ctx, n); err != nil {
		return nil, err
	}

	return n, nil
}

// List returns notifications for user with pagination
func (s *Service) List(ctx context.Context, userID int64, limit, offset int) ([]*Notification, int64, int64, error) {
	// Validate limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	notifications, err := s.notifRepo.ListByUser(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, 0, err
	}

	unread, err := s.notifRepo.CountUnreadByUser(ctx, userID)
	if err != nil {
		unread = 0
	}

	total, err := s.notifRepo.CountByUser(ctx, userID)
	if err != nil {
		total = 0
	}

	return notifications, unread, total, nil
}

// GetUnreadCount returns unread count
func (s *Service) GetUnreadCount(ctx context.Context, userID int64) (int64, error) {
	return s.notifRepo.CountUnreadByUser(ctx, userID)
}

// MarkAsRead marks single notification as read
func (s *Service) MarkAsRead(ctx context.Context, id int64) error {
	return s.notifRepo.MarkAsRead(ctx, id)
}

// MarkAllAsRead marks all notifications as read for user
func (s *Service) MarkAllAsRead(ctx context.Context, userID int64) error {
	return s.notifRepo.MarkAllAsRead(ctx, userID)
}

// Delete removes a notification
func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.notifRepo.Delete(ctx, id)
}

// DeleteOlder removes old notifications
func (s *Service) DeleteOlder(ctx context.Context, days int) (int64, error) {
	age := time.Duration(days*24) * time.Hour
	return s.notifRepo.DeleteOlderThan(ctx, age)
}

// --- Specialized Notification Methods ---

// NotifyBookingCreated notifies owner about new booking
func (s *Service) NotifyBookingCreated(ctx context.Context, ownerID int64, bookingID, studioID, roomID int64, startTime time.Time) error {
	startTimeStr := startTime.Format(time.RFC3339)
	_, err := s.Create(ctx, ownerID, TypeBookingCreated,
		"Новое бронирование",
		fmt.Sprintf("Поступило новое бронирование на %s", startTime.Format("02.01.2006 15:04")),
		&NotificationData{
			BookingID: &bookingID,
			StudioID:  &studioID,
			RoomID:    &roomID,
			StartTime: &startTimeStr,
		},
	)
	return err
}

// NotifyBookingConfirmed notifies client that booking was confirmed
func (s *Service) NotifyBookingConfirmed(ctx context.Context, clientID int64, bookingID, studioID int64) error {
	_, err := s.Create(ctx, clientID, TypeBookingConfirmed,
		"Бронирование подтверждено",
		"Ваше бронирование подтверждено владельцем студии",
		&NotificationData{
			BookingID: &bookingID,
			StudioID:  &studioID,
		},
	)
	return err
}

// NotifyBookingCancelled notifies client that booking was cancelled
func (s *Service) NotifyBookingCancelled(ctx context.Context, clientID int64, bookingID, studioID int64, reason string) error {
	msg := "Ваше бронирование отменено владельцем студии"
	if reason != "" {
		msg = msg + ". Причина: " + reason
	}

	_, err := s.Create(ctx, clientID, TypeBookingCancelled,
		"Бронирование отменено",
		msg,
		&NotificationData{
			BookingID: &bookingID,
			StudioID:  &studioID,
			Reason:    &reason,
		},
	)
	return err
}

// NotifyBookingCompleted notifies both owner and client when booking is completed
func (s *Service) NotifyBookingCompleted(ctx context.Context, userID int64, bookingID, studioID int64) error {
	_, err := s.Create(ctx, userID, TypeBookingCompleted,
		"Бронирование завершено",
		"Бронирование успешно завершено",
		&NotificationData{
			BookingID: &bookingID,
			StudioID:  &studioID,
		},
	)
	return err
}

// NotifyVerificationApproved notifies owner that studio was verified
func (s *Service) NotifyVerificationApproved(ctx context.Context, ownerID int64, studioID int64) error {
	_, err := s.Create(ctx, ownerID, TypeVerificationApproved,
		"Верификация одобрена",
		"Ваша студия успешно прошла верификацию",
		&NotificationData{
			StudioID: &studioID,
		},
	)
	return err
}

// NotifyVerificationRejected notifies owner that studio verification was rejected
func (s *Service) NotifyVerificationRejected(ctx context.Context, ownerID int64, studioID int64, reason string) error {
	msg := "Ваша заявка на верификацию отклонена"
	if reason != "" {
		msg = msg + ". Причина: " + reason
	}

	_, err := s.Create(ctx, ownerID, TypeVerificationRejected,
		"Верификация отклонена",
		msg,
		&NotificationData{
			StudioID: &studioID,
			Reason:   &reason,
		},
	)
	return err
}

// NotifyNewReview notifies owner about new review
func (s *Service) NotifyNewReview(ctx context.Context, ownerID int64, studioID, reviewID int64, rating int) error {
	_, err := s.Create(ctx, ownerID, TypeNewReview,
		"Новый отзыв",
		fmt.Sprintf("Поступил новый отзыв с оценкой %d ⭐", rating),
		&NotificationData{
			StudioID: &studioID,
			ReviewID: &reviewID,
			Rating:   &rating,
		},
	)
	return err
}

// NotifyNewMessage notifies user about new message
func (s *Service) NotifyNewMessage(ctx context.Context, userID int64, senderName, preview string, chatRoomID, messageID int64) error {
	_, err := s.Create(ctx, userID, TypeNewMessage,
		fmt.Sprintf("Новое сообщение от %s", senderName),
		preview,
		&NotificationData{
			ChatRoomID:     &chatRoomID,
			MessageID:      &messageID,
			SenderName:     &senderName,
			MessagePreview: &preview,
		},
	)
	return err
}

// NotifyEquipmentBooked notifies owner when equipment is booked
func (s *Service) NotifyEquipmentBooked(ctx context.Context, ownerID int64, equipmentID, bookingID int64, equipmentName string) error {
	_, err := s.Create(ctx, ownerID, TypeEquipmentBooked,
		"Оборудование забронировано",
		fmt.Sprintf("Оборудование '%s' забронировано", equipmentName),
		&NotificationData{
			EquipmentID: &equipmentID,
			BookingID:   &bookingID,
		},
	)
	return err
}

// NotifyStudioUpdated notifies followers that studio was updated
func (s *Service) NotifyStudioUpdated(ctx context.Context, userID int64, studioID int64, studioName string) error {
	_, err := s.Create(ctx, userID, TypeStudioUpdated,
		"Студия обновлена",
		fmt.Sprintf("Студия '%s' обновила свою информацию", studioName),
		&NotificationData{
			StudioID: &studioID,
		},
	)
	return err
}

// --- Preferences Management ---

// GetPreferences returns user notification preferences
func (s *Service) GetPreferences(ctx context.Context, userID int64) (*UserPreferences, error) {
	if s.prefRepo == nil {
		return GetDefaultPreferences(userID), nil
	}

	prefs, err := s.prefRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// If not found, create default and save
	if prefs == nil {
		prefs = GetDefaultPreferences(userID)
		_ = s.prefRepo.Create(ctx, prefs)
	}

	return prefs, nil
}

// UpdatePreferences updates user notification preferences
func (s *Service) UpdatePreferences(ctx context.Context, userID int64, updates *UserPreferences) (*UserPreferences, error) {
	if s.prefRepo == nil {
		return nil, fmt.Errorf("preferences repository not initialized")
	}

	prefs, err := s.GetPreferences(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Update fields
	if updates.EmailEnabled != prefs.EmailEnabled {
		prefs.EmailEnabled = updates.EmailEnabled
	}
	if updates.PushEnabled != prefs.PushEnabled {
		prefs.PushEnabled = updates.PushEnabled
	}
	if updates.InAppEnabled != prefs.InAppEnabled {
		prefs.InAppEnabled = updates.InAppEnabled
	}
	if updates.DigestEnabled != prefs.DigestEnabled {
		prefs.DigestEnabled = updates.DigestEnabled
	}
	if updates.DigestFrequency != "" {
		prefs.DigestFrequency = updates.DigestFrequency
	}
	if len(updates.PerTypeSettings) > 0 {
		prefs.PerTypeSettings = updates.PerTypeSettings
	}

	if err := s.prefRepo.Update(ctx, prefs); err != nil {
		return nil, err
	}

	return prefs, nil
}

// ResetPreferences resets user preferences to defaults
func (s *Service) ResetPreferences(ctx context.Context, userID int64) (*UserPreferences, error) {
	if s.prefRepo == nil {
		return GetDefaultPreferences(userID), nil
	}

	if err := s.prefRepo.ResetToDefaults(ctx, userID); err != nil {
		return nil, err
	}

	return s.GetPreferences(ctx, userID)
}

// --- Device Tokens Management ---

// RegisterDeviceToken registers a new device token for push notifications
func (s *Service) RegisterDeviceToken(ctx context.Context, userID int64, token, platform, deviceName string) (*DeviceToken, error) {
	if s.deviceTokenRepo == nil {
		return nil, fmt.Errorf("device token repository not initialized")
	}

	dt := &DeviceToken{
		UserID:     userID,
		Token:      token,
		Platform:   platform,
		DeviceName: deviceName,
		IsActive:   true,
		CreatedAt:  time.Now(),
		LastUsedAt: time.Now(),
	}

	if err := s.deviceTokenRepo.Create(ctx, dt); err != nil {
		return nil, err
	}

	return dt, nil
}

// ListDeviceTokens returns all active device tokens for user
func (s *Service) ListDeviceTokens(ctx context.Context, userID int64) ([]*DeviceToken, error) {
	if s.deviceTokenRepo == nil {
		return nil, fmt.Errorf("device token repository not initialized")
	}

	return s.deviceTokenRepo.ListByUser(ctx, userID, true)
}

// DeactivateDeviceToken deactivates a device token
func (s *Service) DeactivateDeviceToken(ctx context.Context, id int64) error {
	if s.deviceTokenRepo == nil {
		return fmt.Errorf("device token repository not initialized")
	}

	return s.deviceTokenRepo.Deactivate(ctx, id)
}

// UpdateDeviceTokenUsage updates last used timestamp
func (s *Service) UpdateDeviceTokenUsage(ctx context.Context, token string) error {
	if s.deviceTokenRepo == nil {
		return nil
	}

	dt, err := s.deviceTokenRepo.GetByToken(ctx, token)
	if err != nil {
		return err
	}
	if dt == nil {
		return fmt.Errorf("device token not found")
	}

	dt.UpdateLastUsed()
	return s.deviceTokenRepo.Update(ctx, dt)
}

// CleanupInactiveTokens removes inactive device tokens older than duration
func (s *Service) CleanupInactiveTokens(ctx context.Context, olderThan time.Duration) (int64, error) {
	if s.deviceTokenRepo == nil {
		return 0, fmt.Errorf("device token repository not initialized")
	}

	return s.deviceTokenRepo.DeleteInactive(ctx, olderThan)
}

// --- Legacy compatibility adapter ---

type legacyRepositoryAdapter struct {
	repo *NotificationRepository
}

func (a *legacyRepositoryAdapter) Create(ctx context.Context, n *Notification) error {
	return a.repo.Create(ctx, n, nil)
}

func (a *legacyRepositoryAdapter) GetByID(ctx context.Context, id int64) (*Notification, error) {
	// Not implemented in legacy
	return nil, nil
}

func (a *legacyRepositoryAdapter) ListByUser(ctx context.Context, userID int64, limit, offset int) ([]*Notification, error) {
	notifications, err := a.repo.GetByUserID(ctx, userID, limit)
	converted := make([]*Notification, len(notifications))
	for i := range notifications {
		converted[i] = &notifications[i]
	}
	return converted, err
}

func (a *legacyRepositoryAdapter) CountByUser(ctx context.Context, userID int64) (int64, error) {
	// Not implemented
	return 0, nil
}

func (a *legacyRepositoryAdapter) CountUnreadByUser(ctx context.Context, userID int64) (int64, error) {
	return a.repo.CountUnread(ctx, userID)
}

func (a *legacyRepositoryAdapter) MarkAsRead(ctx context.Context, id int64) error {
	// Need userID - not available, so use 0
	return a.repo.MarkAsRead(ctx, id, 0)
}

func (a *legacyRepositoryAdapter) MarkAllAsRead(ctx context.Context, userID int64) error {
	return a.repo.MarkAllAsRead(ctx, userID)
}

func (a *legacyRepositoryAdapter) Delete(ctx context.Context, id int64) error {
	return nil
}

func (a *legacyRepositoryAdapter) DeleteByUser(ctx context.Context, userID int64) error {
	return nil
}

func (a *legacyRepositoryAdapter) DeleteOldByUser(ctx context.Context, userID int64, days int) (int64, error) {
	return 0, nil
}

func (a *legacyRepositoryAdapter) DeleteOlderThan(ctx context.Context, age time.Duration) (int64, error) {
	return 0, nil
}