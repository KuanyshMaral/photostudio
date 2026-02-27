-- 000039_deprecate_photo_arrays.up.sql
-- Phase: Replace raw TEXT[] photo arrays and inline upload_id with the new attachments table.
-- 
-- Strategy: We RENAME old columns (not DROP) so data isn't lost during migration.
-- Actual data backfill (populating attachments table from old columns) is a separate
-- one-time admin script that can be run after deploy.

-- ── studios.photos ────────────────────────────────────────────────────────────
-- Keep old column under _deprecated name for data migration script.
ALTER TABLE studios
    RENAME COLUMN photos TO photos_deprecated;

-- Attach a notice so it's clear not to use this in new code.
COMMENT ON COLUMN studios.photos_deprecated IS
    'DEPRECATED: Use SELECT from attachments WHERE target_type=''studio_gallery'' AND target_id=studios.id. This column is kept for data backfill only.';

-- ── rooms.photos ──────────────────────────────────────────────────────────────
ALTER TABLE rooms
    RENAME COLUMN photos TO photos_deprecated;

COMMENT ON COLUMN rooms.photos_deprecated IS
    'DEPRECATED: Use SELECT from attachments WHERE target_type=''room_gallery'' AND target_id=rooms.id. This column is kept for data backfill only.';

-- ── reviews.photos ────────────────────────────────────────────────────────────
ALTER TABLE reviews
    RENAME COLUMN photos TO photos_deprecated;

COMMENT ON COLUMN reviews.photos_deprecated IS
    'DEPRECATED: Use SELECT from attachments WHERE target_type=''review_photos'' AND target_id=reviews.id. This column is kept for data backfill only.';

-- ── messages.upload_id (UUID FK) → replaced by attachments ───────────────────
-- messages had a 1:1 upload_id VARCHAR FK. Now use attachments with target_type=chat_message.
ALTER TABLE messages
    RENAME COLUMN upload_id TO upload_id_deprecated;

COMMENT ON COLUMN messages.upload_id_deprecated IS
    'DEPRECATED: Use SELECT from attachments WHERE target_type=''chat_message'' AND target_id=messages.id. This column is kept for data backfill only.';
