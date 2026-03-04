-- 000042: Add room_id to pre_bookings for room-level conflict checks
ALTER TABLE pre_bookings
    ADD COLUMN IF NOT EXISTS room_id BIGINT REFERENCES rooms(id) ON DELETE CASCADE;

-- Index for fast conflict queries per room
CREATE INDEX IF NOT EXISTS idx_pre_bookings_room_time
    ON pre_bookings (room_id, start_time, end_time);
