-- Phase 6d: audit log of admin actions (PLAN.md §6.0 Q11).
--
-- Append-only: rows are only inserted, in the SAME transaction as the admin
-- action they describe, so an action and its audit row commit or roll back
-- together. No update or delete query is generated for this table.
CREATE TABLE audit_logs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- RESTRICT: an audit trail must not lose its actor.
    actor_id    UUID         NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    action      VARCHAR(64)  NOT NULL,  -- e.g. user.suspend, review.hide
    target_type VARCHAR(32)  NOT NULL,  -- user, store, category, review
    target_id   UUID         NOT NULL,
    details     JSONB        NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- The audit log page, newest first.
CREATE INDEX audit_logs_created_idx ON audit_logs (created_at DESC);
-- "What happened to this user / store / review".
CREATE INDEX audit_logs_target_idx ON audit_logs (target_type, target_id, created_at DESC);
