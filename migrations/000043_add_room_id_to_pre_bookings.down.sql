DROP INDEX IF EXISTS idx_pre_bookings_room_time;
ALTER TABLE pre_bookings DROP COLUMN IF EXISTS room_id;
