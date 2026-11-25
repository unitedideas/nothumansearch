-- Exact-pilot proof integrity.
--
-- PostgreSQL cannot authenticate the NHS outcome MAC because the signing
-- material deliberately lives outside the database. The application therefore
-- re-verifies every qualifying receipt before it can affect 3/5/2/1 proof.
-- This migration closes the complementary relational and clock boundaries:
-- outcome rows must describe the exact pilot ticket/handoff and their canonical
-- JSON must agree with the row, while lifecycle state cannot commit without its
-- append-only canonical event.

-- Constraint triggers below protect future writes. Freeze the three lifecycle
-- relations while proving that every 024-era row already has its required
-- append-only event; otherwise a pre-026 gap would be grandfathered forever.
LOCK TABLE public.provider_pilot_epochs,
           public.provider_pilot_enrollments,
           public.provider_pilot_epoch_events
    IN SHARE ROW EXCLUSIVE MODE;
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM public.provider_pilot_epochs epoch
         WHERE NOT EXISTS (
                   SELECT 1 FROM public.provider_pilot_epoch_events event
                    WHERE event.provider_pilot_epoch_id=epoch.id
                      AND event.event_type='created'
               )
            OR (epoch.status IN ('active','closed') AND NOT EXISTS (
                   SELECT 1 FROM public.provider_pilot_epoch_events event
                    WHERE event.provider_pilot_epoch_id=epoch.id
                      AND event.event_type='activated'
               ))
            OR (epoch.status='closed' AND NOT EXISTS (
                   SELECT 1 FROM public.provider_pilot_epoch_events event
                    WHERE event.provider_pilot_epoch_id=epoch.id
                      AND event.event_type='closed'
               ))
    ) OR EXISTS (
        SELECT 1
          FROM public.provider_pilot_enrollments enrollment
         WHERE NOT EXISTS (
             SELECT 1 FROM public.provider_pilot_epoch_events event
              WHERE event.provider_pilot_epoch_id=enrollment.provider_pilot_epoch_id
                AND event.provider_claim_id=enrollment.provider_claim_id
                AND event.event_type='provider_enrolled'
         )
    ) THEN
        RAISE EXCEPTION 'migration 026 requires complete preexisting provider pilot lifecycle events'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_pilot_lifecycle_legacy_incomplete';
    END IF;
END;
$$;

CREATE OR REPLACE FUNCTION public.enforce_provider_pilot_outcome_receipt()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    ticket_claim UUID;
    ticket_offer UUID;
    ticket_pilot UUID;
    ticket_created_at TIMESTAMPTZ;
    pilot_status TEXT;
    pilot_activated_at TIMESTAMPTZ;
    pilot_closed_at TIMESTAMPTZ;
    handoff_observed_at TIMESTAMPTZ;
    canonical JSONB;
    canonical_key_count INTEGER;
    recorded_epoch TEXT;
    expires_epoch TEXT;
