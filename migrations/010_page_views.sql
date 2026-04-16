CREATE TABLE IF NOT EXISTS page_views (
    id          BIGSERIAL PRIMARY KEY,
    path        TEXT NOT NULL,
    method      TEXT NOT NULL DEFAULT 'GET',
    status      SMALLINT NOT NULL DEFAULT 200,
    ip_hash     TEXT,
    user_agent  TEXT,
    referer     TEXT,
    duration_ms INTEGER,
    is_bot      BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_page_views_created ON page_views (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_page_views_path ON page_views (path);
