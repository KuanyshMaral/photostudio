CREATE EXTENSION IF NOT EXISTS btree_gist;

ALTER TABLE pre_bookings
    ADD CONSTRAINT pre_bookings_no_overlap_active
    EXCLUDE USING gist (
        studio_id WITH =,
        tstzrange(start_time, end_time, '[)') WITH &&
    )
    WHERE (status IN ('pending', 'confirmed_unpaid', 'paid_confirmed'));
