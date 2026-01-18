package chat

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"photostudio/internal/domain"
	"photostudio/internal/modules/notification"
	"photostudio/internal/repository"
)

var (
	ErrNotParticipant       = errors.New("you are not a participant of this conversation")
	ErrBlocked              = errors.New("user has blocked you or you have blocked user")
	ErrRecipientNotFound    = errors.New("recipient not found")
	ErrEmptyContent         = errors.New("message content cannot be empty")
	ErrConversationNotFound = errors.New("conversation not found")
	ErrCannotMessageSelf    = errors.New("cannot send message to yourself")
)

type Service struct {
	chatRepo     *repository.ChatRepository
	userRepo     *repository.UserRepository
	studioRepo   *repository.StudioRepository
	bookingRepo  *repository.BookingRepository
	notifService *notification.Service
}

func NewService(
	chatRepo *repository.ChatRepository,
	userRepo *repository.UserRepository,
	studioRepo *repository.StudioRepository,
	bookingRepo *repository.BookingRepository,
	notifService *notification.Service,
) *Service {
	return &Service{
		chatRepo:     chatRepo,
		userRepo:     userRepo,
		studioRepo:   studioRepo,
		bookingRepo:  bookingRepo,
		notifService: notifService,
	}
}

// ============================================================
// CONVERSATIONS
// ============================================================

func (s *Service) GetOrCreateConversation(
	ctx context.Context,
	senderID int64,
	req CreateConversationRequest,
) (*domain.Conversation, *domain.Message, error) {

	if senderID == req.RecipientID {
		return nil, nil, ErrCannotMessageSelf
	}

	recipient, err := s.userRepo.GetByID(ctx, req.RecipientID)
	if err != nil || recipient == nil {
		return nil, nil, ErrRecipientNotFound
	}

	blocked, err := s.chatRepo.IsBlocked(ctx, senderID, req.RecipientID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to check block status: %w", err)
	}
	if blocked {
		return nil, nil, ErrBlocked
	}

	participantA, participantB := senderID, req.RecipientID
	if participantA > participantB {
		participantA, participantB = participantB, participantA
	}

	existing, err := s.chatRepo.GetConversationByParticipants(ctx, participantA, participantB, req.StudioID, req.BookingID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find conversation: %w", err)
	}

	if existing != nil {
		var msg *domain.Message
		if strings.TrimSpace(req.InitialMessage) != "" {
			msg, _ = s.SendMessage(ctx, senderID, existing.ID, SendMessageRequest{Content: req.InitialMessage})
		}
		_ = s.enrichConversation(ctx, existing, senderID)
		return existing, msg, nil
	}

	conv := &domain.Conversation{
		ParticipantA: participantA,
		ParticipantB: participantB,
		StudioID:     req.StudioID,
		BookingID:    req.BookingID,
	}

	if err := s.chatRepo.CreateConversation(ctx, conv); err != nil {
		return nil, nil, fmt.Errorf("failed to create conversation: %w", err)
	}

	var msg *domain.Message
	if strings.TrimSpace(req.InitialMessage) != "" {
		msg, _ = s.SendMessage(ctx, senderID, conv.ID, SendMessageRequest{Content: req.InitialMessage})
	}

	_ = s.enrichConversation(ctx, conv, senderID)
	return conv, msg, nil
}

func (s *Service) GetUserConversations(ctx context.Context, userID int64, limit, offset int) ([]domain.Conversation, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	convs, err := s.chatRepo.GetUserConversations(ctx, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversations: %w", err)
	}

	for i := range convs {
		_ = s.enrichConversation(ctx, &convs[i], userID)
	}

	return convs, nil
}

func (s *Service) IsParticipant(ctx context.Context, userID, conversationID int64) bool {
	conv, err := s.chatRepo.GetConversationByID(ctx, conversationID)
	if err != nil || conv == nil {
		return false
	}
	return conv.ParticipantA == userID || conv.ParticipantB == userID
}

func (s *Service) enrichConversation(ctx context.Context, conv *domain.Conversation, currentUserID int64) error {
	otherUserID := conv.ParticipantA
	if otherUserID == currentUserID {
		otherUserID = conv.ParticipantB
	}

	otherUser, _ := s.userRepo.GetByID(ctx, otherUserID)
	conv.OtherUser = otherUser

	if conv.StudioID != nil {
		st, _ := s.studioRepo.GetByID(ctx, *conv.StudioID)
		conv.Studio = st
	}

	if conv.BookingID != nil {
		bk, _ := s.bookingRepo.GetByID(ctx, *conv.BookingID)
		conv.Booking = bk
	}

	msgs, _ := s.chatRepo.GetMessages(ctx, conv.ID, 1, nil)
	if len(msgs) > 0 {
		conv.LastMessage = &msgs[0]
	}

	unread, _ := s.chatRepo.CountUnread(ctx, conv.ID, currentUserID)
	conv.UnreadCount = int(unread)

	return nil
}

// ============================================================
// MESSAGES
// ============================================================

