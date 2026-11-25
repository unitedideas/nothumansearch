-- Truthful provider-pilot commercial evidence.
--
-- Nothing in this migration backfills a qualifying company or commitment from
-- legacy accounts, offer status, free-form evidence references, or budget rows.
-- A provider-authenticated acceptance and a separate owner verification are
-- both required. Company identity is an owner-generated keyed digest held
-- against one canonical pilot claim; no raw legal identity is stored here.
--
-- These tables contain only bounded commercial IDs, controlled fields, keyed
-- digests, and exact external evidence references. They deliberately have no
-- query, contact, agent, principal, network, or free-form intent fields.

-- Ticket preparation and positive outcomes now require a separately consented,
-- NHS-observed handoff. Never strand a live ticket created by the earlier
-- contract: the owner must first complete, revoke, or allow every such
-- authorization to expire and explicitly cut clients over.
LOCK TABLE public.action_tickets IN ACCESS EXCLUSIVE MODE;
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM action_tickets
        WHERE status IN ('created','redirected','accepted','activated')
          AND expires_at > clock_timestamp()
          AND authorization_revoked_at IS NULL
    ) THEN
        RAISE EXCEPTION 'migration 022 requires zero live pre-handoff action tickets'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_action_handoff_cutover_inflight';
    END IF;
END;
$$;

ALTER TABLE provider_offers
    ADD COLUMN IF NOT EXISTS commercial_terms_contract_version TEXT NOT NULL DEFAULT ''
        CHECK (commercial_terms_contract_version IN ('', 'nhs-provider-commercial-terms-v1'));
ALTER TABLE provider_offers
    ADD COLUMN IF NOT EXISTS commercial_terms_sha256 TEXT NOT NULL DEFAULT ''
        CHECK (commercial_terms_sha256 = '' OR commercial_terms_sha256 ~ '^[0-9a-f]{64}$');
ALTER TABLE provider_offers_returned
    ADD COLUMN IF NOT EXISTS commercial_terms_contract_version_snapshot TEXT NOT NULL DEFAULT ''
        CHECK (commercial_terms_contract_version_snapshot IN ('', 'nhs-provider-commercial-terms-v1'));
ALTER TABLE provider_offers_returned
    ADD COLUMN IF NOT EXISTS commercial_terms_sha256_snapshot TEXT NOT NULL DEFAULT ''
        CHECK (commercial_terms_sha256_snapshot = '' OR commercial_terms_sha256_snapshot ~ '^[0-9a-f]{64}$');
ALTER TABLE action_tickets
    ADD COLUMN IF NOT EXISTS commercial_terms_contract_version_snapshot TEXT NOT NULL DEFAULT ''
        CHECK (commercial_terms_contract_version_snapshot IN ('', 'nhs-provider-commercial-terms-v1'));
ALTER TABLE action_tickets
    ADD COLUMN IF NOT EXISTS commercial_terms_sha256_snapshot TEXT NOT NULL DEFAULT ''
        CHECK (commercial_terms_sha256_snapshot = '' OR commercial_terms_sha256_snapshot ~ '^[0-9a-f]{64}$');
-- Existing legacy rows retain the fast-default value, but an old binary that
-- omits either new snapshot after this migration must fail NOT NULL rather than
-- minting another unusable pre-handoff ticket.
ALTER TABLE action_tickets
    ALTER COLUMN commercial_terms_contract_version_snapshot DROP DEFAULT;
ALTER TABLE action_tickets
    ALTER COLUMN commercial_terms_sha256_snapshot DROP DEFAULT;

