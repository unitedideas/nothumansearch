-- Not Human Search — Initial Schema
-- Sites indexed by agentic readiness

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Sites: the core entity
CREATE TABLE IF NOT EXISTS sites (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    domain TEXT UNIQUE NOT NULL,
    url TEXT NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',

    -- Agentic readiness signals (boolean flags)
    has_llms_txt BOOLEAN NOT NULL DEFAULT false,
    has_ai_plugin BOOLEAN NOT NULL DEFAULT false,
    has_openapi BOOLEAN NOT NULL DEFAULT false,
    has_robots_ai BOOLEAN NOT NULL DEFAULT false,      -- robots.txt allows AI bots
    has_structured_api BOOLEAN NOT NULL DEFAULT false,  -- any discoverable REST/GraphQL API
    has_mcp_server BOOLEAN NOT NULL DEFAULT false,      -- MCP server endpoint
    has_schema_org BOOLEAN NOT NULL DEFAULT false,      -- structured data markup

    -- Extracted metadata
    llms_txt_content TEXT,
    ai_plugin_json JSONB,
    openapi_summary TEXT,

    -- Scoring
    agentic_score INT NOT NULL DEFAULT 0,  -- 0-100 composite score

    -- Categorization
    category TEXT NOT NULL DEFAULT 'other',
    tags TEXT[] NOT NULL DEFAULT '{}',

    -- Ownership
    is_verified BOOLEAN NOT NULL DEFAULT false,
    owner_email TEXT,
    is_featured BOOLEAN NOT NULL DEFAULT false,

    -- Crawl state
    last_crawled_at TIMESTAMPTZ,
    crawl_status TEXT NOT NULL DEFAULT 'pending',  -- pending, success, error, unreachable
    crawl_error TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sites_agentic_score ON sites(agentic_score DESC);
CREATE INDEX IF NOT EXISTS idx_sites_category ON sites(category);
CREATE INDEX IF NOT EXISTS idx_sites_domain ON sites(domain);
CREATE INDEX IF NOT EXISTS idx_sites_crawl_status ON sites(crawl_status);
CREATE INDEX IF NOT EXISTS idx_sites_tags ON sites USING GIN(tags);

-- Search queries log (for analytics)
CREATE TABLE IF NOT EXISTS search_queries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    query TEXT NOT NULL,
    results_count INT NOT NULL DEFAULT 0,
    user_agent TEXT,
    ip_hash TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_search_queries_created ON search_queries(created_at DESC);

-- Site submissions (users/agents submit sites to be crawled)
CREATE TABLE IF NOT EXISTS submissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    url TEXT NOT NULL,
    submitted_by TEXT,  -- email or agent identifier
    status TEXT NOT NULL DEFAULT 'pending',  -- pending, approved, rejected
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
