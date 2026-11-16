-- Phase 6d: reviews.updated_at means "the author edited the review".
--
-- The public list shows "diubah" (edited) when updated_at > created_at. With
-- the 000012 trigger EVERY update bumped updated_at, so an admin hiding and
-- unhiding a review would mark it as edited by its author. "UPDATE OF rating,
-- comment" fires the trigger only when one of those columns is in the SET
-- list, so a status change leaves updated_at alone.
DROP TRIGGER reviews_set_updated_at ON reviews;

CREATE TRIGGER reviews_set_updated_at
    BEFORE UPDATE OF rating, comment ON reviews
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
