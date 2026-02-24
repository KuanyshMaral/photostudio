-- ============================================================
-- Migration 000033: Chat System Overhaul
-- Drops legacy chat tables and creates Room-based chat schema
-- ============================================================

-- Drop legacy chat tables (from migration 000009)
DROP TABLE IF EXISTS messages CASCADE;
DROP TABLE IF EXISTS conversations CASCADE;
DROP TABLE IF EXISTS blocked_users CASCADE;
DROP TABLE IF EXISTS block_relations CASCADE;

-- block_relations: User A blocks User B
CREATE TABLE IF NOT EXISTS block_relations (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    blocker_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    blocked_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (blocker_id, blocked_id)
);

CREATE INDEX idx_block_relations_blocker ON block_relations(blocker_id);
CREATE INDEX idx_block_relations_blocked ON block_relations(blocked_id);

-- chat_rooms: direct (1-on-1) and group rooms
CREATE TABLE IF NOT EXISTS chat_rooms (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type       VARCHAR(20) NOT NULL DEFAULT 'direct' CHECK (type IN ('direct', 'group')),
    name       VARCHAR(255),                                    -- only for group rooms
    creator_id BIGINT REFERENCES users(id) ON DELETE SET NULL, -- group creator/admin
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- chat_room_members: participants of a room
CREATE TABLE IF NOT EXISTS chat_room_members (
    room_id      UUID   NOT NULL REFERENCES chat_rooms(id) ON DELETE CASCADE,
    user_id      BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role         VARCHAR(20) NOT NULL DEFAULT 'member' CHECK (role IN ('admin', 'member')),
    last_read_at TIMESTAMP,
    joined_at    TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (room_id, user_id)
);

CREATE INDEX idx_chat_room_members_user ON chat_room_members(user_id);

-- messages: chat messages (replaces old messages table)
CREATE TABLE IF NOT EXISTS messages (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id    UUID   NOT NULL REFERENCES chat_rooms(id) ON DELETE CASCADE,
    sender_id  BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content    TEXT   NOT NULL DEFAULT '',
    upload_id  UUID REFERENCES uploads(id) ON DELETE SET NULL,
    is_read    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_messages_room_id ON messages(room_id, created_at DESC);
CREATE INDEX idx_messages_sender  ON messages(sender_id);
CREATE INDEX idx_messages_upload  ON messages(upload_id) WHERE upload_id IS NOT NULL;
