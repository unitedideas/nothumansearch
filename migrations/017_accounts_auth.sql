-- 017_accounts_auth.sql — human accounts, magic-link login, and sessions.
-- Bots authenticate with API keys (api_keys, migration 014). Humans authenticate
-- with a magic link emailed to them, which mints a session cookie. Both an active
-- account and an active api_key flow from the same $4.99/mo subscription.

CREATE TABLE IF NOT EXISTS accounts (
    id                     BIGSERIAL PRIMARY KEY,
    email                  TEXT NOT NULL UNIQUE,
    plan                   TEXT NOT NULL DEFAULT '',
    status                 TEXT NOT NULL DEFAULT 'inactive', -- active | inactive
    monthly_limit          INTEGER NOT NULL DEFAULT 0,
    stripe_customer_id     TEXT NOT NULL DEFAULT '',
    stripe_subscription_id TEXT NOT NULL DEFAULT '',
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS magic_links (
    id         BIGSERIAL PRIMARY KEY,
    email      TEXT NOT NULL,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_magic_links_token ON magic_links (token_hash);
CREATE INDEX IF NOT EXISTS idx_magic_links_expires ON magic_links (expires_at);

CREATE TABLE IF NOT EXISTS sessions (
    id         BIGSERIAL PRIMARY KEY,
    token_hash TEXT NOT NULL UNIQUE,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions (token_hash);
CREATE INDEX IF NOT EXISTS idx_sessions_account ON sessions (account_id);
