-- Restore original column names (revert rename)
ALTER TABLE studios RENAME COLUMN photos_deprecated TO photos;
ALTER TABLE rooms RENAME COLUMN photos_deprecated TO photos;
ALTER TABLE reviews RENAME COLUMN photos_deprecated TO photos;
ALTER TABLE messages RENAME COLUMN upload_id_deprecated TO upload_id;