BEGIN
    SELECT ticket.provider_claim_id, ticket.provider_offer_id,
           ticket.provider_pilot_epoch_id, ticket.created_at
      INTO ticket_claim, ticket_offer, ticket_pilot, ticket_created_at
      FROM action_tickets ticket
     WHERE ticket.id=NEW.action_ticket_id
     FOR KEY SHARE;
    IF NOT FOUND OR
       ticket_claim IS DISTINCT FROM NEW.provider_claim_id OR
       ticket_offer IS DISTINCT FROM NEW.provider_offer_id THEN
        RAISE EXCEPTION 'outcome receipt requires its exact ticket'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_pilot_outcome_exact_ticket';
    END IF;

    -- Migration 024 deliberately leaves historical tickets with a NULL pilot
    -- binding. Their late invalid/duplicate credit path must remain usable;
    -- they are excluded from pilot CTEs and cannot affect 3/5/2/1 proof. The
    -- stronger relational/clock/canonical checks below apply only to tickets
    -- explicitly bound to a protected pilot epoch.
    IF ticket_pilot IS NULL THEN
        RETURN NEW;
    END IF;

    SELECT pilot.status, pilot.activated_at, pilot.closed_at
      INTO pilot_status, pilot_activated_at, pilot_closed_at
      FROM provider_pilot_epochs pilot
     WHERE pilot.id=ticket_pilot
     FOR KEY SHARE;
    IF NOT FOUND OR
       pilot_status NOT IN ('active','closed') OR
       pilot_activated_at IS NULL OR
       ticket_created_at < date_trunc('second', pilot_activated_at) OR
       (pilot_closed_at IS NOT NULL AND ticket_created_at > pilot_closed_at) THEN
        RAISE EXCEPTION 'outcome receipt requires the exact in-window pilot ticket'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_pilot_outcome_exact_ticket';
    END IF;

    IF NEW.outcome IN ('accepted','activated','converted') THEN
        PERFORM 1
          FROM provider_pilot_enrollments enrollment
          JOIN provider_claims claim ON claim.id=enrollment.provider_claim_id
          JOIN sites site ON site.id=claim.site_id
         WHERE enrollment.provider_pilot_epoch_id=ticket_pilot
           AND enrollment.provider_claim_id=ticket_claim
         FOR KEY SHARE OF claim, site;
        IF NOT FOUND OR NOT public.provider_pilot_enrollment_eligibility_is_current(
            ticket_pilot, ticket_claim
        ) THEN
            RAISE EXCEPTION 'positive outcome receipt requires the current Stage 1 eligibility binding'
                USING ERRCODE='23514',
                      CONSTRAINT='provider_pilot_outcome_enrollment_eligibility';
        END IF;

        SELECT handoff.observed_at
          INTO handoff_observed_at
          FROM provider_action_handoff_receipts handoff
         WHERE handoff.action_ticket_id=NEW.action_ticket_id
           AND handoff.provider_claim_id=NEW.provider_claim_id
           AND handoff.provider_offer_id=NEW.provider_offer_id
         FOR KEY SHARE;
        IF NOT FOUND OR
           handoff_observed_at < pilot_activated_at OR
           (pilot_closed_at IS NOT NULL AND handoff_observed_at > pilot_closed_at) THEN
            RAISE EXCEPTION 'positive outcome receipt requires the exact in-window pilot handoff'
                USING ERRCODE='23514',
                      CONSTRAINT='provider_pilot_outcome_exact_handoff';
        END IF;
    END IF;

    -- Production obtains this value from the database clock after the ticket
    -- and offer locks are held. Whole-second equality is part of the signed
    -- v1 receipt contract. The bounded database-clock check prevents a caller
    -- from supplying a historical pilot timestamp while allowing a short,
    -- lock-held transaction to finish signing and accounting atomically.
    IF NEW.created_at IS DISTINCT FROM NEW.provider_reported_at OR
       NEW.created_at IS DISTINCT FROM date_trunc('second', NEW.created_at) OR
       (handoff_observed_at IS NOT NULL AND
        NEW.created_at < date_trunc('second', handoff_observed_at)) OR
       NEW.created_at < statement_timestamp() - INTERVAL '5 minutes' OR
       NEW.created_at > clock_timestamp() + INTERVAL '1 second' THEN
        RAISE EXCEPTION 'outcome receipt timestamp is not a current database-clock fact'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_pilot_outcome_database_clock';
    END IF;

    BEGIN
        canonical := NEW.signed_receipt::jsonb;
    EXCEPTION WHEN OTHERS THEN
        RAISE EXCEPTION 'outcome receipt canonical payload is not JSON'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_pilot_outcome_canonical_row';
    END;
    IF jsonb_typeof(canonical) IS DISTINCT FROM 'object' THEN
        RAISE EXCEPTION 'outcome receipt canonical payload must be an object'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_pilot_outcome_canonical_row';
    END IF;
    SELECT COUNT(*)::integer
      INTO canonical_key_count
      FROM jsonb_object_keys(canonical);
    recorded_epoch := EXTRACT(EPOCH FROM NEW.created_at)::bigint::text;
    expires_epoch := EXTRACT(
        EPOCH FROM NEW.created_at + INTERVAL '315360000 seconds'
    )::bigint::text;
    IF canonical_key_count <> 13 OR
       canonical->>'v' IS DISTINCT FROM '1' OR
       COALESCE(canonical->>'kid','') !~ '^[A-Za-z0-9][A-Za-z0-9._-]{2,63}$' OR
       canonical->>'receipt_id' IS DISTINCT FROM NEW.id::text OR
       canonical->>'ticket_id' IS DISTINCT FROM NEW.action_ticket_id::text OR
       canonical->>'offer_id' IS DISTINCT FROM NEW.provider_offer_id::text OR
       canonical->>'nhs_event_id' IS DISTINCT FROM NEW.nhs_event_id::text OR
       canonical->>'outcome' IS DISTINCT FROM NEW.outcome OR
       canonical->>'provider_reported_at' IS DISTINCT FROM recorded_epoch OR
       canonical->>'recorded_at' IS DISTINCT FROM recorded_epoch OR
       canonical->>'expires_at' IS DISTINCT FROM expires_epoch OR
       canonical->>'charged_minor' IS DISTINCT FROM NEW.billed_cents::text OR
       canonical->>'currency' IS DISTINCT FROM NEW.currency OR
       canonical->>'charge_status' IS DISTINCT FROM NEW.charge_status THEN
        RAISE EXCEPTION 'outcome receipt canonical payload does not match its row'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_pilot_outcome_canonical_row';
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS provider_pilot_outcome_receipt_enforced
    ON public.outcome_receipts;
