-- Append-only capacity reservations for provider-funded action tickets.
--
-- A budget/credit check without a reservation lets many tickets promise the
-- same bounty before any provider callback moves the ledger. These events make
-- the promise explicit: reserve on ticket creation, consume on the configured
-- charge event, and release when an uncharged ticket reaches a terminal
-- provider outcome. Expired or emergency-revoked tickets are also excluded
-- logically from active capacity so a delayed cleanup cannot strand budget.
--
-- This table contains only commercial IDs and bounded money. It deliberately
-- has no query, contact, agent, principal, network, or free-form payload field.
-- Take the table lock before the backfill snapshot. A pre-migration writer that
-- already holds a RowExclusiveLock must commit before this migration can take
-- its snapshot, and new legacy writers remain blocked until the deferred
-- compatibility trigger is installed and the migration commits.
LOCK TABLE public.action_tickets IN SHARE ROW EXCLUSIVE MODE;

CREATE TABLE IF NOT EXISTS provider_capacity_events (
    id                    BIGSERIAL PRIMARY KEY,
    provider_claim_id     UUID NOT NULL,
    provider_offer_id     UUID NOT NULL,
    action_ticket_id      UUID NOT NULL,
    event_type            TEXT NOT NULL
                          CHECK (event_type IN ('reserve', 'consume', 'release')),
    event_reason          TEXT NOT NULL CHECK (
                              (event_type = 'reserve' AND event_reason = 'ticket_created') OR
                              (event_type = 'consume' AND event_reason = 'charge_recorded') OR
                              (event_type = 'release' AND event_reason = 'terminal_without_charge')
                          ),
    amount_cents          BIGINT NOT NULL CHECK (amount_cents BETWEEN 1 AND 1000000),
    currency              TEXT NOT NULL CHECK (currency = 'usd'),
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (provider_offer_id, provider_claim_id)
        REFERENCES provider_offers(id, provider_claim_id) ON DELETE RESTRICT,
    FOREIGN KEY (action_ticket_id, provider_offer_id)
        REFERENCES action_tickets(id, provider_offer_id) ON DELETE RESTRICT,
    UNIQUE (action_ticket_id, event_type)
);
CREATE INDEX IF NOT EXISTS idx_provider_capacity_events_offer_created
    ON provider_capacity_events(provider_offer_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_provider_capacity_events_ticket_created
    ON provider_capacity_events(action_ticket_id, created_at, id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_provider_capacity_one_terminal_per_ticket
    ON provider_capacity_events(action_ticket_id)
    WHERE event_type IN ('consume', 'release');
CREATE OR REPLACE RULE provider_capacity_events_no_update AS
ON UPDATE TO provider_capacity_events DO INSTEAD NOTHING;
CREATE OR REPLACE RULE provider_capacity_events_no_delete AS
ON DELETE TO provider_capacity_events DO INSTEAD NOTHING;

-- Backfill the exact historical ticket promise before new traffic can run.
-- Oversubscribed legacy tickets remain visible as reservations; this fails new
-- ticket creation closed instead of pretending the old promises did not exist.
INSERT INTO provider_capacity_events (
    provider_claim_id, provider_offer_id, action_ticket_id,
    event_type, event_reason, amount_cents, currency, created_at
)
SELECT ticket.provider_claim_id, ticket.provider_offer_id, ticket.id,
       'reserve', 'ticket_created', ticket.bounty_cents_snapshot,
       ticket.currency_snapshot, ticket.created_at
FROM action_tickets ticket;

INSERT INTO provider_capacity_events (
    provider_claim_id, provider_offer_id, action_ticket_id,
    event_type, event_reason, amount_cents, currency, created_at
)
SELECT ticket.provider_claim_id, ticket.provider_offer_id, ticket.id,
       'consume', 'charge_recorded', -charge.amount_cents,
       charge.currency, charge.created_at
FROM action_tickets ticket
JOIN provider_budget_ledger charge
  ON charge.action_ticket_id=ticket.id AND charge.entry_type='charge';

INSERT INTO provider_capacity_events (
    provider_claim_id, provider_offer_id, action_ticket_id,
    event_type, event_reason, amount_cents, currency, created_at
)
SELECT ticket.provider_claim_id, ticket.provider_offer_id, ticket.id,
       'release', 'terminal_without_charge', ticket.bounty_cents_snapshot,
       ticket.currency_snapshot, ticket.updated_at
FROM action_tickets ticket
WHERE ticket.status IN ('rejected', 'duplicate', 'invalid', 'expired')
  AND NOT EXISTS (
      SELECT 1 FROM provider_budget_ledger charge
      WHERE charge.action_ticket_id=ticket.id AND charge.entry_type='charge'
  );

-- Rolling-deploy compatibility boundary. New application code writes the
-- action ticket and reserve event in one transaction. An old binary that only
-- writes the ticket is allowed to finish the INSERT, but fails closed when the
-- transaction checks this deferred constraint at commit.
CREATE OR REPLACE FUNCTION public.enforce_action_ticket_capacity_reservation()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM public.provider_capacity_events capacity
        WHERE capacity.action_ticket_id = NEW.id
          AND capacity.provider_claim_id = NEW.provider_claim_id
          AND capacity.provider_offer_id = NEW.provider_offer_id
          AND capacity.event_type = 'reserve'
          AND capacity.event_reason = 'ticket_created'
          AND capacity.amount_cents = NEW.bounty_cents_snapshot
          AND capacity.currency = NEW.currency_snapshot
    ) THEN
        RAISE EXCEPTION 'action ticket capacity reservation required'
            USING ERRCODE = '23514',
                  CONSTRAINT = 'action_ticket_capacity_reservation_required';
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS action_ticket_capacity_reservation_required
    ON public.action_tickets;
CREATE CONSTRAINT TRIGGER action_ticket_capacity_reservation_required
AFTER INSERT OR UPDATE ON public.action_tickets
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW
EXECUTE FUNCTION public.enforce_action_ticket_capacity_reservation();
