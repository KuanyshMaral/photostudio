-- 000037_create_attachments.up.sql
-- Polymorphic 1:N file→entity relationships.
-- Upload = "dumb warehouse" (raw files).
-- Attachment = "business label" (what role a file plays for a specific entity).

CREATE TABLE attachments (
    id          BIGSERIAL PRIMARY KEY,
    upload_id   VARCHAR(36) NOT NULL REFERENCES uploads(id) ON DELETE CASCADE,

    -- Polymorphic target: which entity owns this file?
    target_id   BIGINT      NOT NULL,
    target_type VARCHAR(50) NOT NULL CHECK (target_type IN (
        'studio_gallery',   -- Photos of the studio itself
        'room_gallery',     -- Photos of a specific room
        'review_photos',    -- Photos attached to a review
        'chat_message'      -- File(s) attached to a chat message
    )),

    -- Ordering within a collection (e.g. gallery slide position)
    sort_order  INT NOT NULL DEFAULT 0,

    -- Optional per-target metadata (e.g. caption for a gallery photo)
    metadata    JSONB NOT NULL DEFAULT '{}',

    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Prevent same file being attached twice to the same target
    UNIQUE (upload_id, target_id, target_type)
);

-- Core lookup: "give me all attachments for this entity in order"
CREATE INDEX idx_attachments_target ON attachments(target_type, target_id, sort_order);
-- Reverse lookup: "where is this upload referenced?"
CREATE INDEX idx_attachments_upload_id ON attachments(upload_id);

COMMENT ON TABLE attachments IS 'Polymorphic 1:N file→entity bridge. upload provides raw file; attachment provides business label.';
COMMENT ON COLUMN attachments.target_type IS 'VARCHAR with CHECK constraint for easy future extension without ALTER TABLE.';
COMMENT ON COLUMN attachments.metadata IS 'Per-target-type structured data. Go uses typed structs; DB stores JSON.';