func (s *Service) SendMessage(ctx context.Context, senderID, conversationID int64, req SendMessageRequest) (*domain.Message, error) {
	if strings.TrimSpace(req.Content) == "" {
		return nil, ErrEmptyContent
	}

	conv, err := s.chatRepo.GetConversationByID(ctx, conversationID)
	if err != nil || conv == nil {
		return nil, ErrConversationNotFound
	}

	if conv.ParticipantA != senderID && conv.ParticipantB != senderID {
		return nil, ErrNotParticipant
	}

	recipientID := conv.ParticipantA
	if recipientID == senderID {
		recipientID = conv.ParticipantB
	}

	blocked, _ := s.chatRepo.IsBlocked(ctx, senderID, recipientID)
	if blocked {
		return nil, ErrBlocked
	}

	msg := &domain.Message{
		ConversationID: conversationID,
		SenderID:       senderID,
		Content:        req.Content,
		MessageType:    domain.MessageTypeText,
	}

	if err := s.chatRepo.CreateMessage(ctx, msg); err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	_ = s.chatRepo.UpdateLastMessageAt(ctx, conversationID)

	sender, _ := s.userRepo.GetByID(ctx, senderID)
	msg.Sender = sender

	return msg, nil
}

func (s *Service) GetMessages(ctx context.Context, userID, conversationID int64, limit int, beforeID *int64) ([]domain.Message, bool, error) {
	if !s.IsParticipant(ctx, userID, conversationID) {
		return nil, false, ErrNotParticipant
	}

	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	msgs, err := s.chatRepo.GetMessages(ctx, conversationID, limit+1, beforeID)
	if err != nil {
		return nil, false, err
	}

	hasMore := len(msgs) > limit
	if hasMore {
		msgs = msgs[:limit]
	}

	for i := range msgs {
		u, _ := s.userRepo.GetByID(ctx, msgs[i].SenderID)
		msgs[i].Sender = u
	}

	return msgs, hasMore, nil
}

func (s *Service) MarkAsRead(ctx context.Context, userID, conversationID int64) (int64, error) {
	if !s.IsParticipant(ctx, userID, conversationID) {
		return 0, ErrNotParticipant
	}
	return s.chatRepo.MarkMessagesAsRead(ctx, conversationID, userID)
}

// ============================================================
// BLOCKING
// ============================================================

func (s *Service) BlockUser(ctx context.Context, blockerID, blockedID int64, reason string) error {
	if blockerID == blockedID {
		return errors.New("cannot block yourself")
	}
	return s.chatRepo.BlockUser(ctx, blockerID, blockedID, reason)
}

func (s *Service) UnblockUser(ctx context.Context, blockerID, blockedID int64) error {
	return s.chatRepo.UnblockUser(ctx, blockerID, blockedID)
}

// ============================================================
// IMAGE MESSAGES
// ============================================================

// SendImageMessage создаёт сообщение с изображением
func (s *Service) SendImageMessage(
	ctx context.Context,
	senderID int64,
	conversationID int64,
	imageURL string,
) (*domain.Message, error) {
	conv, err := s.chatRepo.GetConversationByID(ctx, conversationID)
	if err != nil || conv == nil {
		return nil, ErrConversationNotFound
	}

	if conv.ParticipantA != senderID && conv.ParticipantB != senderID {
		return nil, ErrNotParticipant
	}

	recipientID := conv.ParticipantA
	if recipientID == senderID {
		recipientID = conv.ParticipantB
	}

	blocked, _ := s.chatRepo.IsBlocked(ctx, senderID, recipientID)
	if blocked {
		return nil, ErrBlocked
	}

	msg := &domain.Message{
		ConversationID: conversationID,
		SenderID:       senderID,
		Content:        "[Изображение]",
		MessageType:    domain.MessageTypeImage,
		AttachmentURL:  &imageURL,
	}

	if err := s.chatRepo.CreateMessage(ctx, msg); err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	_ = s.chatRepo.UpdateLastMessageAt(ctx, conversationID)

	sender, _ := s.userRepo.GetByID(ctx, senderID)
	msg.Sender = sender

	return msg, nil
}

// ============================================================
// NOTIFICATIONS
// ============================================================

// NotifyIfOffline создаёт уведомление если получатель offline
//
// Вызывается из Handler после попытки отправки через WebSocket
// Если WebSocket не доставил — создаём notification
func (s *Service) NotifyIfOffline(
	ctx context.Context,
	recipientID int64,
	conversation *domain.Conversation,
	message *domain.Message,
) error {
	// Получаем информацию об отправителе
	sender, err := s.userRepo.GetByID(ctx, message.SenderID)
	if err != nil {
		return err
	}

	// Формируем preview сообщения
	preview := message.Content
	if len(preview) > 50 {
		preview = preview[:50] + "..."
	}

	// Для изображений — специальный текст
	if message.MessageType == domain.MessageTypeImage {
		preview = "📷 Изображение"
	}

	// Создаём уведомление
	title := fmt.Sprintf("Сообщение от %s", sender.Name)
	body := preview

	return s.notifService.Create(
		ctx,
		recipientID,
		domain.NotifNewMessage,
		title,
		body,
		map[string]any{
			"conversation_id": conversation.ID,
			"message_id":      message.ID,
			"sender_id":       message.SenderID,
			"sender_name":     sender.Name,
		},
	)
}

// GetRecipientID возвращает ID получателя для диалога
func (s *Service) GetRecipientID(conversation *domain.Conversation, senderID int64) int64 {
	if conversation.ParticipantA == senderID {
		return conversation.ParticipantB
	}
	return conversation.ParticipantA
}