CREATE TRIGGER provider_pilot_outcome_receipt_enforced
BEFORE INSERT ON public.outcome_receipts
FOR EACH ROW
EXECUTE FUNCTION public.enforce_provider_pilot_outcome_receipt();

CREATE OR REPLACE FUNCTION public.require_provider_pilot_lifecycle_event()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    required_event_type TEXT;
    required_epoch_id UUID;
    required_claim_id UUID;
    required_constraint TEXT;
BEGIN
    IF TG_TABLE_NAME='provider_pilot_enrollments' THEN
        required_event_type := 'provider_enrolled';
        required_epoch_id := NEW.provider_pilot_epoch_id;
        required_claim_id := NEW.provider_claim_id;
        required_constraint := 'provider_pilot_enrollment_event_required';
    ELSIF TG_OP='INSERT' THEN
        required_event_type := 'created';
        required_epoch_id := NEW.id;
        required_claim_id := NULL;
        required_constraint := 'provider_pilot_created_event_required';
    ELSIF NEW.status='active' THEN
        required_event_type := 'activated';
        required_epoch_id := NEW.id;
        required_claim_id := NULL;
        required_constraint := 'provider_pilot_activated_event_required';
    ELSIF NEW.status='closed' THEN
        required_event_type := 'closed';
        required_epoch_id := NEW.id;
        required_claim_id := NULL;
        required_constraint := 'provider_pilot_closed_event_required';
    ELSE
        RETURN NEW;
    END IF;

    IF NOT EXISTS (
        SELECT 1
          FROM provider_pilot_epoch_events event
         WHERE event.provider_pilot_epoch_id=required_epoch_id
           AND event.event_type=required_event_type
           AND event.provider_claim_id IS NOT DISTINCT FROM required_claim_id
    ) THEN
        RAISE EXCEPTION 'provider pilot lifecycle state requires its canonical event'
            USING ERRCODE='23514', CONSTRAINT=required_constraint;
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS provider_pilot_epoch_created_event_required
    ON public.provider_pilot_epochs;
CREATE CONSTRAINT TRIGGER provider_pilot_epoch_created_event_required
AFTER INSERT ON public.provider_pilot_epochs
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW
EXECUTE FUNCTION public.require_provider_pilot_lifecycle_event();

DROP TRIGGER IF EXISTS provider_pilot_enrollment_event_required
    ON public.provider_pilot_enrollments;
CREATE CONSTRAINT TRIGGER provider_pilot_enrollment_event_required
AFTER INSERT ON public.provider_pilot_enrollments
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW
EXECUTE FUNCTION public.require_provider_pilot_lifecycle_event();

DROP TRIGGER IF EXISTS provider_pilot_epoch_transition_event_required
    ON public.provider_pilot_epochs;
CREATE CONSTRAINT TRIGGER provider_pilot_epoch_transition_event_required
AFTER UPDATE ON public.provider_pilot_epochs
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW
WHEN (OLD.status IS DISTINCT FROM NEW.status)
EXECUTE FUNCTION public.require_provider_pilot_lifecycle_event();
