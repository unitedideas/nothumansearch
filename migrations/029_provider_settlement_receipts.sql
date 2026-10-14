-- Stripe-backed post-action settlement receipts for the terms-only provider
-- pilot. An order is not money: it only freezes the exact CPA already bound
-- to a qualifying NHS outcome. A completed payment receipt is appended only
-- after the existing Stripe-signed webhook path proves a paid Checkout Session.
--
-- Privacy boundary: no agent query, principal, contact, network identifier,
-- or Stripe customer data is stored in these tables. The provider account's
-- existing email is used only when the owner creates the Stripe Checkout
-- Session; it is deliberately not copied into this commercial proof schema.

CREATE TABLE IF NOT EXISTS provider_settlement_orders (
    id                              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    provider_claim_id               UUID NOT NULL,
    provider_offer_id               UUID NOT NULL,
    action_ticket_id                UUID NOT NULL,
    outcome_receipt_id              UUID NOT NULL UNIQUE REFERENCES outcome_receipts(id) ON DELETE RESTRICT,
    outcome                          TEXT NOT NULL CHECK (outcome IN ('accepted', 'activated', 'converted')),
    offer_version_snapshot          INTEGER NOT NULL CHECK (offer_version_snapshot > 0),
    terms_contract_version_snapshot TEXT NOT NULL
                                    CHECK (terms_contract_version_snapshot = 'nhs-provider-commercial-terms-v1'),
    terms_sha256_snapshot           TEXT NOT NULL CHECK (terms_sha256_snapshot ~ '^[0-9a-f]{64}$'),
    amount_cents                    BIGINT NOT NULL CHECK (amount_cents BETWEEN 1 AND 1000000),
    currency                        TEXT NOT NULL CHECK (currency = 'usd'),
    created_at                      TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
    FOREIGN KEY (provider_offer_id, provider_claim_id)
        REFERENCES provider_offers(id, provider_claim_id) ON DELETE RESTRICT,
    FOREIGN KEY (action_ticket_id, provider_offer_id)
        REFERENCES action_tickets(id, provider_offer_id) ON DELETE RESTRICT
);
CREATE INDEX IF NOT EXISTS idx_provider_settlement_orders_claim_created
    ON provider_settlement_orders(provider_claim_id, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_provider_settlement_orders_offer_created
    ON provider_settlement_orders(provider_offer_id, created_at DESC, id DESC);

-- Stripe Checkout creation is a separate immutable fact. The caller passes
-- the order UUID as its Stripe idempotency key, so a retry returns the same
-- session instead of creating another charge opportunity.
CREATE TABLE IF NOT EXISTS provider_settlement_checkout_sessions (
    id                          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    provider_settlement_order_id UUID NOT NULL UNIQUE
                                REFERENCES provider_settlement_orders(id) ON DELETE RESTRICT,
    stripe_checkout_session_id TEXT NOT NULL UNIQUE CHECK (
                                stripe_checkout_session_id ~ '^cs_[A-Za-z0-9]{8,255}$'),
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp()
);
CREATE INDEX IF NOT EXISTS idx_provider_settlement_checkout_created
    ON provider_settlement_checkout_sessions(created_at DESC, id DESC);

-- This is the sole completed-payment fact. It is not derived from a redirect,
-- an internal ledger debit, a provider callback, or a Checkout URL. Its values
-- originate from the Stripe webhook after signature verification.
CREATE TABLE IF NOT EXISTS provider_settlement_payment_receipts (
    id                          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    provider_settlement_order_id UUID NOT NULL UNIQUE
                                REFERENCES provider_settlement_orders(id) ON DELETE RESTRICT,
    stripe_checkout_session_id TEXT NOT NULL UNIQUE CHECK (
                                stripe_checkout_session_id ~ '^cs_[A-Za-z0-9]{8,255}$'),
    stripe_payment_intent_id   TEXT NOT NULL UNIQUE CHECK (
                                stripe_payment_intent_id ~ '^pi_[A-Za-z0-9]{8,255}$'),
    stripe_event_id            TEXT NOT NULL UNIQUE CHECK (
                                stripe_event_id ~ '^evt_[A-Za-z0-9]{8,255}$'),
    amount_cents               BIGINT NOT NULL CHECK (amount_cents BETWEEN 1 AND 1000000),
    currency                   TEXT NOT NULL CHECK (currency = 'usd'),
    paid_at                    TIMESTAMPTZ NOT NULL,
    created_at                 TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp()
);
CREATE INDEX IF NOT EXISTS idx_provider_settlement_payment_paid
    ON provider_settlement_payment_receipts(paid_at DESC, id DESC);

CREATE OR REPLACE FUNCTION enforce_provider_settlement_order()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    actual_claim UUID;
    actual_offer UUID;
    actual_ticket UUID;
    actual_outcome TEXT;
    actual_amount BIGINT;
    actual_currency TEXT;
    actual_version INTEGER;
    actual_contract TEXT;
    actual_terms TEXT;
    actual_billing_mode TEXT;
BEGIN
    SELECT receipt.provider_claim_id, receipt.provider_offer_id,
           receipt.action_ticket_id, receipt.outcome, receipt.billed_cents,
           receipt.currency, ticket.offer_version_snapshot,
           ticket.commercial_terms_contract_version_snapshot,
           ticket.commercial_terms_sha256_snapshot, ticket.billing_mode_snapshot
      INTO actual_claim, actual_offer, actual_ticket, actual_outcome,
           actual_amount, actual_currency, actual_version, actual_contract,
           actual_terms, actual_billing_mode
      FROM outcome_receipts receipt
      JOIN action_tickets ticket ON ticket.id=receipt.action_ticket_id
     WHERE receipt.id=NEW.outcome_receipt_id
       AND receipt.charge_status='charged'
       AND receipt.billed_cents > 0
       AND (
           (receipt.outcome='accepted' AND ticket.status IN ('accepted','activated','converted')) OR
           (receipt.outcome='activated' AND ticket.status IN ('activated','converted')) OR
           (receipt.outcome='converted' AND ticket.status='converted')
       )
       AND ticket.authorization_revoked_at IS NULL
       AND NOT EXISTS (
           SELECT 1 FROM provider_budget_ledger credit
            WHERE credit.action_ticket_id=ticket.id AND credit.entry_type='credit'
       )
     FOR KEY SHARE OF receipt, ticket;
    IF NOT FOUND OR actual_billing_mode <> 'terms' OR
       NEW.provider_claim_id <> actual_claim OR NEW.provider_offer_id <> actual_offer OR
       NEW.action_ticket_id <> actual_ticket OR NEW.outcome <> actual_outcome OR
       NEW.amount_cents <> actual_amount OR NEW.currency <> actual_currency OR
       NEW.offer_version_snapshot <> actual_version OR
       NEW.terms_contract_version_snapshot <> actual_contract OR
       NEW.terms_sha256_snapshot <> actual_terms THEN
        RAISE EXCEPTION 'provider settlement order must exactly bind one current charged terms outcome'
            USING ERRCODE='23514', CONSTRAINT='provider_settlement_order_exact_outcome';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER provider_settlement_order_enforced
BEFORE INSERT ON provider_settlement_orders
FOR EACH ROW EXECUTE FUNCTION enforce_provider_settlement_order();

CREATE OR REPLACE FUNCTION enforce_provider_settlement_checkout_session()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM provider_settlement_orders
         WHERE id=NEW.provider_settlement_order_id
    ) THEN
        RAISE EXCEPTION 'provider settlement checkout must reference an order'
            USING ERRCODE='23514', CONSTRAINT='provider_settlement_checkout_order';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER provider_settlement_checkout_session_enforced
