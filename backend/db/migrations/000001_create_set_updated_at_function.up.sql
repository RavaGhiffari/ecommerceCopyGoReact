-- A shared trigger function for every table that has an updated_at column.
--
-- Why in the database instead of Go: if any query (a sqlc query, a manual
-- fix in psql, a future admin script) forgets to set updated_at, the value
-- would silently go stale. A BEFORE UPDATE trigger makes it impossible to
-- forget. Tables opt in from Phase 1 onwards with:
--
--   CREATE TRIGGER users_set_updated_at
--   BEFORE UPDATE ON users
--   FOR EACH ROW EXECUTE FUNCTION set_updated_at();
--
-- No extension is needed for UUIDs: gen_random_uuid() is built into
-- PostgreSQL 13+.

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;
