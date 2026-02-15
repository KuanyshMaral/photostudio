package notification

import (
	"context"
	"log"
	"time"
)

// CleanupService handles background cleanup tasks for notifications
type CleanupService struct {
	repo Repository
	deviceTokenRepo DeviceTokenRepository
}

// NewCleanupService creates cleanup service
func NewCleanupService(repo Repository, deviceTokenRepo DeviceTokenRepository) *CleanupService {
	return &CleanupService{
		repo:            repo,
		deviceTokenRepo: deviceTokenRepo,
	}
}

// CleanupOldNotifications removes notifications older than specified days
func (c *CleanupService) CleanupOldNotifications(ctx context.Context, daysToKeep int) error {
	startTime := time.Now()

	deleted, err := c.repo.DeleteOlderThan(ctx, time.Duration(daysToKeep*24)*time.Hour)
	if err != nil {
		log.Printf("Error cleaning up old notifications: %v", err)
		return err
	}

	duration := time.Since(startTime)
	log.Printf("Cleanup completed: deleted %d old notifications in %v", deleted, duration)

	return nil
}

// CleanupUserNotifications removes old notifications for a specific user
func (c *CleanupService) CleanupUserNotifications(ctx context.Context, userID int64, daysToKeep int) error {
	startTime := time.Now()

	deleted, err := c.repo.DeleteOldByUser(ctx, userID, daysToKeep)
	if err != nil {
		log.Printf("Error cleaning up notifications for user %d: %v", userID, err)
		return err
	}

	duration := time.Since(startTime)
	log.Printf("Cleanup completed for user %d: deleted %d old notifications in %v", userID, deleted, duration)

	return nil
}

// CleanupInactiveDeviceTokens removes device tokens that haven't been used recently
func (c *CleanupService) CleanupInactiveDeviceTokens(ctx context.Context, inactiveDays int) error {
	if c.deviceTokenRepo == nil {
		return nil
	}

	startTime := time.Now()

	deleted, err := c.deviceTokenRepo.DeleteInactive(ctx, time.Duration(inactiveDays*24)*time.Hour)
	if err != nil {
		log.Printf("Error cleaning up inactive device tokens: %v", err)
		return err
	}

	duration := time.Since(startTime)
	log.Printf("Cleanup completed: deleted %d inactive device tokens in %v", deleted, duration)

	return nil
}

// RunScheduledCleanup runs all cleanup tasks
func (c *CleanupService) RunScheduledCleanup(ctx context.Context, config CleanupConfig) error {
	log.Println("Starting scheduled cleanup tasks...")

	startTime := time.Now()

	// Cleanup old notifications
	if err := c.CleanupOldNotifications(ctx, config.NotificationRetentionDays); err != nil {
		log.Printf("Warning: notification cleanup failed: %v", err)
	}

	// Cleanup inactive device tokens
	if err := c.CleanupInactiveDeviceTokens(ctx, config.DeviceTokenInactivityDays); err != nil {
		log.Printf("Warning: device token cleanup failed: %v", err)
	}

	duration := time.Since(startTime)
	log.Printf("All cleanup tasks completed in %v", duration)

	return nil
}

// CleanupConfig holds configuration for cleanup tasks
type CleanupConfig struct {
	NotificationRetentionDays     int           // Keep notifications for N days (default: 90)
	DeviceTokenInactivityDays     int           // Remove unused tokens after N days (default: 90)
	CleanupInterval               time.Duration // How often to run cleanup (default: 24h)
	EnableAutomaticCleanup        bool          // Enable automatic cleanup via goroutine
}

// DefaultCleanupConfig returns default cleanup configuration
func DefaultCleanupConfig() CleanupConfig {
	return CleanupConfig{
		NotificationRetentionDays: 90,
		DeviceTokenInactivityDays: 90,
		CleanupInterval:           24 * time.Hour,
		EnableAutomaticCleanup:    true,
	}
}

// ScheduleCleanup starts a background goroutine for periodic cleanup
func (c *CleanupService) ScheduleCleanup(ctx context.Context, config CleanupConfig) chan struct{} {
	if !config.EnableAutomaticCleanup {
		log.Println("Automatic cleanup is disabled")
		return nil
	}

	stopCh := make(chan struct{})

	go func() {
		ticker := time.NewTicker(config.CleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := c.RunScheduledCleanup(ctx, config); err != nil {
					log.Printf("Scheduled cleanup error: %v", err)
				}
			case <-stopCh:
				log.Println("Scheduled cleanup stopped")
				return
			case <-ctx.Done():
				log.Println("Scheduled cleanup stopped (context Done)")
				return
			}
		}
	}()

	log.Printf("Scheduled cleanup started with interval %v", config.CleanupInterval)
	return stopCh
}