-- Provider API-key-authenticated acceptances are immutable. The pilot-company
-- acceptance is deliberately separate from the owner-verified company mapping.
-- Terms renewals form a single, non-branching chain and extend the prior
-- acceptance under the same canonical terms hash.
CREATE TABLE IF NOT EXISTS provider_commercial_acceptance_events (
    id                         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    provider_claim_id          UUID NOT NULL,
    provider_offer_id          UUID,
    provider_api_key_id        BIGINT NOT NULL,
    event_type                 TEXT NOT NULL CHECK (event_type IN (
                                   'pilot_company', 'terms_acceptance', 'terms_renewal'
                               )),
    related_acceptance_event_id UUID,
    offer_version_snapshot     INTEGER,
    terms_contract_version     TEXT NOT NULL DEFAULT '',
    exact_terms_sha256         TEXT NOT NULL DEFAULT '',
    idempotency_key_hash       TEXT NOT NULL CHECK (idempotency_key_hash ~ '^[0-9a-f]{64}$'),
    payload_hash               TEXT NOT NULL CHECK (payload_hash ~ '^[0-9a-f]{64}$'),
    provider_acceptance_reference TEXT NOT NULL CHECK (
                                   provider_acceptance_reference <> '' AND
                                   length(provider_acceptance_reference) <= 200
                               ),
    provider_accepted_at       TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
    valid_until                TIMESTAMPTZ,
    created_at                 TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
    UNIQUE (id, provider_claim_id),
    UNIQUE (provider_claim_id, idempotency_key_hash),
    UNIQUE (provider_claim_id, provider_acceptance_reference),
    FOREIGN KEY (provider_offer_id, provider_claim_id)
        REFERENCES provider_offers(id, provider_claim_id) ON DELETE RESTRICT,
    FOREIGN KEY (provider_api_key_id, provider_claim_id)
        REFERENCES provider_api_keys(id, provider_claim_id) ON DELETE RESTRICT,
    FOREIGN KEY (related_acceptance_event_id, provider_claim_id)
        REFERENCES provider_commercial_acceptance_events(id, provider_claim_id)
        ON DELETE RESTRICT,
    CHECK (provider_accepted_at = created_at),
    CHECK (
        (event_type = 'pilot_company'
            AND provider_offer_id IS NULL
            AND related_acceptance_event_id IS NULL
            AND offer_version_snapshot IS NULL
            AND terms_contract_version = ''
            AND exact_terms_sha256 = ''
            AND valid_until IS NULL) OR
        (event_type = 'terms_acceptance'
            AND provider_offer_id IS NOT NULL
            AND related_acceptance_event_id IS NULL
            AND offer_version_snapshot > 0
            AND terms_contract_version = 'nhs-provider-commercial-terms-v1'
            AND exact_terms_sha256 ~ '^[0-9a-f]{64}$'
            AND valid_until > provider_accepted_at) OR
        (event_type = 'terms_renewal'
            AND provider_offer_id IS NOT NULL
            AND related_acceptance_event_id IS NOT NULL
            AND offer_version_snapshot > 0
            AND terms_contract_version = 'nhs-provider-commercial-terms-v1'
            AND exact_terms_sha256 ~ '^[0-9a-f]{64}$'
            AND valid_until > provider_accepted_at)
    )
);
CREATE INDEX IF NOT EXISTS idx_provider_commercial_acceptance_claim_created
    ON provider_commercial_acceptance_events(provider_claim_id, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_provider_commercial_acceptance_offer_created
    ON provider_commercial_acceptance_events(provider_offer_id, created_at DESC, id DESC)
    WHERE provider_offer_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_provider_commercial_acceptance_one_renewal
    ON provider_commercial_acceptance_events(related_acceptance_event_id)
    WHERE event_type = 'terms_renewal';
CREATE OR REPLACE RULE provider_commercial_acceptance_no_update AS
ON UPDATE TO provider_commercial_acceptance_events DO INSTEAD NOTHING;
CREATE OR REPLACE RULE provider_commercial_acceptance_no_delete AS
ON DELETE TO provider_commercial_acceptance_events DO INSTEAD NOTHING;

-- One canonical pilot claim per externally deduplicated company. The keyed
-- digest is produced in the owner evidence system; NHS never receives the raw
-- billing/legal counterparty key. Requiring a provider-authenticated company
-- acceptance prevents owner-only rows from satisfying the commercial gate.
CREATE TABLE IF NOT EXISTS provider_pilot_companies (
    id                         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_key_hash           TEXT NOT NULL UNIQUE CHECK (company_key_hash ~ '^[0-9a-f]{64}$'),
    provider_claim_id          UUID NOT NULL UNIQUE REFERENCES provider_claims(id) ON DELETE RESTRICT,
    provider_api_key_id        BIGINT NOT NULL,
    provider_acceptance_event_id UUID NOT NULL UNIQUE,
    provider_acceptance_reference TEXT NOT NULL CHECK (
                                   provider_acceptance_reference <> '' AND
                                   length(provider_acceptance_reference) <= 200
                               ),
    identity_evidence_reference TEXT NOT NULL CHECK (
                                   identity_evidence_reference <> '' AND
                                   length(identity_evidence_reference) <= 200
                               ),
    operator_reference         TEXT NOT NULL CHECK (
                                   operator_reference <> '' AND
                                   length(operator_reference) <= 200
                               ),
    provider_accepted_at       TIMESTAMPTZ NOT NULL,
    owner_verified_at          TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
    created_at                 TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
    UNIQUE (id, provider_claim_id),
    FOREIGN KEY (provider_api_key_id, provider_claim_id)
        REFERENCES provider_api_keys(id, provider_claim_id) ON DELETE RESTRICT,
    FOREIGN KEY (provider_acceptance_event_id, provider_claim_id)
        REFERENCES provider_commercial_acceptance_events(id, provider_claim_id)
        ON DELETE RESTRICT,
    CHECK (provider_accepted_at <= owner_verified_at),
    CHECK (owner_verified_at = created_at)
);
CREATE INDEX IF NOT EXISTS idx_provider_pilot_companies_claim
    ON provider_pilot_companies(provider_claim_id, owner_verified_at DESC);
CREATE OR REPLACE RULE provider_pilot_companies_no_update AS
ON UPDATE TO provider_pilot_companies DO INSTEAD NOTHING;
CREATE OR REPLACE RULE provider_pilot_companies_no_delete AS
ON DELETE TO provider_pilot_companies DO INSTEAD NOTHING;

-- Owner-verified commitment evidence. Funding events are written atomically
-- with their budget-ledger rows by the application. A qualifying-action link
-- is mandatory for a fund to count as replenishment; ordering by insertion time
-- is intentionally insufficient. Terms rows must point to an authenticated
-- provider acceptance and renewals must extend the exact same terms hash.
CREATE TABLE IF NOT EXISTS provider_commercial_commitment_events (
    id                         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    provider_pilot_company_id  UUID NOT NULL,
    provider_claim_id          UUID NOT NULL,
    provider_offer_id          UUID NOT NULL,
    provider_api_key_id        BIGINT NOT NULL,
    event_type                 TEXT NOT NULL CHECK (event_type IN (
                                   'prepaid_fund', 'fund_reversal',
                                   'terms_acceptance', 'terms_renewal'
                               )),
    related_event_id           UUID REFERENCES provider_commercial_commitment_events(id) ON DELETE RESTRICT,
    provider_acceptance_event_id UUID,
    budget_ledger_entry_id     BIGINT UNIQUE REFERENCES provider_budget_ledger(id) ON DELETE RESTRICT,
    qualifying_action_ticket_id UUID REFERENCES action_tickets(id) ON DELETE RESTRICT,
    offer_version_snapshot     INTEGER NOT NULL CHECK (offer_version_snapshot > 0),
    terms_contract_version     TEXT NOT NULL CHECK (
                                   terms_contract_version = 'nhs-provider-commercial-terms-v1'
                               ),
    exact_terms_sha256         TEXT NOT NULL CHECK (exact_terms_sha256 ~ '^[0-9a-f]{64}$'),
    amount_cents               BIGINT NOT NULL CHECK (amount_cents BETWEEN -100000000 AND 100000000),
    currency                   TEXT NOT NULL CHECK (currency = 'usd'),
    source_system              TEXT NOT NULL CHECK (
                                   source_system <> '' AND length(source_system) <= 80
                               ),
    source_event_id            TEXT NOT NULL CHECK (
                                   source_event_id <> '' AND length(source_event_id) <= 120
                               ),
    source_effective_at        TIMESTAMPTZ NOT NULL,
    provider_acceptance_reference TEXT NOT NULL CHECK (
                                   provider_acceptance_reference <> '' AND
                                   length(provider_acceptance_reference) <= 200
                               ),
    operator_reference         TEXT NOT NULL CHECK (
                                   operator_reference <> '' AND
                                   length(operator_reference) <= 200
                               ),
    owner_evidence_reference   TEXT NOT NULL CHECK (
                                   owner_evidence_reference <> '' AND
                                   length(owner_evidence_reference) <= 200
                               ),
    provider_accepted_at       TIMESTAMPTZ NOT NULL,
    valid_until                TIMESTAMPTZ,
    owner_verified_at          TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
    created_at                 TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
    UNIQUE (source_system, source_event_id),
    UNIQUE (provider_acceptance_event_id),
    FOREIGN KEY (provider_pilot_company_id, provider_claim_id)
        REFERENCES provider_pilot_companies(id, provider_claim_id) ON DELETE RESTRICT,
    FOREIGN KEY (provider_offer_id, provider_claim_id)
        REFERENCES provider_offers(id, provider_claim_id) ON DELETE RESTRICT,
    FOREIGN KEY (provider_api_key_id, provider_claim_id)
        REFERENCES provider_api_keys(id, provider_claim_id) ON DELETE RESTRICT,
    FOREIGN KEY (provider_acceptance_event_id, provider_claim_id)
        REFERENCES provider_commercial_acceptance_events(id, provider_claim_id)
        ON DELETE RESTRICT,
    CHECK (source_effective_at <= owner_verified_at),
    CHECK (provider_accepted_at <= owner_verified_at),
    CHECK (owner_verified_at = created_at),
    CHECK (
        (event_type = 'prepaid_fund'
            AND related_event_id IS NULL
            AND provider_acceptance_event_id IS NULL
            AND budget_ledger_entry_id IS NOT NULL
            AND amount_cents > 0
            AND valid_until IS NULL) OR
        (event_type = 'fund_reversal'
            AND related_event_id IS NOT NULL
            AND provider_acceptance_event_id IS NULL
            AND budget_ledger_entry_id IS NOT NULL
            AND qualifying_action_ticket_id IS NULL
            AND amount_cents < 0
            AND valid_until IS NULL) OR
        (event_type = 'terms_acceptance'
            AND related_event_id IS NULL
            AND provider_acceptance_event_id IS NOT NULL
            AND budget_ledger_entry_id IS NULL
            AND qualifying_action_ticket_id IS NULL
            AND amount_cents = 0
            AND valid_until > provider_accepted_at) OR
        (event_type = 'terms_renewal'
            AND related_event_id IS NOT NULL
            AND provider_acceptance_event_id IS NOT NULL
            AND budget_ledger_entry_id IS NULL
            AND qualifying_action_ticket_id IS NULL
            AND amount_cents = 0
            AND valid_until > provider_accepted_at)
    )
);
CREATE INDEX IF NOT EXISTS idx_provider_commercial_commitment_company_created
    ON provider_commercial_commitment_events(provider_pilot_company_id, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_provider_commercial_commitment_offer_created
    ON provider_commercial_commitment_events(provider_offer_id, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_provider_commercial_commitment_related
    ON provider_commercial_commitment_events(related_event_id, created_at, id)
    WHERE related_event_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_provider_commercial_one_terms_renewal
    ON provider_commercial_commitment_events(related_event_id)
    WHERE event_type = 'terms_renewal';
CREATE OR REPLACE RULE provider_commercial_commitment_no_update AS
ON UPDATE TO provider_commercial_commitment_events DO INSTEAD NOTHING;
CREATE OR REPLACE RULE provider_commercial_commitment_no_delete AS
ON DELETE TO provider_commercial_commitment_events DO INSTEAD NOTHING;

-- NHS records a handoff only when the principal's agent presents the exact
-- ticket bearer token to the dedicated handoff surface. The receipt contains
-- no query, identity, contact, referrer, user-agent, or network data. It binds
-- one durable observation to the exact ticket and commercial terms that were
-- already consented to; it never charges either party.
CREATE TABLE IF NOT EXISTS provider_action_handoff_receipts (
    id                         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    action_ticket_id           UUID NOT NULL UNIQUE,
    provider_claim_id          UUID NOT NULL,
    provider_offer_id          UUID NOT NULL,
    offer_version_snapshot     INTEGER NOT NULL CHECK (offer_version_snapshot > 0),
    commercial_terms_contract_version_snapshot TEXT NOT NULL CHECK (
                                   commercial_terms_contract_version_snapshot =
                                   'nhs-provider-commercial-terms-v1'
                               ),
    commercial_terms_sha256_snapshot TEXT NOT NULL CHECK (
                                   commercial_terms_sha256_snapshot ~ '^[0-9a-f]{64}$'
                               ),
    presented_token_hash       TEXT NOT NULL CHECK (presented_token_hash ~ '^[0-9a-f]{64}$'),
    principal_handoff_consent  BOOLEAN NOT NULL CHECK (principal_handoff_consent),
    handoff_consent_version    TEXT NOT NULL CHECK (
                                   handoff_consent_version =
                                   'nhs-provider-handoff-consent-v1'
                               ),
    event_contract_version     TEXT NOT NULL CHECK (
                                   event_contract_version = 'nhs-action-handoff-v1'
                               ),
    observed_at                TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
    created_at                 TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
    FOREIGN KEY (provider_offer_id, provider_claim_id)
        REFERENCES provider_offers(id, provider_claim_id) ON DELETE RESTRICT,
    FOREIGN KEY (action_ticket_id, provider_offer_id)
        REFERENCES action_tickets(id, provider_offer_id) ON DELETE RESTRICT,
    CHECK (observed_at = created_at)
);
CREATE INDEX IF NOT EXISTS idx_provider_action_handoff_offer_observed
    ON provider_action_handoff_receipts(provider_offer_id, observed_at DESC, id DESC);
CREATE OR REPLACE RULE provider_action_handoff_no_update AS
ON UPDATE TO provider_action_handoff_receipts DO INSTEAD NOTHING;
CREATE OR REPLACE RULE provider_action_handoff_no_delete AS
ON DELETE TO provider_action_handoff_receipts DO INSTEAD NOTHING;

CREATE OR REPLACE FUNCTION public.enforce_provider_commercial_acceptance_event()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    claim_status TEXT;
    claim_last_verified TIMESTAMPTZ;
    key_status TEXT;
    offer_claim UUID;
    offer_status TEXT;
    offer_billing_mode TEXT;
    offer_version INTEGER;
    offer_contract_version TEXT;
    offer_terms_sha256 TEXT;
    offer_period_days INTEGER;
    prior provider_commercial_acceptance_events%ROWTYPE;
BEGIN
    SELECT status, verification_last_succeeded_at
      INTO claim_status, claim_last_verified
      FROM provider_claims
     WHERE id=NEW.provider_claim_id
     FOR UPDATE;
    IF claim_status IS DISTINCT FROM 'verified' OR
       claim_last_verified IS NULL OR
       claim_last_verified <= clock_timestamp() - INTERVAL '7 days' THEN
        RAISE EXCEPTION 'fresh verified provider claim required'
            USING ERRCODE='23514', CONSTRAINT='provider_commercial_acceptance_fresh_claim';
    END IF;

    SELECT status INTO key_status
      FROM provider_api_keys
     WHERE id=NEW.provider_api_key_id AND provider_claim_id=NEW.provider_claim_id
     FOR UPDATE;
    IF key_status IS DISTINCT FROM 'active' THEN
        RAISE EXCEPTION 'active provider key required'
            USING ERRCODE='23514', CONSTRAINT='provider_commercial_acceptance_active_key';
    END IF;

    IF NEW.event_type = 'pilot_company' THEN
        RETURN NEW;
    END IF;

    SELECT provider_claim_id, status, billing_mode, version,
           commercial_terms_contract_version, commercial_terms_sha256,
           terms_period_days
      INTO offer_claim, offer_status, offer_billing_mode, offer_version,
           offer_contract_version, offer_terms_sha256, offer_period_days
      FROM provider_offers
     WHERE id=NEW.provider_offer_id
     FOR UPDATE;
    IF offer_claim IS DISTINCT FROM NEW.provider_claim_id OR
       offer_status NOT IN ('draft','active','paused') OR
       offer_billing_mode IS DISTINCT FROM 'terms' OR
       offer_contract_version IS DISTINCT FROM NEW.terms_contract_version OR
       offer_terms_sha256 IS DISTINCT FROM NEW.exact_terms_sha256 OR
       offer_version IS DISTINCT FROM NEW.offer_version_snapshot OR
       offer_period_days IS NULL THEN
        RAISE EXCEPTION 'acceptance does not match exact current terms'
            USING ERRCODE='23514', CONSTRAINT='provider_commercial_acceptance_exact_terms';
    END IF;

    IF NEW.event_type = 'terms_renewal' THEN
        SELECT * INTO prior
          FROM provider_commercial_acceptance_events
         WHERE id=NEW.related_acceptance_event_id;
        IF prior.id IS NULL OR
           prior.event_type NOT IN ('terms_acceptance','terms_renewal') OR
           prior.provider_claim_id IS DISTINCT FROM NEW.provider_claim_id OR
           prior.provider_offer_id IS DISTINCT FROM NEW.provider_offer_id OR
           prior.exact_terms_sha256 IS DISTINCT FROM NEW.exact_terms_sha256 OR
           NEW.valid_until <= prior.valid_until THEN
            RAISE EXCEPTION 'terms renewal must extend the same exact acceptance chain'
                USING ERRCODE='23514', CONSTRAINT='provider_commercial_acceptance_renewal_chain';
        END IF;
    END IF;
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS provider_commercial_acceptance_event_enforced
    ON provider_commercial_acceptance_events;
CREATE TRIGGER provider_commercial_acceptance_event_enforced
BEFORE INSERT ON provider_commercial_acceptance_events
FOR EACH ROW EXECUTE FUNCTION public.enforce_provider_commercial_acceptance_event();

CREATE OR REPLACE FUNCTION public.enforce_provider_pilot_company()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    accepted provider_commercial_acceptance_events%ROWTYPE;
BEGIN
    SELECT * INTO accepted
      FROM provider_commercial_acceptance_events
     WHERE id=NEW.provider_acceptance_event_id;
    IF accepted.id IS NULL OR
       accepted.event_type IS DISTINCT FROM 'pilot_company' OR
       accepted.provider_claim_id IS DISTINCT FROM NEW.provider_claim_id OR
       accepted.provider_api_key_id IS DISTINCT FROM NEW.provider_api_key_id OR
       accepted.provider_acceptance_reference IS DISTINCT FROM NEW.provider_acceptance_reference OR
       accepted.provider_accepted_at IS DISTINCT FROM NEW.provider_accepted_at THEN
        RAISE EXCEPTION 'pilot company requires exact provider-authenticated acceptance'
            USING ERRCODE='23514', CONSTRAINT='provider_pilot_company_exact_acceptance';
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM provider_claims claim
        WHERE claim.id=NEW.provider_claim_id
          AND claim.status='verified'
          AND claim.verification_last_succeeded_at > clock_timestamp() - INTERVAL '7 days'
    ) THEN
        RAISE EXCEPTION 'fresh verified provider claim required'
            USING ERRCODE='23514', CONSTRAINT='provider_pilot_company_fresh_claim';
    END IF;
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS provider_pilot_company_enforced ON provider_pilot_companies;
CREATE TRIGGER provider_pilot_company_enforced
BEFORE INSERT ON provider_pilot_companies
FOR EACH ROW EXECUTE FUNCTION public.enforce_provider_pilot_company();

CREATE OR REPLACE FUNCTION public.enforce_provider_commercial_commitment_event()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    company_claim UUID;
    company_key BIGINT;
    company_acceptance TEXT;
    claim_status TEXT;
    claim_last_verified TIMESTAMPTZ;
    offer_claim UUID;
    offer_version INTEGER;
    offer_contract_version TEXT;
    offer_terms_sha256 TEXT;
    offer_billing_mode TEXT;
    ledger provider_budget_ledger%ROWTYPE;
    accepted provider_commercial_acceptance_events%ROWTYPE;
    related provider_commercial_commitment_events%ROWTYPE;
    reversed_cents NUMERIC;
    charged_cents BIGINT;
    charged_at TIMESTAMPTZ;
BEGIN
    SELECT provider_claim_id, provider_api_key_id, provider_acceptance_reference
      INTO company_claim, company_key, company_acceptance
      FROM provider_pilot_companies
     WHERE id=NEW.provider_pilot_company_id;
    IF company_claim IS DISTINCT FROM NEW.provider_claim_id OR
       (NEW.event_type IN ('prepaid_fund','fund_reversal') AND (
          company_key IS DISTINCT FROM NEW.provider_api_key_id OR
          company_acceptance IS DISTINCT FROM NEW.provider_acceptance_reference
       )) THEN
        RAISE EXCEPTION 'commitment does not match verified pilot company'
            USING ERRCODE='23514', CONSTRAINT='provider_commercial_commitment_company';
    END IF;
    -- Match every application write path's claim-then-offer lock order. The
    -- offer row locks below also close the evidence-insert versus direct terms
    -- update race: either this insert validates the updated terms or the update
    -- waits and then sees the committed immutable evidence.
    SELECT status, verification_last_succeeded_at
      INTO claim_status, claim_last_verified
      FROM provider_claims
     WHERE id=NEW.provider_claim_id
     FOR UPDATE;
    -- A refund or chargeback must remain recordable after ownership expires;
    -- otherwise stale providers would leave overstated settlement evidence.
    IF NEW.event_type <> 'fund_reversal' AND (
       claim_status IS DISTINCT FROM 'verified' OR
       claim_last_verified IS NULL OR
       claim_last_verified <= clock_timestamp() - INTERVAL '7 days') THEN
        RAISE EXCEPTION 'fresh verified provider claim required'
            USING ERRCODE='23514', CONSTRAINT='provider_commercial_commitment_fresh_claim';
    END IF;

    SELECT provider_claim_id, version, commercial_terms_contract_version,
           commercial_terms_sha256, billing_mode
      INTO offer_claim, offer_version, offer_contract_version,
           offer_terms_sha256, offer_billing_mode
      FROM provider_offers
     WHERE id=NEW.provider_offer_id
     FOR UPDATE;
    IF offer_claim IS DISTINCT FROM NEW.provider_claim_id OR
       offer_version IS DISTINCT FROM NEW.offer_version_snapshot OR
       offer_contract_version IS DISTINCT FROM NEW.terms_contract_version OR
       offer_terms_sha256 IS DISTINCT FROM NEW.exact_terms_sha256 OR
       offer_terms_sha256 = '' THEN
        RAISE EXCEPTION 'commitment does not match exact offer terms'
            USING ERRCODE='23514', CONSTRAINT='provider_commercial_commitment_exact_terms';
    END IF;

    IF NEW.event_type IN ('prepaid_fund','fund_reversal') THEN
        IF offer_billing_mode IS DISTINCT FROM 'prepaid' THEN
            RAISE EXCEPTION 'fund commitment requires prepaid offer'
                USING ERRCODE='23514', CONSTRAINT='provider_commercial_commitment_prepaid_offer';
        END IF;
        SELECT * INTO ledger FROM provider_budget_ledger WHERE id=NEW.budget_ledger_entry_id;
        IF ledger.id IS NULL OR
           ledger.provider_claim_id IS DISTINCT FROM NEW.provider_claim_id OR
           ledger.provider_offer_id IS DISTINCT FROM NEW.provider_offer_id OR
           ledger.amount_cents IS DISTINCT FROM NEW.amount_cents OR
           ledger.currency IS DISTINCT FROM NEW.currency OR
           (NEW.event_type='prepaid_fund' AND ledger.entry_type IS DISTINCT FROM 'fund') OR
           (NEW.event_type='fund_reversal' AND ledger.entry_type IS DISTINCT FROM 'adjustment') THEN
            RAISE EXCEPTION 'commitment does not match budget ledger entry'
                USING ERRCODE='23514', CONSTRAINT='provider_commercial_commitment_budget_entry';
        END IF;
    END IF;

    IF NEW.event_type = 'prepaid_fund' AND NEW.qualifying_action_ticket_id IS NOT NULL THEN
        SELECT -charge.amount_cents, charge.created_at
          INTO charged_cents, charged_at
          FROM provider_budget_ledger charge
          JOIN action_tickets ticket ON ticket.id=charge.action_ticket_id
         WHERE charge.action_ticket_id=NEW.qualifying_action_ticket_id
           AND charge.provider_offer_id=NEW.provider_offer_id
           AND charge.entry_type='charge'
           AND NOT ticket.source_is_synthetic
           AND ticket.authorization_revoked_at IS NULL
           AND ticket.status IN ('accepted','activated','converted')
           AND NOT EXISTS (
               SELECT 1 FROM provider_budget_ledger credit
               WHERE credit.action_ticket_id=ticket.id AND credit.entry_type='credit'
           );
        IF charged_cents IS NULL OR NEW.amount_cents < charged_cents OR
           NEW.source_effective_at <= charged_at THEN
            RAISE EXCEPTION 'replenishment requires a later unreversed real charge'
                USING ERRCODE='23514', CONSTRAINT='provider_commercial_commitment_replenishment';
        END IF;
    END IF;

    IF NEW.event_type = 'fund_reversal' THEN
        SELECT * INTO related
          FROM provider_commercial_commitment_events
         WHERE id=NEW.related_event_id
         FOR UPDATE;
        IF related.id IS NULL OR related.event_type IS DISTINCT FROM 'prepaid_fund' OR
           related.provider_pilot_company_id IS DISTINCT FROM NEW.provider_pilot_company_id OR
           related.provider_claim_id IS DISTINCT FROM NEW.provider_claim_id OR
           related.provider_offer_id IS DISTINCT FROM NEW.provider_offer_id OR
           related.currency IS DISTINCT FROM NEW.currency OR
           NEW.source_effective_at < related.source_effective_at THEN
            RAISE EXCEPTION 'fund reversal must reference the same verified fund'
                USING ERRCODE='23514', CONSTRAINT='provider_commercial_commitment_reversal_link';
        END IF;
        SELECT COALESCE(-SUM(amount_cents::numeric),0)
          INTO reversed_cents
          FROM provider_commercial_commitment_events
         WHERE related_event_id=related.id AND event_type='fund_reversal';
        IF reversed_cents > related.amount_cents + NEW.amount_cents THEN
            RAISE EXCEPTION 'fund reversals exceed verified source amount'
                USING ERRCODE='23514', CONSTRAINT='provider_commercial_commitment_reversal_amount';
        END IF;
    END IF;

    IF NEW.event_type IN ('terms_acceptance','terms_renewal') THEN
        IF offer_billing_mode IS DISTINCT FROM 'terms' THEN
            RAISE EXCEPTION 'terms commitment requires terms offer'
                USING ERRCODE='23514', CONSTRAINT='provider_commercial_commitment_terms_offer';
        END IF;
        SELECT * INTO accepted
          FROM provider_commercial_acceptance_events
         WHERE id=NEW.provider_acceptance_event_id;
        IF accepted.id IS NULL OR
           accepted.event_type IS DISTINCT FROM NEW.event_type OR
           accepted.provider_claim_id IS DISTINCT FROM NEW.provider_claim_id OR
           accepted.provider_offer_id IS DISTINCT FROM NEW.provider_offer_id OR
           accepted.provider_api_key_id IS DISTINCT FROM NEW.provider_api_key_id OR
           accepted.offer_version_snapshot IS DISTINCT FROM NEW.offer_version_snapshot OR
           accepted.terms_contract_version IS DISTINCT FROM NEW.terms_contract_version OR
           accepted.exact_terms_sha256 IS DISTINCT FROM NEW.exact_terms_sha256 OR
           accepted.provider_accepted_at IS DISTINCT FROM NEW.provider_accepted_at OR
           accepted.valid_until IS DISTINCT FROM NEW.valid_until THEN
            RAISE EXCEPTION 'terms commitment requires exact provider acceptance'
                USING ERRCODE='23514', CONSTRAINT='provider_commercial_commitment_terms_acceptance';
        END IF;
    END IF;

    IF NEW.event_type = 'terms_renewal' THEN
        SELECT * INTO related
          FROM provider_commercial_commitment_events
         WHERE id=NEW.related_event_id
         FOR UPDATE;
        IF related.id IS NULL OR related.event_type NOT IN ('terms_acceptance','terms_renewal') OR
           related.provider_pilot_company_id IS DISTINCT FROM NEW.provider_pilot_company_id OR
           related.provider_claim_id IS DISTINCT FROM NEW.provider_claim_id OR
           related.provider_offer_id IS DISTINCT FROM NEW.provider_offer_id OR
           related.exact_terms_sha256 IS DISTINCT FROM NEW.exact_terms_sha256 OR
           accepted.related_acceptance_event_id IS DISTINCT FROM related.provider_acceptance_event_id OR
           NEW.valid_until <= related.valid_until THEN
            RAISE EXCEPTION 'verified renewal must extend the same exact terms chain'
                USING ERRCODE='23514', CONSTRAINT='provider_commercial_commitment_terms_renewal';
        END IF;
    END IF;
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS provider_commercial_commitment_event_enforced
    ON provider_commercial_commitment_events;
CREATE TRIGGER provider_commercial_commitment_event_enforced
BEFORE INSERT ON provider_commercial_commitment_events
FOR EACH ROW EXECUTE FUNCTION public.enforce_provider_commercial_commitment_event();

CREATE OR REPLACE FUNCTION public.enforce_provider_action_handoff_receipt()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    claim_status TEXT;
    claim_last_verified TIMESTAMPTZ;
    offer_claim UUID;
    offer_status TEXT;
    offer_version INTEGER;
    offer_billing_mode TEXT;
    offer_contract_version TEXT;
    offer_terms_sha256 TEXT;
    ticket_claim UUID;
    ticket_offer UUID;
    ticket_version INTEGER;
    ticket_contract_version TEXT;
    ticket_terms_sha256 TEXT;
    ticket_token_hash TEXT;
    ticket_synthetic BOOLEAN;
    ticket_consent BOOLEAN;
    ticket_consent_version TEXT;
    ticket_status TEXT;
    ticket_expires_at TIMESTAMPTZ;
    ticket_revoked_at TIMESTAMPTZ;
    observed_now TIMESTAMPTZ;
BEGIN
    -- Preserve the global claim -> offer -> ticket lock order used by action
    -- creation and provider callbacks, then decide freshness against the clock
    -- only after all three authoritative rows are held.
    SELECT status, verification_last_succeeded_at
      INTO claim_status, claim_last_verified
      FROM provider_claims
     WHERE id=NEW.provider_claim_id
     FOR UPDATE;

    SELECT provider_claim_id, status, version, billing_mode,
           commercial_terms_contract_version, commercial_terms_sha256
      INTO offer_claim, offer_status, offer_version, offer_billing_mode,
           offer_contract_version, offer_terms_sha256
      FROM provider_offers
     WHERE id=NEW.provider_offer_id
     FOR UPDATE;

    SELECT provider_claim_id, provider_offer_id, offer_version_snapshot,
           commercial_terms_contract_version_snapshot,
           commercial_terms_sha256_snapshot, token_hash, source_is_synthetic,
           principal_consent, consent_version, status, expires_at,
           authorization_revoked_at
      INTO ticket_claim, ticket_offer, ticket_version,
           ticket_contract_version, ticket_terms_sha256, ticket_token_hash,
           ticket_synthetic, ticket_consent, ticket_consent_version,
           ticket_status, ticket_expires_at, ticket_revoked_at
      FROM action_tickets
     WHERE id=NEW.action_ticket_id
     FOR UPDATE;

    observed_now := clock_timestamp();
    IF claim_status IS DISTINCT FROM 'verified' OR
       claim_last_verified IS NULL OR
       claim_last_verified <= observed_now - INTERVAL '7 days' THEN
        RAISE EXCEPTION 'fresh verified provider claim required for handoff'
            USING ERRCODE='23514', CONSTRAINT='provider_action_handoff_fresh_claim';
    END IF;
    IF offer_claim IS DISTINCT FROM NEW.provider_claim_id OR
       offer_status IS DISTINCT FROM 'active' OR
       offer_version IS DISTINCT FROM NEW.offer_version_snapshot OR
       offer_contract_version IS DISTINCT FROM
           NEW.commercial_terms_contract_version_snapshot OR
       offer_terms_sha256 IS DISTINCT FROM NEW.commercial_terms_sha256_snapshot OR
       offer_terms_sha256 = '' THEN
        RAISE EXCEPTION 'handoff does not match exact active offer terms'
            USING ERRCODE='23514', CONSTRAINT='provider_action_handoff_exact_offer';
    END IF;
    IF ticket_claim IS DISTINCT FROM NEW.provider_claim_id OR
       ticket_offer IS DISTINCT FROM NEW.provider_offer_id OR
       ticket_version IS DISTINCT FROM NEW.offer_version_snapshot OR
       ticket_contract_version IS DISTINCT FROM
           NEW.commercial_terms_contract_version_snapshot OR
       ticket_terms_sha256 IS DISTINCT FROM NEW.commercial_terms_sha256_snapshot OR
       ticket_token_hash IS DISTINCT FROM NEW.presented_token_hash OR
       NOT NEW.principal_handoff_consent OR
       NEW.handoff_consent_version IS DISTINCT FROM
           'nhs-provider-handoff-consent-v1' OR
       ticket_synthetic OR NOT ticket_consent OR
       ticket_consent_version IS DISTINCT FROM 'nhs-principal-consent-v1' OR
       ticket_status IS DISTINCT FROM 'created' OR
       ticket_expires_at <= observed_now OR ticket_revoked_at IS NOT NULL THEN
        RAISE EXCEPTION 'handoff requires an exact live consent-attested ticket bearer'
            USING ERRCODE='23514', CONSTRAINT='provider_action_handoff_exact_ticket';
    END IF;
    IF NOT EXISTS (
        SELECT 1
        FROM provider_pilot_companies company
        WHERE company.provider_claim_id=NEW.provider_claim_id
          AND NOT EXISTS (
              SELECT 1 FROM provider_budget_ledger unverified
              WHERE unverified.provider_offer_id=NEW.provider_offer_id
                AND unverified.entry_type IN ('fund','adjustment')
                AND NOT EXISTS (
                    SELECT 1 FROM provider_commercial_commitment_events linked
                    WHERE linked.budget_ledger_entry_id=unverified.id
                      AND (
                        (linked.event_type='prepaid_fund' AND unverified.entry_type='fund') OR
                        (linked.event_type='fund_reversal' AND unverified.entry_type='adjustment')
                      )
                )
          )
          AND (
            (offer_billing_mode='prepaid' AND EXISTS (
                SELECT 1 FROM provider_commercial_commitment_events fund
                WHERE fund.provider_pilot_company_id=company.id
                  AND fund.provider_claim_id=NEW.provider_claim_id
                  AND fund.provider_offer_id=NEW.provider_offer_id
                  AND fund.event_type='prepaid_fund'
                  AND fund.offer_version_snapshot=NEW.offer_version_snapshot
                  AND fund.terms_contract_version=
                      NEW.commercial_terms_contract_version_snapshot
                  AND fund.exact_terms_sha256=NEW.commercial_terms_sha256_snapshot
                  AND fund.amount_cents + COALESCE((
                      SELECT SUM(reversal.amount_cents)
                      FROM provider_commercial_commitment_events reversal
                      WHERE reversal.related_event_id=fund.id
                        AND reversal.event_type='fund_reversal'
                  ),0) > 0
            )) OR
            (offer_billing_mode='terms' AND EXISTS (
                SELECT 1 FROM provider_commercial_commitment_events terms
                WHERE terms.provider_pilot_company_id=company.id
                  AND terms.provider_claim_id=NEW.provider_claim_id
                  AND terms.provider_offer_id=NEW.provider_offer_id
                  AND terms.event_type IN ('terms_acceptance','terms_renewal')
                  AND terms.offer_version_snapshot=NEW.offer_version_snapshot
                  AND terms.terms_contract_version=
                      NEW.commercial_terms_contract_version_snapshot
                  AND terms.exact_terms_sha256=NEW.commercial_terms_sha256_snapshot
                  AND terms.valid_until > observed_now
            ))
          )
    ) THEN
        RAISE EXCEPTION 'verified commercial evidence required for handoff'
            USING ERRCODE='23514', CONSTRAINT='provider_action_handoff_commercial_evidence';
    END IF;
    NEW.event_contract_version := 'nhs-action-handoff-v1';
    NEW.observed_at := observed_now;
    NEW.created_at := observed_now;
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS provider_action_handoff_receipt_enforced
    ON provider_action_handoff_receipts;
CREATE TRIGGER provider_action_handoff_receipt_enforced
BEFORE INSERT ON provider_action_handoff_receipts
FOR EACH ROW EXECUTE FUNCTION public.enforce_provider_action_handoff_receipt();

-- A positive ticket state is an NHS-observed fact, not an application hint.
-- Keep this invariant below the application so a stale binary or direct SQL
-- writer cannot bypass the separately consented handoff. The controlled flow
-- inserts the immutable receipt before changing created -> redirected in the
-- same transaction; later positive provider outcomes retain that receipt.
CREATE OR REPLACE FUNCTION public.enforce_action_ticket_observed_handoff_status()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM provider_action_handoff_receipts receipt
        WHERE receipt.action_ticket_id=NEW.id
          AND receipt.provider_claim_id=NEW.provider_claim_id
          AND receipt.provider_offer_id=NEW.provider_offer_id
          AND receipt.offer_version_snapshot=NEW.offer_version_snapshot
          AND receipt.commercial_terms_contract_version_snapshot=
              NEW.commercial_terms_contract_version_snapshot
          AND receipt.commercial_terms_sha256_snapshot=
              NEW.commercial_terms_sha256_snapshot
          AND receipt.event_contract_version='nhs-action-handoff-v1'
    ) THEN
        RAISE EXCEPTION 'positive ticket status requires an observed handoff receipt'
            USING ERRCODE='23514',
                  CONSTRAINT='action_ticket_observed_handoff_required';
    END IF;
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS action_ticket_observed_handoff_status_enforced
    ON action_tickets;
CREATE TRIGGER action_ticket_observed_handoff_status_enforced
AFTER UPDATE OF status ON action_tickets
FOR EACH ROW
WHEN (
    OLD.status IS DISTINCT FROM NEW.status AND
    NEW.status IN ('redirected','accepted','activated','converted')
)
EXECUTE FUNCTION public.enforce_action_ticket_observed_handoff_status();
DROP TRIGGER IF EXISTS action_ticket_observed_handoff_insert_enforced
    ON action_tickets;
CREATE TRIGGER action_ticket_observed_handoff_insert_enforced
AFTER INSERT ON action_tickets
FOR EACH ROW
WHEN (NEW.status IN ('redirected','accepted','activated','converted'))
EXECUTE FUNCTION public.enforce_action_ticket_observed_handoff_status();

-- Once either party has accepted an offer's commercial contract, direct SQL
-- cannot rewrite any hash-bearing term beneath the immutable evidence. Status,
-- activation timestamps, and budget state remain independently mutable through
-- their existing controlled workflows.
CREATE OR REPLACE FUNCTION public.enforce_provider_offer_commercial_immutability()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF ROW(
        OLD.version, OLD.offer_name, OLD.offer_summary, OLD.action_type,
        OLD.action_url, OLD.disclosure_label, OLD.charge_event,
        OLD.bounty_cents, OLD.currency, OLD.principal_price_mode,
        OLD.principal_price_cents, OLD.principal_currency, OLD.billing_mode,
        OLD.terms_credit_limit_cents, OLD.terms_period_days,
        OLD.commercial_terms_contract_version, OLD.commercial_terms_sha256
    ) IS DISTINCT FROM ROW(
        NEW.version, NEW.offer_name, NEW.offer_summary, NEW.action_type,
        NEW.action_url, NEW.disclosure_label, NEW.charge_event,
        NEW.bounty_cents, NEW.currency, NEW.principal_price_mode,
        NEW.principal_price_cents, NEW.principal_currency, NEW.billing_mode,
        NEW.terms_credit_limit_cents, NEW.terms_period_days,
        NEW.commercial_terms_contract_version, NEW.commercial_terms_sha256
    ) AND (
        EXISTS (
            SELECT 1 FROM provider_commercial_acceptance_events accepted
            WHERE accepted.provider_offer_id=OLD.id
        ) OR EXISTS (
            SELECT 1 FROM provider_commercial_commitment_events commitment
            WHERE commitment.provider_offer_id=OLD.id
        )
    ) THEN
        RAISE EXCEPTION 'accepted provider commercial terms are immutable'
            USING ERRCODE='23514', CONSTRAINT='provider_offer_commercial_immutability';
    END IF;
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS provider_offer_commercial_immutability_enforced
    ON provider_offers;
CREATE TRIGGER provider_offer_commercial_immutability_enforced
BEFORE UPDATE ON provider_offers
FOR EACH ROW EXECUTE FUNCTION public.enforce_provider_offer_commercial_immutability();
