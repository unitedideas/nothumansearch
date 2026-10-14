-- Provider-funded action exchange. Organic discovery remains separate and
-- unchanged: these tables can only add a disclosed action offer alongside a
-- site that was already returned organically.
--
-- Privacy boundary: this schema deliberately has no raw query, network
-- identifier, user-agent, contact, principal-identity, or free-form intent
-- payload columns. Agent intent is reduced to controlled fields.
-- USD pilot caps: bounty/receipt $10k (1000000 cents), principal fixed price
-- $1m (100000000), terms credit $100k (10000000), and absolute per-entry plus
-- per-offer cumulative ledger exposure $1m (100000000).

CREATE TABLE IF NOT EXISTS provider_claims (
    id                        UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    account_id                BIGINT NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    site_id                   UUID NOT NULL REFERENCES sites(id) ON DELETE RESTRICT,
    domain_snapshot           TEXT NOT NULL CHECK (domain_snapshot <> ''),
    verification_method       TEXT NOT NULL DEFAULT 'dns_txt'
                              CHECK (verification_method = 'dns_txt'),
    verification_record_name  TEXT NOT NULL CHECK (verification_record_name <> ''),
    verification_token_hash   TEXT NOT NULL UNIQUE
                              CHECK (verification_token_hash ~ '^[0-9a-f]{64}$'),
    challenge_expires_at      TIMESTAMPTZ NOT NULL,
    status                    TEXT NOT NULL DEFAULT 'pending'
                              CHECK (status IN ('pending', 'verified', 'revoked')),
    verified_at               TIMESTAMPTZ,
    verification_last_succeeded_at TIMESTAMPTZ,
    verification_last_attempted_at TIMESTAMPTZ,
    verification_consecutive_failures SMALLINT NOT NULL DEFAULT 0
                              CHECK (verification_consecutive_failures BETWEEN 0 AND 3),
    verification_next_check_at TIMESTAMPTZ,
    verification_lease_id     UUID,
    verification_lease_until  TIMESTAMPTZ,
    revoked_at                TIMESTAMPTZ,
    created_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (
        (status = 'pending'  AND verified_at IS NULL AND revoked_at IS NULL
                             AND challenge_expires_at > created_at) OR
        (status = 'verified' AND verified_at IS NOT NULL AND revoked_at IS NULL) OR
        (status = 'revoked'  AND revoked_at IS NOT NULL)
    ),
    CHECK (
        (status = 'pending' AND verification_last_succeeded_at IS NULL
                            AND verification_last_attempted_at IS NULL
                            AND verification_consecutive_failures = 0
                            AND verification_next_check_at IS NULL
                            AND verification_lease_id IS NULL
                            AND verification_lease_until IS NULL) OR
        (status = 'verified' AND verification_last_succeeded_at IS NOT NULL
                             AND verification_next_check_at IS NOT NULL) OR
        (status = 'revoked' AND verification_next_check_at IS NULL
                            AND verification_lease_id IS NULL
                            AND verification_lease_until IS NULL)
    ),
    CHECK (
        (verification_lease_id IS NULL AND verification_lease_until IS NULL) OR
        (status = 'verified' AND verification_lease_id IS NOT NULL
                             AND verification_lease_until IS NOT NULL)
    )
);

-- Parallel pending challenges prevent one unverified account from reserving a
-- domain. The first account that proves DNS ownership wins; verification
-- revokes the other pending challenges under a site-row lock.
CREATE UNIQUE INDEX IF NOT EXISTS idx_provider_claims_one_verified_site
    ON provider_claims(site_id) WHERE status = 'verified';
