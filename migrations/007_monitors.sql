-- Monitors: email subscribers who want alerts when a site's agentic
-- readiness drops. Many-to-many between emails and domains, so one email
-- can watch many domains and one domain can have many watchers.
-- Each subscription gets an unguessable token for unsubscribe links.

CREATE TABLE IF NOT EXISTS monitors (
    id                 BIGSERIAL PRIMARY KEY,
    email              TEXT NOT NULL,
    domain             TEXT NOT NULL,
    token              TEXT NOT NULL UNIQUE,
    last_score         INT,
    last_signals_hash  TEXT,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_checked_at    TIMESTAMPTZ,
    last_notified_at   TIMESTAMPTZ,
    UNIQUE (email, domain)
);

CREATE INDEX IF NOT EXISTS monitors_domain_idx ON monitors (domain);
CREATE INDEX IF NOT EXISTS monitors_last_checked_idx ON monitors (last_checked_at NULLS FIRST);
