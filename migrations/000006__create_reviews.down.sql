DROP TRIGGER IF EXISTS update_studio_rating_trigger ON reviews;
DROP FUNCTION IF EXISTS trg_reviews_update_studio_rating();
DROP FUNCTION IF EXISTS update_studio_rating(BIGINT);

DROP TRIGGER IF EXISTS trg_reviews_set_updated_at ON reviews;
DROP FUNCTION IF EXISTS set_updated_at_reviews();

DROP TABLE IF EXISTS reviews;