CREATE UNIQUE INDEX IF NOT EXISTS idx_provider_claims_one_pending_account_site
    ON provider_claims(account_id, site_id) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_provider_claims_account_status
    ON provider_claims(account_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_provider_claims_pending_expiry
    ON provider_claims(challenge_expires_at) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_provider_claims_verification_due
    ON provider_claims(verification_next_check_at, verification_lease_until)
    WHERE status = 'verified';
CREATE INDEX IF NOT EXISTS idx_provider_claims_verification_freshness
    ON provider_claims(verification_last_succeeded_at)
    WHERE status = 'verified';

CREATE TABLE IF NOT EXISTS provider_api_keys (
    id                 BIGSERIAL PRIMARY KEY,
    provider_claim_id  UUID NOT NULL REFERENCES provider_claims(id) ON DELETE CASCADE,
    key_hash           TEXT NOT NULL UNIQUE CHECK (key_hash ~ '^[0-9a-f]{64}$'),
    key_prefix         TEXT NOT NULL UNIQUE CHECK (key_prefix <> ''),
    status             TEXT NOT NULL DEFAULT 'active'
                       CHECK (status IN ('active', 'revoked')),
    last_used_at       TIMESTAMPTZ,
    revoked_at         TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (id, provider_claim_id),
    CHECK (
        (status = 'active' AND revoked_at IS NULL) OR
        (status = 'revoked' AND revoked_at IS NOT NULL)
    )
);
CREATE INDEX IF NOT EXISTS idx_provider_api_keys_claim_status
    ON provider_api_keys(provider_claim_id, status);
CREATE UNIQUE INDEX IF NOT EXISTS idx_provider_api_keys_one_active_claim
    ON provider_api_keys(provider_claim_id) WHERE status = 'active';

CREATE TABLE IF NOT EXISTS provider_offers (
    id                       UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    provider_claim_id        UUID NOT NULL REFERENCES provider_claims(id) ON DELETE RESTRICT,
    status                   TEXT NOT NULL DEFAULT 'draft'
                             CHECK (status IN ('draft', 'active', 'paused')),
    version                  INTEGER NOT NULL DEFAULT 1 CHECK (version > 0),
    offer_name               TEXT NOT NULL
                             CHECK (length(offer_name) BETWEEN 1 AND 80),
    offer_summary            TEXT NOT NULL
                             CHECK (length(offer_summary) BETWEEN 1 AND 280),
    action_type              TEXT NOT NULL
                             CHECK (action_type IN (
                                 'lead', 'demo', 'trial', 'signup', 'purchase',
                                 'quote', 'application', 'booking'
                             )),
    action_url               TEXT NOT NULL CHECK (
                                 action_url ~ '^https://' AND
                                 octet_length(action_url) <= 1536
                             ),
    disclosure_label         TEXT NOT NULL DEFAULT 'Provider-funded action'
                             CHECK (disclosure_label = 'Provider-funded action'),
    charge_event             TEXT NOT NULL
                             CHECK (charge_event IN ('accepted', 'activated', 'converted')),
    bounty_cents             BIGINT NOT NULL CHECK (bounty_cents BETWEEN 1 AND 1000000),
    currency                 TEXT NOT NULL CHECK (currency = 'usd'),
    principal_price_mode     TEXT NOT NULL
                             CHECK (principal_price_mode IN (
                                 'free', 'fixed', 'quote', 'provider_pricing'
                             )),
    principal_price_cents    BIGINT CHECK (
                                 principal_price_cents IS NULL OR
                                 principal_price_cents BETWEEN 0 AND 100000000
                             ),
    principal_currency       TEXT NOT NULL CHECK (principal_currency = 'usd'),
    billing_mode             TEXT NOT NULL
                             CHECK (billing_mode IN ('prepaid', 'terms')),
    terms_credit_limit_cents BIGINT,
    terms_period_days        INTEGER,
    terms_period_anchor_at   TIMESTAMPTZ,
    terms_evidence_reference TEXT NOT NULL DEFAULT ''
                             CHECK (length(terms_evidence_reference) <= 200),
    activated_at             TIMESTAMPTZ,
    paused_at                TIMESTAMPTZ,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (id, provider_claim_id),
    UNIQUE (provider_claim_id, action_type, action_url),
    CHECK (
        (principal_price_mode = 'free' AND principal_price_cents = 0) OR
        (principal_price_mode = 'fixed' AND principal_price_cents > 0) OR
        (principal_price_mode IN ('quote', 'provider_pricing')
                                  AND principal_price_cents IS NULL)
    ),
    CHECK (
        (billing_mode = 'prepaid' AND terms_credit_limit_cents IS NULL
                                  AND terms_period_days IS NULL
                                  AND terms_period_anchor_at IS NULL) OR
        (billing_mode = 'terms' AND terms_credit_limit_cents BETWEEN 1 AND 10000000
                                AND terms_period_days BETWEEN 1 AND 90
                                AND (
                                    (status = 'draft' AND terms_period_anchor_at IS NULL) OR
                                    (status IN ('active','paused') AND terms_period_anchor_at IS NOT NULL)
                                ))
    ),
    CHECK (
        (status = 'draft' AND activated_at IS NULL AND paused_at IS NULL) OR
        (status = 'active' AND activated_at IS NOT NULL AND paused_at IS NULL
                           AND terms_evidence_reference <> '') OR
        (status = 'paused' AND activated_at IS NOT NULL AND paused_at IS NOT NULL
                           AND terms_evidence_reference <> '')
    )
);
CREATE INDEX IF NOT EXISTS idx_provider_offers_claim_status
    ON provider_offers(provider_claim_id, status, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_provider_offers_public_active
    ON provider_offers(provider_claim_id, action_type)
    WHERE status = 'active';

-- Exact disclosed-offer evidence. A search receipt alone proves only organic
-- site membership; this bounded snapshot proves which paid offer/version the
-- agent actually received beside that result. No query or agent identity is
-- stored here.
CREATE TABLE IF NOT EXISTS provider_offers_returned (
    search_receipt_id       UUID NOT NULL REFERENCES search_receipts(id) ON DELETE CASCADE,
    provider_offer_id       UUID NOT NULL,
    provider_claim_id       UUID NOT NULL,
    offer_version_snapshot  INTEGER NOT NULL CHECK (offer_version_snapshot > 0),
    offer_name_snapshot     TEXT NOT NULL CHECK (length(offer_name_snapshot) BETWEEN 1 AND 80),
    action_type_snapshot    TEXT NOT NULL CHECK (action_type_snapshot IN (
                                'lead', 'demo', 'trial', 'signup', 'purchase',
                                'quote', 'application', 'booking'
                            )),
    disclosure_snapshot     TEXT NOT NULL CHECK (disclosure_snapshot = 'Provider-funded action'),
    bounty_cents_snapshot   BIGINT NOT NULL CHECK (bounty_cents_snapshot BETWEEN 1 AND 1000000),
    currency_snapshot       TEXT NOT NULL CHECK (currency_snapshot = 'usd'),
    charge_event_snapshot   TEXT NOT NULL CHECK (
                                charge_event_snapshot IN ('accepted', 'activated', 'converted')
                            ),
    returned_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (search_receipt_id, provider_offer_id),
    FOREIGN KEY (provider_offer_id, provider_claim_id)
        REFERENCES provider_offers(id, provider_claim_id) ON DELETE RESTRICT
);
CREATE INDEX IF NOT EXISTS idx_provider_offers_returned_offer
    ON provider_offers_returned(provider_offer_id, returned_at DESC);
CREATE OR REPLACE RULE provider_offers_returned_no_update AS
ON UPDATE TO provider_offers_returned DO INSTEAD NOTHING;

CREATE TABLE IF NOT EXISTS action_tickets (
    id                    UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    provider_claim_id     UUID NOT NULL,
    provider_offer_id     UUID NOT NULL,
    search_receipt_id     UUID REFERENCES search_receipts(id) ON DELETE SET NULL,
    source_is_synthetic   BOOLEAN NOT NULL DEFAULT false,
    token_hash            TEXT NOT NULL UNIQUE CHECK (token_hash ~ '^[0-9a-f]{64}$'),
    token_nonce           TEXT NOT NULL CHECK (token_nonce ~ '^[A-Za-z0-9_-]{32}$'),
    creation_request_hash TEXT NOT NULL CHECK (creation_request_hash ~ '^[0-9a-f]{64}$'),
    offer_version_snapshot INTEGER NOT NULL CHECK (offer_version_snapshot > 0),
    offer_name_snapshot   TEXT NOT NULL CHECK (length(offer_name_snapshot) BETWEEN 1 AND 80),
    offer_summary_snapshot TEXT NOT NULL CHECK (length(offer_summary_snapshot) BETWEEN 1 AND 280),
    action_type_snapshot  TEXT NOT NULL CHECK (action_type_snapshot IN (
                              'lead', 'demo', 'trial', 'signup', 'purchase',
                              'quote', 'application', 'booking'
                          )),
    action_url_snapshot   TEXT NOT NULL CHECK (
                              action_url_snapshot ~ '^https://' AND
                              octet_length(action_url_snapshot) <= 1536
                          ),
    disclosure_snapshot   TEXT NOT NULL CHECK (disclosure_snapshot = 'Provider-funded action'),
    charge_event_snapshot TEXT NOT NULL
                          CHECK (charge_event_snapshot IN ('accepted', 'activated', 'converted')),
    bounty_cents_snapshot BIGINT NOT NULL CHECK (bounty_cents_snapshot BETWEEN 1 AND 1000000),
    currency_snapshot     TEXT NOT NULL CHECK (currency_snapshot = 'usd'),
    billing_mode_snapshot TEXT NOT NULL CHECK (billing_mode_snapshot IN ('prepaid', 'terms')),
    terms_evidence_reference_snapshot TEXT NOT NULL
                          CHECK (
                              terms_evidence_reference_snapshot <> '' AND
                              length(terms_evidence_reference_snapshot) <= 200
                          ),
    terms_credit_limit_cents_snapshot BIGINT,
    terms_period_days_snapshot INTEGER,
    terms_period_anchor_at_snapshot TIMESTAMPTZ,
    attribution_key_id_snapshot TEXT NOT NULL
                          CHECK (attribution_key_id_snapshot ~ '^[A-Za-z0-9][A-Za-z0-9._-]{2,63}$'),
    principal_price_mode_snapshot TEXT NOT NULL CHECK (
                              principal_price_mode_snapshot IN (
                                  'free', 'fixed', 'quote', 'provider_pricing'
                              )
                          ),
    principal_price_cents_snapshot BIGINT CHECK (
                              principal_price_cents_snapshot IS NULL OR
                              principal_price_cents_snapshot BETWEEN 0 AND 100000000
                          ),
    principal_currency_snapshot TEXT NOT NULL
                          CHECK (principal_currency_snapshot = 'usd'),
    demand_topic          TEXT NOT NULL CHECK (demand_topic IN (
                              'payments', 'commerce', 'jobs', 'data', 'search',
                              'weather', 'maps', 'email', 'messaging', 'image',
                              'video', 'audio', 'documents', 'security', 'finance',
                              'health', 'education', 'news', 'analytics',
                              'automation', 'productivity', 'identity', 'storage',
                              'ai-tools', 'developer-tools', 'other', 'redacted'
                          )),
    region_code           TEXT NOT NULL DEFAULT ''
                          CHECK (region_code = '' OR region_code ~ '^[A-Z]{2}(-[A-Z0-9]{1,3})?$'),
    budget_band           TEXT NOT NULL DEFAULT 'unspecified'
                          CHECK (budget_band IN (
                              'unspecified', 'under_100', '100_499',
                              '500_1999', '2000_plus'
                          )),
    urgency               TEXT NOT NULL DEFAULT 'unspecified'
                          CHECK (urgency IN (
                              'unspecified', 'now', '7_days', '30_days', 'researching'
                          )),
    requirement_flags     TEXT[] NOT NULL DEFAULT '{}'
                          CHECK (cardinality(requirement_flags) <= 8)
                          CHECK (requirement_flags <@ ARRAY[
                              'api_access', 'mcp', 'sandbox', 'self_serve',
                              'enterprise', 'compliance', 'multilingual', 'human_support'
                          ]::TEXT[]),
    principal_consent     BOOLEAN NOT NULL CHECK (principal_consent),
    consent_version       TEXT NOT NULL
                          CHECK (consent_version = 'nhs-principal-consent-v1'),
    status                TEXT NOT NULL DEFAULT 'created'
                          CHECK (status IN (
                              'created', 'redirected', 'accepted', 'activated',
                              'converted', 'rejected', 'duplicate', 'invalid', 'expired'
                          )),
    expires_at            TIMESTAMPTZ NOT NULL,
    intent_redacted_at    TIMESTAMPTZ,
    authorization_revoked_at TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (provider_offer_id, provider_claim_id)
        REFERENCES provider_offers(id, provider_claim_id) ON DELETE RESTRICT,
    UNIQUE (id, provider_offer_id),
    UNIQUE (search_receipt_id, provider_offer_id),
    CHECK (expires_at > created_at),
    CHECK (authorization_revoked_at IS NULL OR authorization_revoked_at >= created_at),
    CHECK (
        (principal_price_mode_snapshot = 'free' AND principal_price_cents_snapshot = 0) OR
        (principal_price_mode_snapshot = 'fixed' AND principal_price_cents_snapshot > 0) OR
        (principal_price_mode_snapshot IN ('quote', 'provider_pricing')
            AND principal_price_cents_snapshot IS NULL)
    ),
    CHECK (
        (billing_mode_snapshot = 'prepaid'
            AND terms_credit_limit_cents_snapshot IS NULL
            AND terms_period_days_snapshot IS NULL
            AND terms_period_anchor_at_snapshot IS NULL) OR
        (billing_mode_snapshot = 'terms'
            AND terms_credit_limit_cents_snapshot BETWEEN 1 AND 10000000
            AND terms_period_days_snapshot BETWEEN 1 AND 90
            AND terms_period_anchor_at_snapshot IS NOT NULL)
    ),
    CHECK (
        (intent_redacted_at IS NULL AND demand_topic <> 'redacted') OR
        (intent_redacted_at IS NOT NULL AND demand_topic = 'redacted'
            AND region_code = '' AND budget_band = 'unspecified'
            AND urgency = 'unspecified' AND cardinality(requirement_flags) = 0)
    )
);
CREATE INDEX IF NOT EXISTS idx_action_tickets_offer_status
    ON action_tickets(provider_offer_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_action_tickets_receipt
    ON action_tickets(search_receipt_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_action_tickets_expiry
    ON action_tickets(expires_at) WHERE status IN ('created', 'redirected');

-- amount_cents is signed: funding/credits are positive; charges are negative;
-- adjustments may use either sign but never zero.
CREATE TABLE IF NOT EXISTS provider_budget_ledger (
    id                    BIGSERIAL PRIMARY KEY,
    provider_claim_id     UUID NOT NULL,
    provider_offer_id     UUID NOT NULL,
    action_ticket_id      UUID,
    entry_type            TEXT NOT NULL
                          CHECK (entry_type IN ('fund', 'charge', 'credit', 'adjustment')),
    amount_cents          BIGINT NOT NULL CHECK (amount_cents BETWEEN -100000000 AND 100000000),
    currency              TEXT NOT NULL CHECK (currency = 'usd'),
    external_reference    TEXT NOT NULL CHECK (
                              external_reference <> '' AND
                              length(external_reference) <= 200
                          ),
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (provider_offer_id, provider_claim_id)
        REFERENCES provider_offers(id, provider_claim_id) ON DELETE RESTRICT,
    FOREIGN KEY (action_ticket_id, provider_offer_id)
        REFERENCES action_tickets(id, provider_offer_id) ON DELETE RESTRICT,
    UNIQUE (provider_offer_id, entry_type, external_reference),
    CHECK (
        (entry_type IN ('fund', 'credit') AND amount_cents > 0) OR
        (entry_type = 'charge' AND amount_cents < 0) OR
        (entry_type = 'adjustment' AND amount_cents <> 0)
    ),
    CHECK (
        (entry_type IN ('fund', 'adjustment') AND action_ticket_id IS NULL) OR
        (entry_type IN ('charge', 'credit') AND action_ticket_id IS NOT NULL)
    )
);
CREATE INDEX IF NOT EXISTS idx_provider_budget_offer_created
    ON provider_budget_ledger(provider_offer_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_provider_budget_claim_created
    ON provider_budget_ledger(provider_claim_id, created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_provider_budget_one_charge_per_ticket
    ON provider_budget_ledger(action_ticket_id)
    WHERE entry_type = 'charge' AND action_ticket_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_provider_budget_one_credit_per_ticket
    ON provider_budget_ledger(action_ticket_id)
    WHERE entry_type = 'credit' AND action_ticket_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_provider_budget_unique_funding_reference
    ON provider_budget_ledger(external_reference) WHERE entry_type = 'fund';
CREATE OR REPLACE RULE provider_budget_ledger_no_update AS
ON UPDATE TO provider_budget_ledger DO INSTEAD NOTHING;
CREATE OR REPLACE RULE provider_budget_ledger_no_delete AS
ON DELETE TO provider_budget_ledger DO INSTEAD NOTHING;

CREATE TABLE IF NOT EXISTS outcome_receipts (
    id                    UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    nhs_event_id          UUID NOT NULL UNIQUE,
    provider_claim_id     UUID NOT NULL,
    provider_offer_id     UUID NOT NULL,
    action_ticket_id      UUID NOT NULL,
    provider_api_key_id   BIGINT NOT NULL,
    idempotency_key_hash  TEXT NOT NULL CHECK (idempotency_key_hash ~ '^[0-9a-f]{64}$'),
    payload_hash          TEXT NOT NULL CHECK (payload_hash ~ '^[0-9a-f]{64}$'),
    outcome               TEXT NOT NULL CHECK (outcome IN (
                              'accepted', 'activated', 'converted',
                              'rejected', 'duplicate', 'invalid'
                          )),
    billed_cents          BIGINT NOT NULL DEFAULT 0 CHECK (billed_cents BETWEEN 0 AND 1000000),
    charge_status         TEXT NOT NULL
                          CHECK (charge_status IN ('charged', 'credited', 'none')),
    currency              TEXT NOT NULL CHECK (currency = 'usd'),
    signed_receipt        TEXT NOT NULL CHECK (signed_receipt <> ''),
    signature             TEXT NOT NULL CHECK (signature ~ '^[A-Za-z0-9_-]{43}$'),
    provider_reported_at  TIMESTAMPTZ NOT NULL,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (provider_offer_id, provider_claim_id)
        REFERENCES provider_offers(id, provider_claim_id) ON DELETE RESTRICT,
    FOREIGN KEY (action_ticket_id, provider_offer_id)
        REFERENCES action_tickets(id, provider_offer_id) ON DELETE RESTRICT,
    FOREIGN KEY (provider_api_key_id, provider_claim_id)
        REFERENCES provider_api_keys(id, provider_claim_id) ON DELETE RESTRICT,
    UNIQUE (provider_claim_id, idempotency_key_hash),
    UNIQUE (action_ticket_id, outcome),
    CHECK (
        (charge_status = 'none' AND billed_cents = 0) OR
        (charge_status = 'charged' AND billed_cents > 0
            AND outcome IN ('accepted', 'activated', 'converted')) OR
        (charge_status = 'credited' AND billed_cents > 0
            AND outcome IN ('duplicate', 'invalid'))
    )
);
CREATE INDEX IF NOT EXISTS idx_outcome_receipts_ticket_created
    ON outcome_receipts(action_ticket_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_outcome_receipts_claim_created
    ON outcome_receipts(provider_claim_id, created_at DESC);
CREATE OR REPLACE RULE outcome_receipts_no_update AS
ON UPDATE TO outcome_receipts DO INSTEAD NOTHING;
CREATE OR REPLACE RULE outcome_receipts_no_delete AS
ON DELETE TO outcome_receipts DO INSTEAD NOTHING;

-- Append-only operator evidence for cross-tenant commercial actions. References
-- are exact non-secret IDs into the owner-controlled evidence system; no notes,
-- contact data, secrets, or mutable JSON payload is accepted.
CREATE TABLE IF NOT EXISTS provider_admin_audit_events (
    id                    BIGSERIAL PRIMARY KEY,
    provider_claim_id     UUID NOT NULL,
    provider_offer_id     UUID NOT NULL,
    event_type            TEXT NOT NULL
                          CHECK (event_type IN ('activate', 'emergency_pause')),
    operator_reference    TEXT NOT NULL
                          CHECK (operator_reference <> '' AND length(operator_reference) <= 200),
    evidence_reference    TEXT NOT NULL
                          CHECK (evidence_reference <> '' AND length(evidence_reference) <= 200),
    previous_status       TEXT NOT NULL CHECK (previous_status IN ('draft','active','paused')),
    new_status            TEXT NOT NULL CHECK (new_status IN ('active','paused')),
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (provider_offer_id, provider_claim_id)
        REFERENCES provider_offers(id, provider_claim_id) ON DELETE RESTRICT
);
CREATE INDEX IF NOT EXISTS idx_provider_admin_audit_offer_created
    ON provider_admin_audit_events(provider_offer_id, created_at DESC, id DESC);
CREATE OR REPLACE RULE provider_admin_audit_no_update AS
ON UPDATE TO provider_admin_audit_events DO INSTEAD NOTHING;
CREATE OR REPLACE RULE provider_admin_audit_no_delete AS
ON DELETE TO provider_admin_audit_events DO INSTEAD NOTHING;

-- Search receipts have a 30-day retention boundary. Redact the controlled
-- intent fields before a receipt is pruned while retaining only the commercial
-- ticket, consent attestation, immutable offer snapshot, and outcome receipts.
CREATE OR REPLACE RULE redact_action_ticket_intent_on_receipt_delete AS
ON DELETE TO search_receipts DO ALSO
    UPDATE action_tickets
       SET search_receipt_id=NULL,
           demand_topic='redacted',
           region_code='',
           budget_band='unspecified',
           urgency='unspecified',
           requirement_flags='{}',
           intent_redacted_at=COALESCE(intent_redacted_at,NOW()),
           updated_at=NOW()
     WHERE search_receipt_id=OLD.id;
