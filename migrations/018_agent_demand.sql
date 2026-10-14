-- Query-free discovery receipts for the provider-funded intent exchange.
-- Search text, query fingerprints, IP hashes, user agents, and alleged agent
-- identities are deliberately absent. Provider reporting uses only controlled
-- demand topics and suppresses segmented buckets below an application threshold.

CREATE TABLE IF NOT EXISTS search_receipts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    public_id TEXT NOT NULL UNIQUE,
    surface TEXT NOT NULL CHECK (surface IN ('web', 'rest', 'mcp', 'unknown')),
    explicit_category TEXT NOT NULL DEFAULT '',
    demand_topics TEXT[] NOT NULL DEFAULT '{}',
    has_api BOOLEAN NOT NULL DEFAULT false,
    has_mcp BOOLEAN NOT NULL DEFAULT false,
    has_openapi BOOLEAN NOT NULL DEFAULT false,
    has_llms_txt BOOLEAN NOT NULL DEFAULT false,
    result_count INTEGER NOT NULL DEFAULT 0 CHECK (result_count >= 0),
    page_number INTEGER NOT NULL DEFAULT 1 CHECK (page_number > 0),
    page_size INTEGER NOT NULL DEFAULT 0 CHECK (page_size >= 0),
    is_synthetic BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Migrations are idempotently replayed on boot, so upgrade databases where
-- search_receipts was created before synthetic smoke-test isolation existed.
ALTER TABLE search_receipts
    ADD COLUMN IF NOT EXISTS is_synthetic BOOLEAN NOT NULL DEFAULT false;

CREATE INDEX IF NOT EXISTS idx_search_receipts_created
    ON search_receipts(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_search_receipts_category_created
    ON search_receipts(explicit_category, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_search_receipts_topics
    ON search_receipts USING GIN(demand_topics);
CREATE INDEX IF NOT EXISTS idx_search_receipts_real_created
    ON search_receipts(created_at DESC) WHERE NOT is_synthetic;

-- A returned result is not called an impression: the server can prove only
-- that it put this record in a response, not that an agent or person read it.
CREATE TABLE IF NOT EXISTS organic_results_returned (
    id BIGSERIAL PRIMARY KEY,
    search_receipt_id UUID NOT NULL REFERENCES search_receipts(id) ON DELETE CASCADE,
    site_id UUID REFERENCES sites(id) ON DELETE SET NULL,
    site_domain_snapshot TEXT NOT NULL,
    organic_position INTEGER NOT NULL CHECK (organic_position > 0),
    score_snapshot INTEGER NOT NULL DEFAULT 0,
    returned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(search_receipt_id, site_domain_snapshot)
);

CREATE INDEX IF NOT EXISTS idx_organic_results_domain_returned
    ON organic_results_returned(site_domain_snapshot, returned_at DESC);
CREATE INDEX IF NOT EXISTS idx_organic_results_receipt_position
    ON organic_results_returned(search_receipt_id, organic_position);

-- A selection is recorded only after a successful detail request and only if
-- that domain appeared in the referenced organic result set.
CREATE TABLE IF NOT EXISTS result_selections (
    id BIGSERIAL PRIMARY KEY,
    search_receipt_id UUID NOT NULL REFERENCES search_receipts(id) ON DELETE CASCADE,
    site_id UUID REFERENCES sites(id) ON DELETE SET NULL,
    site_domain_snapshot TEXT NOT NULL,
    surface TEXT NOT NULL CHECK (surface IN ('web', 'rest', 'mcp', 'unknown')),
    selected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(search_receipt_id, site_domain_snapshot)
);

CREATE INDEX IF NOT EXISTS idx_result_selections_domain_selected
    ON result_selections(site_domain_snapshot, selected_at DESC);
