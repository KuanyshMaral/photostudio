package domain

import "time"

// MessageType определяет тип сообщения
type MessageType string

const (
	MessageTypeText  MessageType = "text"
	MessageTypeImage MessageType = "image"
	MessageTypeFile  MessageType = "file"
	// MessageTypeSystem — системное уведомление
	MessageTypeSystem MessageType = "system"
)

// Conversation представляет диалог между двумя пользователями
// Может быть привязан к конкретной студии или бронированию
type Conversation struct {
	// ID диалога
	ID int64 `json:"id" gorm:"primaryKey"`

	// Участник A (всегда ID меньше чем B)
	// Это правило упрощает поиск существующего диалога
	ParticipantA int64 `json:"participant_a" gorm:"not null"`

	// Участник B
	ParticipantB int64 `json:"participant_b" gorm:"not null"`

	// ID студии (опционально)
	// Если заполнено — диалог о конкретной студии
	StudioID *int64 `json:"studio_id,omitempty"`

	// ID бронирования (опционально)
	// Если заполнено — диалог о конкретной брони
	BookingID *int64 `json:"booking_id,omitempty"`

	// Используется для сортировки списка диалогов
	LastMessageAt time.Time `json:"last_message_at" gorm:"default:CURRENT_TIMESTAMP"`
	CreatedAt     time.Time `json:"created_at" gorm:"default:CURRENT_TIMESTAMP"`

	// ========== Виртуальные поля (не в БД) ==========
	// Эти поля заполняются в Service, не хранятся в БД

	// Другой участник диалога (для отображения в списке)
	OtherUser *User `json:"other_user,omitempty" gorm:"-"`

	// Информация о студии (если есть)
	Studio *Studio `json:"studio,omitempty" gorm:"-"`

	// Информация о бронировании (если есть)
	Booking *Booking `json:"booking,omitempty" gorm:"-"`

	// Последнее сообщение (для preview в списке)
	LastMessage *Message `json:"last_message,omitempty" gorm:"-"`

	// Количество непрочитанных сообщений
	UnreadCount int `json:"unread_count" gorm:"-"`
}

// TableName указывает GORM имя таблицы
func (Conversation) TableName() string {
	return "conversations"
}

// Message представляет одно сообщение в диалоге
type Message struct {
	// ========== Поля из БД ==========

	// ID сообщения
	ID int64 `json:"id" gorm:"primaryKey"`

	ConversationID int64 `json:"conversation_id" gorm:"not null;index"`

	SenderID int64 `json:"sender_id" gorm:"not null"`

	// Текст сообщения
	// Для type=image содержит "[Изображение]"
	Content string `json:"content" gorm:"not null"`

	// Тип сообщения: text, image, file, system
	MessageType MessageType `json:"message_type" gorm:"default:'text'"`

	// URL вложения (для image/file)
	AttachmentURL *string `json:"attachment_url,omitempty"`

	// Прочитано ли сообщение получателем
	IsRead bool `json:"is_read" gorm:"default:false"`

	// Когда было прочитано
	ReadAt *time.Time `json:"read_at,omitempty"`

	// Дата отправки
	CreatedAt time.Time `json:"created_at" gorm:"default:CURRENT_TIMESTAMP"`

	// ========== Виртуальные поля ==========

	// Информация об отправителе
	Sender *User `json:"sender,omitempty" gorm:"-"`
}

// TableName указывает GORM имя таблицы
func (Message) TableName() string {
	return "messages"
}

// BlockedUser представляет блокировку одного пользователя другим
type BlockedUser struct {
	// ID записи
	ID int64 `json:"id" gorm:"primaryKey"`

	// Кто заблокировал
	BlockerID int64 `json:"blocker_id" gorm:"not null"`

	// Кого заблокировали
	BlockedID int64 `json:"blocked_id" gorm:"not null"`

	// Причина блокировки (опционально)
	Reason string `json:"reason,omitempty"`

	// Дата блокировки
	CreatedAt time.Time `json:"created_at" gorm:"default:CURRENT_TIMESTAMP"`
}

// TableName указывает GORM имя таблицы
func (BlockedUser) TableName() string {
	return "blocked_users"
}
