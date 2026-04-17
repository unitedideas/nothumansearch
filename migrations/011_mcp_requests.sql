CREATE TABLE IF NOT EXISTS mcp_requests (
    id         BIGSERIAL PRIMARY KEY,
    method     TEXT NOT NULL,
    tool_name  TEXT,
    arguments  JSONB,
    result_count INT,
    user_agent TEXT,
    ip_hash    TEXT,
    duration_ms INT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_mcp_requests_created ON mcp_requests (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_mcp_requests_tool ON mcp_requests (tool_name);
