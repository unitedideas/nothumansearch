ALTER TABLE monitors ADD COLUMN IF NOT EXISTS last_admin_action TEXT;
ALTER TABLE monitors ADD COLUMN IF NOT EXISTS last_admin_action_at TIMESTAMPTZ;
ALTER TABLE monitors ADD COLUMN IF NOT EXISTS last_admin_operator TEXT;
ALTER TABLE monitors ADD COLUMN IF NOT EXISTS last_admin_source TEXT;
ALTER TABLE monitors ADD COLUMN IF NOT EXISTS score_rerun_requested_at TIMESTAMPTZ;
ALTER TABLE monitors ADD COLUMN IF NOT EXISTS remediation_offered_at TIMESTAMPTZ;
ALTER TABLE monitors ADD COLUMN IF NOT EXISTS private_review_notes TEXT;

CREATE TABLE IF NOT EXISTS monitor_admin_actions (
    id          BIGSERIAL PRIMARY KEY,
    monitor_id  BIGINT NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    action      TEXT NOT NULL,
    operator    TEXT NOT NULL,
    source      TEXT NOT NULL,
    notes       TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS monitor_admin_actions_monitor_idx ON monitor_admin_actions (monitor_id, created_at DESC);
CREATE INDEX IF NOT EXISTS monitors_last_admin_action_idx ON monitors (last_admin_action, last_admin_action_at DESC);
