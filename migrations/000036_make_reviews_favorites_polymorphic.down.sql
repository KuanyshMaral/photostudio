-- Revert polymorphic schema for Reviews and Favorites

-- ==========================================
-- FAVORITES
-- ==========================================

DROP INDEX IF EXISTS idx_favorites_unique_entity;
DROP INDEX IF EXISTS idx_favorites_entity;

ALTER TABLE favorites DROP COLUMN entity_type;
ALTER TABLE favorites RENAME COLUMN entity_id TO studio_id;

CREATE UNIQUE INDEX idx_user_studio ON favorites(user_id, studio_id);

-- ==========================================
-- REVIEWS
-- ==========================================

DROP TRIGGER IF EXISTS update_studio_rating_trigger ON reviews;
DROP FUNCTION IF EXISTS trg_reviews_update_studio_rating();
DROP FUNCTION IF EXISTS update_studio_rating(BIGINT);

DROP INDEX IF EXISTS idx_reviews_target;
DROP INDEX IF EXISTS idx_reviews_author;
DROP INDEX IF EXISTS idx_reviews_unique_per_target;

ALTER TABLE reviews ADD COLUMN studio_id BIGINT REFERENCES studios(id) ON DELETE CASCADE;
ALTER TABLE reviews ADD COLUMN booking_id BIGINT REFERENCES bookings(id) ON DELETE SET NULL;

UPDATE reviews SET studio_id = target_id WHERE target_type = 'studio';
UPDATE reviews SET booking_id = context_id WHERE context_type = 'booking';

-- Data loss potential here if reviews exist for other targets! We just accept it for down migration.
DELETE FROM reviews WHERE studio_id IS NULL;
ALTER TABLE reviews ALTER COLUMN studio_id SET NOT NULL;

ALTER TABLE reviews DROP COLUMN target_type;
ALTER TABLE reviews DROP COLUMN target_id;
ALTER TABLE reviews DROP COLUMN context_type;
ALTER TABLE reviews DROP COLUMN context_id;
ALTER TABLE reviews DROP COLUMN criteria;

ALTER TABLE reviews RENAME COLUMN author_id TO user_id;

CREATE UNIQUE INDEX idx_one_review_per_user_studio ON reviews(studio_id, user_id) WHERE is_hidden = false;
CREATE INDEX idx_reviews_studio_created_at ON reviews(studio_id, created_at DESC);

-- Restore original trigger
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

CREATE TRIGGER update_studio_rating_trigger
    AFTER INSERT OR UPDATE OR DELETE ON reviews
    FOR EACH ROW
EXECUTE FUNCTION trg_reviews_update_studio_rating();
