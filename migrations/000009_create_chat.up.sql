CREATE TABLE IF NOT EXISTS conversations (

    id INTEGER PRIMARY KEY AUTOINCREMENT,

    -- participant_a всегда < participant_b для консистентности
    participant_a INTEGER NOT NULL,
    participant_b INTEGER NOT NULL,
    
    -- Контекст диалога (опционально)
    -- studio_id: если диалог про конкретную студию
    -- booking_id: если диалог про конкретную бронь
    studio_id INTEGER,
    booking_id INTEGER,
    
    -- Время последнего сообщения (для сортировки списка чатов)
    last_message_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    -- Дата создания
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    -- Внешние ключи
    FOREIGN KEY (participant_a) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (participant_b) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (studio_id) REFERENCES studios(id) ON DELETE SET NULL,
    FOREIGN KEY (booking_id) REFERENCES bookings(id) ON DELETE SET NULL,

    -- Это упрощает поиск существующего диалога
    CHECK (participant_a < participant_b)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_conversation_unique 
ON conversations(
    participant_a, 
    participant_b, 
    COALESCE(studio_id, 0), 
    COALESCE(booking_id, 0)
);

-- Индексы для быстрого поиска "мои диалоги"
CREATE INDEX IF NOT EXISTS idx_conv_participant_a ON conversations(participant_a);
CREATE INDEX IF NOT EXISTS idx_conv_participant_b ON conversations(participant_b);

CREATE TABLE IF NOT EXISTS messages (

    id INTEGER PRIMARY KEY AUTOINCREMENT,
    
    -- К какому диалогу относится
    conversation_id INTEGER NOT NULL,
    
    -- Кто отправил
    sender_id INTEGER NOT NULL,
    
    -- Содержимое сообщения
    content TEXT NOT NULL,
    
    -- Тип сообщения:
    message_type TEXT DEFAULT 'text' 
        CHECK (message_type IN ('text', 'image', 'file', 'system')),
    
    -- URL вложения (для image/file типов)
    attachment_url TEXT,
    
    -- Статус прочтения
    is_read INTEGER DEFAULT 0,  -- 0 = false, 1 = true (SQLite)
    read_at DATETIME,
    
    -- Дата создания
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    -- Внешние ключи
    FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE,
    FOREIGN KEY (sender_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Индекс для загрузки истории сообщений (сортировка по времени)
CREATE INDEX IF NOT EXISTS idx_messages_conversation 
ON messages(conversation_id, created_at DESC);

-- Индекс для подсчёта непрочитанных
CREATE INDEX IF NOT EXISTS idx_messages_unread 
ON messages(conversation_id, is_read) 
WHERE is_read = 0;

CREATE TABLE IF NOT EXISTS blocked_users (
    -- Первичный ключ
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    
    -- Кто заблокировал
    blocker_id INTEGER NOT NULL,
    
    -- Кого заблокировали
    blocked_id INTEGER NOT NULL,
    
    -- Причина блокировки (опционально)
    reason TEXT,
    
    -- Дата блокировки
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    -- Внешние ключи
    FOREIGN KEY (blocker_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (blocked_id) REFERENCES users(id) ON DELETE CASCADE,
    
    -- Уникальность: нельзя заблокировать дважды
    UNIQUE(blocker_id, blocked_id)
);

-- Индекс для проверки "заблокирован ли пользователь"
CREATE INDEX IF NOT EXISTS idx_blocked_users_check 
ON blocked_users(blocker_id, blocked_id);
