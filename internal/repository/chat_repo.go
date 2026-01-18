package repository

import (
	"context"
	"errors"

	"photostudio/internal/domain"

	"gorm.io/gorm"
)

type ChatRepository struct {
	db *gorm.DB
}

func NewChatRepository(db *gorm.DB) *ChatRepository {
	return &ChatRepository{db: db}
}

// CreateConversation создаёт новый диалог
//
// Важно: перед вызовом убедиться что participant_a < participant_b
// Это гарантирует уникальность при поиске
func (r *ChatRepository) CreateConversation(ctx context.Context, conv *domain.Conversation) error {
	return r.db.WithContext(ctx).Create(conv).Error
}

// GetConversationByID возвращает диалог по ID
func (r *ChatRepository) GetConversationByID(ctx context.Context, id int64) (*domain.Conversation, error) {
	var conv domain.Conversation
	err := r.db.WithContext(ctx).First(&conv, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("conversation not found")
		}
		return nil, err
	}
	return &conv, nil
}

func (r *ChatRepository) GetConversationByParticipants(
	ctx context.Context,
	userA, userB int64,
	studioID, bookingID *int64,
) (*domain.Conversation, error) {
	// Гарантируем порядок: participant_a всегда меньше
	if userA > userB {
		userA, userB = userB, userA
	}

	query := r.db.WithContext(ctx).
		Where("participant_a = ? AND participant_b = ?", userA, userB)

	// Фильтр по studio_id
	if studioID != nil {
		query = query.Where("studio_id = ?", *studioID)
	} else {
		query = query.Where("studio_id IS NULL")
	}

	// Фильтр по booking_id
	if bookingID != nil {
		query = query.Where("booking_id = ?", *bookingID)
	} else {
		query = query.Where("booking_id IS NULL")
	}

	var conv domain.Conversation
	err := query.First(&conv).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Диалог не найден — это OK
		}
		return nil, err
	}

	return &conv, nil
}

func (r *ChatRepository) GetUserConversations(
	ctx context.Context,
	userID int64,
	limit, offset int,
) ([]domain.Conversation, error) {
	var convs []domain.Conversation

	err := r.db.WithContext(ctx).
		// Пользователь может быть participant_a или participant_b
		Where("participant_a = ? OR participant_b = ?", userID, userID).
		// Сортировка: новые сообщения сверху
		Order("last_message_at DESC").
		// Пагинация
		Limit(limit).
		Offset(offset).
		Find(&convs).Error

	return convs, err
}

// UpdateLastMessageAt обновляет время последнего сообщения в диалоге
//
// Вызывается после отправки каждого сообщения
func (r *ChatRepository) UpdateLastMessageAt(ctx context.Context, conversationID int64) error {
	return r.db.WithContext(ctx).
		Model(&domain.Conversation{}).
		Where("id = ?", conversationID).
		Update("last_message_at", gorm.Expr("CURRENT_TIMESTAMP")).Error
}

// CreateMessage создаёт новое сообщение
func (r *ChatRepository) CreateMessage(ctx context.Context, msg *domain.Message) error {
	return r.db.WithContext(ctx).Create(msg).Error
}

// GetMessages возвращает сообщения диалога
//
// Параметры:
// - conversationID: ID диалога
// - limit: максимум сообщений
// - beforeID: для пагинации — получить сообщения старше указанного ID
//
// Возвращает сообщения в хронологическом порядке (старые первые)
func (r *ChatRepository) GetMessages(
	ctx context.Context,
	conversationID int64,
	limit int,
	beforeID *int64,
) ([]domain.Message, error) {
	query := r.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID)

	// Пагинация: сообщения старше указанного ID
	if beforeID != nil && *beforeID > 0 {
		query = query.Where("id < ?", *beforeID)
	}

	var messages []domain.Message

	// Получаем в обратном порядке (для LIMIT)
	err := query.
		Order("created_at DESC").
		Limit(limit).
		Find(&messages).Error

	if err != nil {
		return nil, err
	}

	// Разворачиваем в хронологический порядок
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// MarkMessagesAsRead помечает все непрочитанные сообщения как прочитанные
//
// Параметры:
// - conversationID: ID диалога
// - readerID: ID пользователя который читает
//
// Помечаются только сообщения ОТ другого пользователя (не свои)
//
// Возвращает количество помеченных сообщений
func (r *ChatRepository) MarkMessagesAsRead(
	ctx context.Context,
	conversationID, readerID int64,
) (int64, error) {
	result := r.db.WithContext(ctx).
		Model(&domain.Message{}).
		Where("conversation_id = ?", conversationID).
		Where("sender_id != ?", readerID). // Не свои сообщения
		Where("is_read = ?", false).       // Только непрочитанные
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": gorm.Expr("CURRENT_TIMESTAMP"),
		})

	return result.RowsAffected, result.Error
}

// CountUnread считает количество непрочитанных сообщений для пользователя
//
// Используется для отображения badge "3" на иконке чата
func (r *ChatRepository) CountUnread(
	ctx context.Context,
	conversationID, userID int64,
) (int64, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Model(&domain.Message{}).
		Where("conversation_id = ?", conversationID).
		Where("sender_id != ?", userID). // Сообщения от другого пользователя
		Where("is_read = ?", false).     // Непрочитанные
		Count(&count).Error

	return count, err
}

// CountTotalUnread считает общее количество непрочитанных сообщений
// во всех диалогах пользователя
func (r *ChatRepository) CountTotalUnread(ctx context.Context, userID int64) (int64, error) {
	var count int64

	// Подзапрос: все conversation_id где пользователь участник
	subQuery := r.db.Model(&domain.Conversation{}).
		Select("id").
		Where("participant_a = ? OR participant_b = ?", userID, userID)

	err := r.db.WithContext(ctx).
		Model(&domain.Message{}).
		Where("conversation_id IN (?)", subQuery).
		Where("sender_id != ?", userID).
		Where("is_read = ?", false).
		Count(&count).Error

	return count, err
}

// BlockUser блокирует пользователя
func (r *ChatRepository) BlockUser(
	ctx context.Context,
	blockerID, blockedID int64,
	reason string,
) error {
	blocked := &domain.BlockedUser{
		BlockerID: blockerID,
		BlockedID: blockedID,
		Reason:    reason,
	}
	return r.db.WithContext(ctx).Create(blocked).Error
}

// UnblockUser снимает блокировку
func (r *ChatRepository) UnblockUser(
	ctx context.Context,
	blockerID, blockedID int64,
) error {
	return r.db.WithContext(ctx).
		Where("blocker_id = ? AND blocked_id = ?", blockerID, blockedID).
		Delete(&domain.BlockedUser{}).Error
}

// IsBlocked проверяет есть ли блокировка между пользователями
//
// Проверяется в обе стороны:
// - userA заблокировал userB
// - userB заблокировал userA
//
// Если любая блокировка существует — возвращает true
func (r *ChatRepository) IsBlocked(ctx context.Context, userA, userB int64) (bool, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Model(&domain.BlockedUser{}).
		Where(
			"(blocker_id = ? AND blocked_id = ?) OR (blocker_id = ? AND blocked_id = ?)",
			userA, userB, userB, userA,
		).
		Count(&count).Error

	return count > 0, err
}

// GetBlockedByUser возвращает список заблокированных пользователей
func (r *ChatRepository) GetBlockedByUser(ctx context.Context, userID int64) ([]domain.BlockedUser, error) {
	var blocked []domain.BlockedUser

	err := r.db.WithContext(ctx).
		Where("blocker_id = ?", userID).
		Find(&blocked).Error

	return blocked, err
}
