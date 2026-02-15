# Notification Module - Integration Guide

## 📚 Обзор

Полностью переработанный модуль `notification` с детальной типизацией, поддержкой preferences, device tokens и интеграцией с внешними сервисами.

## 🗂️ Структура Файлов

```
internal/domain/notification/
├── entity.go ........................ Domain entities (Notification, NotificationData, UserPreferences, DeviceToken)
├── dto.go ........................... API response types
├── repository.go .................... Repository interfaces & GORM implementations
├── service.go ....................... Business logic & specialized notification methods
├── extended_service.go .............. Email/Push integration layer
├── cleanup.go ....................... Background cleanup service
├── handler.go ....................... Notification API handlers  
├── preferences_handler.go ........... Preferences API handlers
├── device_tokens_handler.go ......... Device tokens API handlers
└── routes.go ........................ Route registration

migrations/
├── 000030_enhance_notifications_add_preferences_device_tokens.up.sql
└── 000030_enhance_notifications_add_preferences_device_tokens.down.sql
```

## 🚀 Использование в Других Domains

### 1. Инициализация в main/setup

```go
import (
    notifDomain "photostudio/internal/domain/notification"
)

// В функции инициализации сервисов
func setupNotificationServices(db *gorm.DB) {
    // Создать repositories
    notifRepo := notifDomain.NewRepository(db)
    prefRepo := notifDomain.NewPreferencesRepository(db)
    deviceTokenRepo := notifDomain.NewDeviceTokenRepository(db)
    
    // Создать service
    notifService := notifDomain.NewService(notifRepo, prefRepo, deviceTokenRepo)
    
    // Опционально: создать extended service для интеграции с email/push
    extService := notifDomain.NewExtendedService(notifService, &notifDomain.ExternalServices{
        EmailService: emailService,    // Ваш email сервис
        PushService:  pushService,     // Ваш push сервис
    })
    
    // Создать handlers
    notifHandler := notifDomain.NewHandler(notifService)
    prefsHandler := notifDomain.NewPreferencesHandler(notifService)
    devicesHandler := notifDomain.NewDeviceTokensHandler(notifService)
    
    // Регистрировать routes
    notifDomain.RegisterRoutes(router.Group("/api"), notifHandler, prefsHandler, devicesHandler)
    
    // Опционально: запустить cleanup service
    cleanupService := notifDomain.NewCleanupService(notifRepo, deviceTokenRepo)
    config := notifDomain.DefaultCleanupConfig()
    cleanupService.ScheduleCleanup(context.Background(), config)
}
```

### 2. Использование в Booking Domain

```go
import (
    bookingDomain "photostudio/internal/domain/booking"
    notifDomain "photostudio/internal/domain/notification"
)

type BookingService struct {
    bookingRepo bookingDomain.Repository
    notifService *notifDomain.Service  // Inject here
}

// При создании нового бронирования
func (s *BookingService) CreateBooking(ctx context.Context, booking *bookingDomain.Booking) error {
    // ... business logic ...
    
    // Notify owner about new booking
    if err := s.notifService.NotifyBookingCreated(
        ctx,
        booking.OwnerID,
        booking.ID,
        booking.StudioID,
        booking.RoomID,
        booking.StartTime,
    ); err != nil {
        log.Printf("Failed to send notification: %v", err)
        // Don't fail booking creation if notification fails
    }
    
    return nil
}

// При подтверждении бронирования
func (s *BookingService) ConfirmBooking(ctx context.Context, bookingID int64) error {
    booking, err := s.bookingRepo.GetByID(ctx, bookingID)
    if err != nil {
        return err
    }
    
    // ... update booking status ...
    
    // Notify client
    if err := s.notifService.NotifyBookingConfirmed(ctx, booking.ClientID, bookingID, booking.StudioID); err != nil {
        log.Printf("Failed to send confirmation notification: %v", err)
    }
    
    return nil
}

// При отмене бронирования
func (s *BookingService) CancelBooking(ctx context.Context, bookingID int64, reason string) error {
    booking, err := s.bookingRepo.GetByID(ctx, bookingID)
    if err != nil {
        return err
    }
    
    // ... update booking status ...
    
    // Notify client about cancellation
    if err := s.notifService.NotifyBookingCancelled(ctx, booking.ClientID, bookingID, booking.StudioID, reason); err != nil {
        log.Printf("Failed to send cancellation notification: %v", err)
    }
    
    return nil
}
```