BEFORE INSERT ON provider_settlement_checkout_sessions
FOR EACH ROW EXECUTE FUNCTION enforce_provider_settlement_checkout_session();

CREATE OR REPLACE FUNCTION enforce_provider_settlement_payment_receipt()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    expected_session TEXT;
    expected_amount BIGINT;
    expected_currency TEXT;
    ticket_id UUID;
BEGIN
    SELECT checkout.stripe_checkout_session_id, settlement.amount_cents,
           settlement.currency, settlement.action_ticket_id
      INTO expected_session, expected_amount, expected_currency, ticket_id
      FROM provider_settlement_orders settlement
      JOIN provider_settlement_checkout_sessions checkout
        ON checkout.provider_settlement_order_id=settlement.id
     WHERE settlement.id=NEW.provider_settlement_order_id
     FOR KEY SHARE OF settlement, checkout;
    IF NOT FOUND OR NEW.stripe_checkout_session_id <> expected_session OR
       NEW.amount_cents <> expected_amount OR NEW.currency <> expected_currency OR
       EXISTS (
           SELECT 1 FROM provider_budget_ledger credit
            WHERE credit.action_ticket_id=ticket_id AND credit.entry_type='credit'
       ) THEN
        RAISE EXCEPTION 'provider settlement payment must match an uncredited exact checkout order'
            USING ERRCODE='23514', CONSTRAINT='provider_settlement_payment_exact_checkout';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER provider_settlement_payment_receipt_enforced
BEFORE INSERT ON provider_settlement_payment_receipts
FOR EACH ROW EXECUTE FUNCTION enforce_provider_settlement_payment_receipt();

CREATE OR REPLACE RULE provider_settlement_orders_no_update AS
ON UPDATE TO provider_settlement_orders DO INSTEAD NOTHING;
CREATE OR REPLACE RULE provider_settlement_orders_no_delete AS
ON DELETE TO provider_settlement_orders DO INSTEAD NOTHING;
CREATE OR REPLACE RULE provider_settlement_checkout_sessions_no_update AS
ON UPDATE TO provider_settlement_checkout_sessions DO INSTEAD NOTHING;
CREATE OR REPLACE RULE provider_settlement_checkout_sessions_no_delete AS
ON DELETE TO provider_settlement_checkout_sessions DO INSTEAD NOTHING;
CREATE OR REPLACE RULE provider_settlement_payment_receipts_no_update AS
ON UPDATE TO provider_settlement_payment_receipts DO INSTEAD NOTHING;
CREATE OR REPLACE RULE provider_settlement_payment_receipts_no_delete AS
ON DELETE TO provider_settlement_payment_receipts DO INSTEAD NOTHING;
