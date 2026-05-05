CREATE TABLE IF NOT EXISTS intent_events (
    id BIGSERIAL PRIMARY KEY,
    visit_id TEXT NOT NULL DEFAULT '',
    event_name TEXT NOT NULL,
    entity_type TEXT NOT NULL DEFAULT '',
    entity_id TEXT NOT NULL DEFAULT '',
    path TEXT NOT NULL DEFAULT '',
    referrer TEXT NOT NULL DEFAULT '',
    user_agent TEXT NOT NULL DEFAULT '',
    ip_hash TEXT NOT NULL DEFAULT '',
    is_bot BOOLEAN NOT NULL DEFAULT false,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_intent_events_created ON intent_events (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_intent_events_event_created ON intent_events (event_name, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_intent_events_entity ON intent_events (entity_type, entity_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_intent_events_ip_created ON intent_events (ip_hash, created_at DESC);