### 3. Использование в Review Domain

```go
type ReviewService struct {
    reviewRepo reviewDomain.Repository
    notifService *notifDomain.Service
}

// При создании нового отзыва
func (s *ReviewService) CreateReview(ctx context.Context, review *reviewDomain.Review) error {
    // ... save review ...
    
    if err := s.notifService.NotifyNewReview(
        ctx,
        review.StudioOwnerID,
        review.StudioID,
        review.ID,
        review.Rating,
    ); err != nil {
        log.Printf("Failed to notify about new review: %v", err)
    }
    
    return nil
}
```

### 4. Использование в Chat/Message Domain

```go
type MessageService struct {
    messageRepo messageDomain.Repository
    notifService *notifDomain.Service
}

// При отправке нового сообщения
func (s *MessageService) SendMessage(ctx context.Context, msg *messageDomain.Message) error {
    // ... save message ...
    
    if err := s.notifService.NotifyNewMessage(
        ctx,
        msg.RecipientID,
        msg.SenderName,
        msg.PreviewText, // First 100 chars
        msg.ChatRoomID,
        msg.ID,
    ); err != nil {
        log.Printf("Failed to notify about new message: %v", err)
    }
    
    return nil
}
```

### 5. Использование в Verification Domain

```go
type VerificationService struct {
    verificationRepo verificationDomain.Repository
    notifService *notifDomain.Service
}

// При одобрении верификации
func (s *VerificationService) ApproveStudioVerification(ctx context.Context, studioID int64) error {
    studio, err := s.studioRepo.GetByID(ctx, studioID)
    if err != nil {
        return err
    }
    
    // ... update verification status ...
    
    if err := s.notifService.NotifyVerificationApproved(ctx, studio.OwnerID, studioID); err != nil {
        log.Printf("Failed to notify about verification approval: %v", err)
    }
    
    return nil
}

// При отклонении верификации
func (s *VerificationService) RejectStudioVerification(ctx context.Context, studioID int64, reason string) error {
    studio, err := s.studioRepo.GetByID(ctx, studioID)
    if err != nil {
        return err
    }
    
    // ... update verification status ...
    
    if err := s.notifService.NotifyVerificationRejected(ctx, studio.OwnerID, studioID, reason); err != nil {
        log.Printf("Failed to notify about verification rejection: %v", err)
    }
    
    return nil
}
```

## 🔌 API Endpoints

### Notifications
- `GET /api/notifications` - Получить уведомления с пагинацией
- `GET /api/notifications/unread-count` - Количество непрочитанных
- `PATCH /api/notifications/{id}/read` - Отметить как прочитанное
- `POST /api/notifications/read-all` - Отметить все как прочитанные
- `DELETE /api/notifications/{id}` - Удалить уведомление

### Preferences
- `GET /api/notifications/preferences` - Получить предпочтения
- `PATCH /api/notifications/preferences` - Обновить предпочтения
- `POST /api/notifications/preferences/reset` - Сбросить на defaults

### Device Tokens
- `POST /api/notifications/device-tokens` - Зарегистрировать device
- `GET /api/notifications/device-tokens` - Список активных devices
- `DELETE /api/notifications/device-tokens/{id}` - Деактивировать device

## 📊 Типы Уведомлений

```go
const (
    TypeBookingCreated       Type = "booking_created"       // Owner
    TypeBookingConfirmed     Type = "booking_confirmed"     // Client
    TypeBookingCancelled     Type = "booking_cancelled"     // Client
    TypeBookingCompleted     Type = "booking_completed"     // Both
    TypeVerificationApproved Type = "verification_approved" // Owner
    TypeVerificationRejected Type = "verification_rejected" // Owner
    TypeNewReview            Type = "new_review"            // Owner
    TypeNewMessage           Type = "new_message"           // Both
    TypeEquipmentBooked      Type = "equipment_booked"      // Owner
    TypeStudioUpdated        Type = "studio_updated"        // Followers
)
```

## 🔍 Структурированные Данные

