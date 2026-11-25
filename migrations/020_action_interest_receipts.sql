-- Provider-independent action-interest receipts for Stage 1 demand proof.
-- These rows prove only that a caller attested current principal interest in
-- one controlled next step for an exact organic result. They do not contact a
-- provider, create an action ticket, affect rank, or create commercial proof.
-- No query, contact detail, free-form text, IP/user-agent value, or alleged
-- agent/principal identity is accepted by this schema.

-- A composite foreign key below makes the non-synthetic requirement a database
-- invariant rather than an application convention. The ordinary UUID primary
-- key remains unchanged and is still the canonical search-receipt identity.
CREATE UNIQUE INDEX IF NOT EXISTS idx_search_receipts_id_synthetic
    ON search_receipts(id, is_synthetic);

CREATE TABLE IF NOT EXISTS action_interest_receipts (
    id                         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    public_id                  TEXT NOT NULL UNIQUE
                               CHECK (public_id ~ '^nhs_air_[A-Za-z0-9_-]{16}$'),
    search_receipt_id          UUID NOT NULL,
    source_is_synthetic        BOOLEAN NOT NULL DEFAULT false
                               CHECK (NOT source_is_synthetic),
    site_domain_snapshot       TEXT NOT NULL
                               CHECK (length(site_domain_snapshot) BETWEEN 3 AND 253),
    action_type                TEXT NOT NULL CHECK (action_type IN (
                                   'quote', 'trial', 'demo', 'booking',
                                   'application', 'signup', 'purchase'
                               )),
    surface                    TEXT NOT NULL
                               CHECK (surface IN ('web', 'rest', 'mcp', 'unknown')),
    caller_attests_principal_interest BOOLEAN NOT NULL
                               CHECK (caller_attests_principal_interest),
    confirmation_version       TEXT NOT NULL
                               CHECK (confirmation_version = 'nhs-action-interest-v1'),
    created_at                 TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at                 TIMESTAMPTZ NOT NULL,
    UNIQUE (search_receipt_id, site_domain_snapshot),
    CONSTRAINT action_interest_receipts_non_synthetic_fk
    FOREIGN KEY (search_receipt_id, source_is_synthetic)
        REFERENCES search_receipts(id, is_synthetic) ON DELETE CASCADE,
    CONSTRAINT action_interest_receipts_returned_result_fk
    FOREIGN KEY (search_receipt_id, site_domain_snapshot)
        REFERENCES organic_results_returned(search_receipt_id, site_domain_snapshot)
        ON DELETE CASCADE,
    CONSTRAINT action_interest_receipts_positive_ttl
        CHECK (expires_at > created_at),
    CONSTRAINT action_interest_receipts_max_ttl
        CHECK (expires_at <= created_at + INTERVAL '30 days')
);

CREATE INDEX IF NOT EXISTS idx_action_interest_receipts_domain_created
    ON action_interest_receipts(site_domain_snapshot, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_action_interest_receipts_action_created
    ON action_interest_receipts(action_type, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_action_interest_receipts_expires
    ON action_interest_receipts(expires_at);

-- An action-interest receipt is immutable while present. Deletion remains
-- available so hourly expiry cleanup and the search-receipt cascade can erase it.
CREATE OR REPLACE RULE action_interest_receipts_no_update AS
ON UPDATE TO action_interest_receipts DO INSTEAD NOTHING;
