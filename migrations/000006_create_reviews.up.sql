CREATE TABLE IF NOT EXISTS reviews (
                                       id              BIGSERIAL PRIMARY KEY,
                                       studio_id       BIGINT NOT NULL REFERENCES studios(id) ON DELETE CASCADE,
                                       user_id         BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                                       booking_id      BIGINT REFERENCES bookings(id) ON DELETE SET NULL,

                                       rating          INTEGER NOT NULL CHECK (rating BETWEEN 1 AND 5),
                                       comment         TEXT,
                                       photos          TEXT[],

                                       owner_response  TEXT,
                                       responded_at    TIMESTAMPTZ,

                                       is_verified     BOOLEAN DEFAULT false,
                                       is_hidden       BOOLEAN DEFAULT false,

                                       created_at      TIMESTAMPTZ DEFAULT NOW(),
                                       updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_one_review_per_user_studio
    ON reviews(studio_id, user_id)
    WHERE is_hidden = false;

CREATE INDEX IF NOT EXISTS idx_reviews_studio_created_at
    ON reviews(studio_id, created_at DESC);

CREATE OR REPLACE FUNCTION set_updated_at_reviews()
    RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_reviews_set_updated_at ON reviews;
CREATE TRIGGER trg_reviews_set_updated_at
    BEFORE UPDATE ON reviews
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at_reviews();

CREATE OR REPLACE FUNCTION update_studio_rating(p_studio_id BIGINT)
    RETURNS VOID AS $$
BEGIN
    UPDATE studios
    SET
        rating = (
            SELECT COALESCE(AVG(r.rating), 0)
            FROM reviews r
            WHERE r.studio_id = p_studio_id AND r.is_hidden = false
        ),
        total_reviews = (
            SELECT COUNT(*)
            FROM reviews r
            WHERE r.studio_id = p_studio_id AND r.is_hidden = false
        )
    WHERE id = p_studio_id;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION trg_reviews_update_studio_rating()
    RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        PERFORM update_studio_rating(NEW.studio_id);
        RETURN NEW;
    ELSIF TG_OP = 'UPDATE' THEN
        IF NEW.studio_id <> OLD.studio_id THEN
            PERFORM update_studio_rating(OLD.studio_id);
        END IF;
        PERFORM update_studio_rating(NEW.studio_id);
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        PERFORM update_studio_rating(OLD.studio_id);
        RETURN OLD;
    END IF;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS update_studio_rating_trigger ON reviews;
CREATE TRIGGER update_studio_rating_trigger
    AFTER INSERT OR UPDATE OR DELETE ON reviews
    FOR EACH ROW
EXECUTE FUNCTION trg_reviews_update_studio_rating();