```go
type NotificationData struct {
    BookingID              *int64   `json:"booking_id,omitempty"`
    StudioID               *int64   `json:"studio_id,omitempty"`
    RoomID                 *int64   `json:"room_id,omitempty"`
    ReviewID               *int64   `json:"review_id,omitempty"`
    EquipmentID            *int64   `json:"equipment_id,omitempty"`
    MessageID              *int64   `json:"message_id,omitempty"`
    ChatRoomID             *int64   `json:"chat_room_id,omitempty"`
    Rating                 *int     `json:"rating,omitempty"`
    SenderName             *string  `json:"sender_name,omitempty"`
    MessagePreview         *string  `json:"message_preview,omitempty"`
    Reason                 *string  `json:"reason,omitempty"`
    StartTime              *string  `json:"start_time,omitempty"`       // ISO8601
    EndTime                *string  `json:"end_time,omitempty"`         // ISO8601
    CancellationReason     *string  `json:"cancellation_reason,omitempty"`
}
```

## ⚙️ Cleanup Service

Автоматически удаляет старые уведомления и неактивные device tokens:

```go
// Конфигурация
config := notifDomain.CleanupConfig{
    NotificationRetentionDays:  90,   // Держать уведомления 90 дней
    DeviceTokenInactivityDays:  90,   // Удалять unused tokens после 90 дней
    CleanupInterval:            24 * time.Hour, // Запускать ежедневно
    EnableAutomaticCleanup:     true,
}

cleanupService := notifDomain.NewCleanupService(notifRepo, deviceTokenRepo)
stopCh := cleanupService.ScheduleCleanup(ctx, config)

// После завершения
close(stopCh)
```

## 🎯 Preferences Management

Пользователи могут управлять каналами для каждого типа уведомлений:

```go
// Пример JSON структуры preferences
{
    "email_enabled": true,
    "push_enabled": true,
    "in_app_enabled": true,
    "digest_enabled": true,
    "digest_frequency": "weekly",
    "per_type_settings": {
        "booking_created": {
            "in_app": true,
            "email": true,
            "push": true
        },
        "new_message": {
            "in_app": true,
            "email": false,
            "push": true
        },
        "new_review": {
            "in_app": true,
            "email": true,
            "push": false
        }
    }
}
```

## 🧪 Тестирование

```go
// Mock repository для тестов
type mockNotificationRepository struct {
    // Implement Repository interface
}

// Пример теста
func TestNotifyBookingCreated(t *testing.T) {
    mockRepo := &mockNotificationRepository{}
    mockPrefRepo := &mockPreferencesRepository{}
    mockDeviceRepo := &mockDeviceTokenRepository{}
    
    svc := notifDomain.NewService(mockRepo, mockPrefRepo, mockDeviceRepo)
    
    err := svc.NotifyBookingCreated(
        context.Background(),
        ownerID,
        bookingID,
        studioID,
        roomID,
        time.Now(),
    )
    
    assert.NoError(t, err)
}
```

## 🔐 Безопасность

1. **User Isolation** - Все уведомления привязаны к userID, нельзя получить чужие
2. **Rate Limiting** - Рекомендуется добавить rate limiting на endpoints
3. **Token Validation** - Все device tokens валидируются перед использованием
4. **Preferences Override** - Система уважает preferences пользователя перед отправкой

## 📝 Миграция Существующих Данных

Миграция обновит существующую таблицу notifications:

```sql
-- Добавит read_at column
ALTER TABLE notifications ADD COLUMN read_at TIMESTAMPTZ;

-- Переименует message -> body
ALTER TABLE notifications RENAME COLUMN message TO body;

-- Создаст две новые таблицы:
-- - user_notification_preferences
-- - device_tokens

-- Создаст индексы для оптимизации
```

Существующие данные не будут потеряны - все старые уведомления останутся и будут работать с новой структурой.

## 🚨 Important Notes

1. **Constructor Compatibility** - Если используется старый `NewService(repo *NotificationRepository)`, используйте вместо этого `NewServiceLegacy()`
2. **Backward Compatibility** - Старый repository методы поддерживаются через adapter pattern
3. **Error Handling** - Ошибки уведомления не должны блокировать основные операции (например, создание бронирования)
4. **Logging** - Все ошибки логируются, рекомендуется мониторить логи

## 📈 Performance Considerations

1. **Индексы** - Все часто запрашиваемые поля имеют индексы
2. **Pagination** - Всегда используйте пpagination для ListByUser
3. **Cleanup** - Старые уведомления автоматически удаляются через cleanup service
4. **JSON Storage** - NotificationData хранится в JSONB для быстрого поиска

## 🔗 Ссылки

- [Notification Entity](entity.go)
- [Service API](service.go)
- [Repository Interface](repository.go)
- [Cleanup Service](cleanup.go)
- [Extended Service](extended_service.go)
