-- Exact processor fee/net evidence for provider settlement payments.
--
-- A signed paid Checkout receipt proves collection but not retained value.
-- This migration binds the corresponding Stripe charge balance transaction's
-- gross, fee, net, currency, and availability schedule to that payment. A
-- separate append-only availability receipt is required before the net amount
-- can enter mechanism-selection proof.
--
-- Privacy boundary: Stripe object identifiers remain private database binding
-- coordinates. No query, prompt, search receipt, agent/principal identity,
-- contact, network data, customer data, or payment method is stored here.

CREATE TABLE IF NOT EXISTS provider_settlement_processor_balance_receipts (
    id                            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    provider_settlement_payment_receipt_id UUID NOT NULL UNIQUE
                                  REFERENCES provider_settlement_payment_receipts(id) ON DELETE RESTRICT,
    stripe_charge_id              TEXT NOT NULL UNIQUE CHECK (
                                  stripe_charge_id ~ '^ch_[A-Za-z0-9]{8,255}$'),
    stripe_balance_transaction_id TEXT NOT NULL UNIQUE CHECK (
                                  stripe_balance_transaction_id ~ '^txn_[A-Za-z0-9]{8,255}$'),
    gross_amount_cents            BIGINT NOT NULL CHECK (gross_amount_cents BETWEEN 1 AND 1000000),
    fee_cents                     BIGINT NOT NULL CHECK (fee_cents BETWEEN 0 AND 1000000),
    net_cents                     BIGINT NOT NULL CHECK (net_cents BETWEEN 1 AND 1000000),
    currency                      TEXT NOT NULL CHECK (currency = 'usd'),
    initial_status                TEXT NOT NULL CHECK (initial_status IN ('pending','available')),
    available_on                  TIMESTAMPTZ NOT NULL,
    processor_observed_at         TIMESTAMPTZ NOT NULL,
    created_at                    TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
    CHECK (gross_amount_cents - fee_cents = net_cents),
    CHECK (processor_observed_at <= created_at)
);
CREATE INDEX IF NOT EXISTS idx_provider_settlement_processor_balance_created
    ON provider_settlement_processor_balance_receipts(created_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS provider_settlement_processor_availability_receipts (
    id                            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    processor_balance_receipt_id  UUID NOT NULL UNIQUE
                                  REFERENCES provider_settlement_processor_balance_receipts(id) ON DELETE RESTRICT,
    gross_amount_cents            BIGINT NOT NULL CHECK (gross_amount_cents BETWEEN 1 AND 1000000),
    fee_cents                     BIGINT NOT NULL CHECK (fee_cents BETWEEN 0 AND 1000000),
    net_cents                     BIGINT NOT NULL CHECK (net_cents BETWEEN 1 AND 1000000),
    currency                      TEXT NOT NULL CHECK (currency = 'usd'),
    available_on                  TIMESTAMPTZ NOT NULL,
    processor_verified_at         TIMESTAMPTZ NOT NULL,
    created_at                    TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
    CHECK (gross_amount_cents - fee_cents = net_cents),
    CHECK (processor_verified_at <= created_at),
    CHECK (processor_verified_at >= available_on)
);
CREATE INDEX IF NOT EXISTS idx_provider_settlement_processor_available_created
    ON provider_settlement_processor_availability_receipts(created_at DESC, id DESC);

CREATE OR REPLACE FUNCTION enforce_provider_settlement_processor_balance_receipt()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    expected_amount BIGINT;
    expected_currency TEXT;
    expected_paid_at TIMESTAMPTZ;
BEGIN
    SELECT amount_cents, currency, paid_at
      INTO expected_amount, expected_currency, expected_paid_at
      FROM provider_settlement_payment_receipts
     WHERE id=NEW.provider_settlement_payment_receipt_id
     FOR KEY SHARE;
    IF NOT FOUND OR NEW.gross_amount_cents <> expected_amount OR
       NEW.currency <> expected_currency OR
       NEW.processor_observed_at < expected_paid_at THEN
        RAISE EXCEPTION 'processor balance receipt must exactly bind one paid settlement'
            USING ERRCODE='23514', CONSTRAINT='provider_settlement_processor_balance_exact_payment';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER provider_settlement_processor_balance_receipt_enforced
BEFORE INSERT ON provider_settlement_processor_balance_receipts
FOR EACH ROW EXECUTE FUNCTION enforce_provider_settlement_processor_balance_receipt();

CREATE OR REPLACE FUNCTION enforce_provider_settlement_processor_availability_receipt()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    expected_gross BIGINT;
    expected_fee BIGINT;
    expected_net BIGINT;
    expected_currency TEXT;
    expected_available_on TIMESTAMPTZ;
BEGIN
    SELECT gross_amount_cents, fee_cents, net_cents, currency, available_on
      INTO expected_gross, expected_fee, expected_net, expected_currency, expected_available_on
      FROM provider_settlement_processor_balance_receipts
     WHERE id=NEW.processor_balance_receipt_id
     FOR KEY SHARE;
    IF NOT FOUND OR NEW.gross_amount_cents <> expected_gross OR
       NEW.fee_cents <> expected_fee OR NEW.net_cents <> expected_net OR
       NEW.currency <> expected_currency OR NEW.available_on <> expected_available_on THEN
        RAISE EXCEPTION 'processor availability receipt must exactly bind one balance receipt'
            USING ERRCODE='23514', CONSTRAINT='provider_settlement_processor_availability_exact_balance';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER provider_settlement_processor_availability_receipt_enforced
BEFORE INSERT ON provider_settlement_processor_availability_receipts
FOR EACH ROW EXECUTE FUNCTION enforce_provider_settlement_processor_availability_receipt();

CREATE OR REPLACE RULE provider_settlement_processor_balance_receipts_no_update AS
ON UPDATE TO provider_settlement_processor_balance_receipts DO INSTEAD NOTHING;
CREATE OR REPLACE RULE provider_settlement_processor_balance_receipts_no_delete AS
ON DELETE TO provider_settlement_processor_balance_receipts DO INSTEAD NOTHING;
CREATE OR REPLACE RULE provider_settlement_processor_availability_receipts_no_update AS
ON UPDATE TO provider_settlement_processor_availability_receipts DO INSTEAD NOTHING;
CREATE OR REPLACE RULE provider_settlement_processor_availability_receipts_no_delete AS
ON DELETE TO provider_settlement_processor_availability_receipts DO INSTEAD NOTHING;
