-- Migration to implement polymorphic relations for Reviews and Favorites

-- ==========================================
-- REVIEWS
-- ==========================================

-- 1. Rename user_id to author_id for consistency
ALTER TABLE reviews RENAME COLUMN user_id TO author_id;

-- 2. Add polymorphic columns
ALTER TABLE reviews ADD COLUMN target_type VARCHAR(50);
ALTER TABLE reviews ADD COLUMN target_id BIGINT;
ALTER TABLE reviews ADD COLUMN context_type VARCHAR(50);
ALTER TABLE reviews ADD COLUMN context_id BIGINT;
ALTER TABLE reviews ADD COLUMN criteria JSONB NOT NULL DEFAULT '{}';

-- 3. Backfill data based on legacy columns
UPDATE reviews 
SET target_type = 'studio', 
    target_id = studio_id, 
    context_type = CASE WHEN booking_id IS NOT NULL THEN 'booking' ELSE NULL END, 
    context_id = booking_id;

-- 4. Enforce NOT NULL on generic target fields
ALTER TABLE reviews ALTER COLUMN target_type SET NOT NULL;
ALTER TABLE reviews ALTER COLUMN target_id SET NOT NULL;

-- 5. Drop legacy Studio and Booking columns
ALTER TABLE reviews DROP COLUMN studio_id CASCADE;
ALTER TABLE reviews DROP COLUMN booking_id CASCADE;

-- 6. Drop old indexes
DROP INDEX IF EXISTS idx_one_review_per_user_studio;
DROP INDEX IF EXISTS idx_reviews_studio_created_at;

-- 7. Add polymorphic indexes
CREATE INDEX idx_reviews_target ON reviews(target_type, target_id);
CREATE INDEX idx_reviews_author ON reviews(author_id);
-- Ensure users don't review the same target+context twice
CREATE UNIQUE INDEX idx_reviews_unique_per_target ON reviews(author_id, target_type, target_id, context_id);

-- 8. Recreate Studio Rating Update Trigger (adapted for polymorphic fields)
CREATE OR REPLACE FUNCTION update_studio_rating(p_studio_id BIGINT)
    RETURNS VOID AS $$
BEGIN
    UPDATE studios
    SET
        rating = (
            SELECT COALESCE(AVG(r.rating), 0)
            FROM reviews r
            WHERE r.target_type = 'studio' AND r.target_id = p_studio_id AND r.is_hidden = false
        ),
        total_reviews = (
            SELECT COUNT(*)
            FROM reviews r
            WHERE r.target_type = 'studio' AND r.target_id = p_studio_id AND r.is_hidden = false
        )
    WHERE id = p_studio_id;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION trg_reviews_update_studio_rating()
    RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF NEW.target_type = 'studio' THEN
            PERFORM update_studio_rating(NEW.target_id);
        END IF;
        RETURN NEW;
    ELSIF TG_OP = 'UPDATE' THEN
        IF OLD.target_type = 'studio' AND NEW.target_type = 'studio' AND NEW.target_id <> OLD.target_id THEN
            PERFORM update_studio_rating(OLD.target_id);
            PERFORM update_studio_rating(NEW.target_id);
        ELSIF OLD.target_type = 'studio' THEN
            PERFORM update_studio_rating(OLD.target_id);
        ELSIF NEW.target_type = 'studio' THEN
            PERFORM update_studio_rating(NEW.target_id);
        END IF;
        
        -- Also handle rating value/hidden changes if ID stays same
        IF OLD.target_type = 'studio' AND NEW.target_type = 'studio' AND NEW.target_id = OLD.target_id THEN
            PERFORM update_studio_rating(NEW.target_id);
        END IF;

        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        IF OLD.target_type = 'studio' THEN
            PERFORM update_studio_rating(OLD.target_id);
        END IF;
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

-- ==========================================
-- FAVORITES
-- ==========================================

-- 1. Rename studio_id to entity_id for generic use
ALTER TABLE favorites RENAME COLUMN studio_id TO entity_id;

-- 2. Add polymorphic entity_type
ALTER TABLE favorites ADD COLUMN entity_type VARCHAR(50);

-- 3. Backfill with 'studio' as legacy type
UPDATE favorites SET entity_type = 'studio';

-- 4. Enforce NOT NULL on entity_type
ALTER TABLE favorites ALTER COLUMN entity_type SET NOT NULL;

-- 5. Drop old index
DROP INDEX IF EXISTS idx_user_studio;

-- 6. Recreate indexes for polymorphic approach
CREATE UNIQUE INDEX idx_favorites_unique_entity ON favorites(user_id, entity_type, entity_id);
CREATE INDEX idx_favorites_entity ON favorites(entity_type, entity_id);
