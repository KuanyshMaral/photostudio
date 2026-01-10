CREATE TABLE IF NOT EXISTS notifications (
                                             id BIGSERIAL PRIMARY KEY,
                                             user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    title VARCHAR(255) NOT NULL,
    message TEXT,
    is_read BOOLEAN DEFAULT FALSE,
    data JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
    );

CREATE INDEX IF NOT EXISTS idx_notifications_user_unread
    ON notifications(user_id, is_read)
    WHERE is_read = FALSE;

CREATE INDEX IF NOT EXISTS idx_notifications_user_created
    ON notifications(user_id, created_at DESC);

COMMENT ON TABLE notifications IS 'In-app уведомления пользователей';
COMMENT ON COLUMN notifications.type IS 'Типы: booking_created, booking_confirmed, booking_cancelled, verification_approved, verification_rejected, new_review';
COMMENT ON COLUMN notifications.data IS 'JSON с дополнительными данными: booking_id, studio_id, etc.';