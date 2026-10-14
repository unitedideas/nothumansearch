-- Optional, separately consented disclosure of an action ticket's controlled
-- intent to the exact DNS-verified provider after an NHS-observed handoff.
-- Existing handoffs are deliberately backfilled as declined. This migration
-- does not add query, identity, contact, network, or free-form fields.

LOCK TABLE public.action_tickets IN ACCESS EXCLUSIVE MODE;
LOCK TABLE public.provider_action_handoff_receipts IN ACCESS EXCLUSIVE MODE;

ALTER TABLE provider_action_handoff_receipts
    ADD COLUMN IF NOT EXISTS principal_controlled_intent_disclosure_consent
        BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE provider_action_handoff_receipts
    ADD COLUMN IF NOT EXISTS controlled_intent_disclosure_consent_version
        TEXT NOT NULL DEFAULT '';

ALTER TABLE provider_action_handoff_receipts
    DROP CONSTRAINT IF EXISTS provider_handoff_intent_disclosure_consent_pair;
ALTER TABLE provider_action_handoff_receipts
    ADD CONSTRAINT provider_handoff_intent_disclosure_consent_pair CHECK (
        (
            NOT principal_controlled_intent_disclosure_consent AND
            controlled_intent_disclosure_consent_version = ''
        ) OR (
            principal_controlled_intent_disclosure_consent AND
            controlled_intent_disclosure_consent_version =
                'nhs-provider-controlled-intent-disclosure-consent-v1'
        )
    );

-- A disclosure consent must keep authorizing the exact bounded bundle the
-- principal saw at handoff. The resolver reads action_tickets, so direct SQL
-- may not rewrite the returned action type, signed authorization times, or
-- controlled-intent fields. The
-- only allowed change is the existing one-way privacy redaction transition;
-- once redacted, the bundle cannot be restored or changed.
CREATE OR REPLACE FUNCTION public.enforce_action_ticket_controlled_intent_immutability()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF OLD.status IS DISTINCT FROM NEW.status AND NOT (
        (OLD.status = 'created' AND NEW.status IN (
            'redirected', 'rejected', 'duplicate', 'invalid', 'expired'
        )) OR
        (OLD.status = 'redirected' AND NEW.status IN (
            'accepted', 'rejected', 'duplicate', 'invalid', 'expired'
        )) OR
        (OLD.status = 'accepted' AND NEW.status IN (
            'activated', 'duplicate', 'invalid', 'expired'
        )) OR
        (OLD.status = 'activated' AND NEW.status IN (
            'converted', 'duplicate', 'invalid', 'expired'
        )) OR
        (OLD.status = 'converted' AND NEW.status IN ('duplicate', 'invalid'))
    ) THEN
        RAISE EXCEPTION 'action ticket status transition is not monotonic'
            USING ERRCODE='23514',
                  CONSTRAINT='action_ticket_status_transition';
    END IF;

    IF OLD.authorization_revoked_at IS NOT NULL AND
       OLD.authorization_revoked_at IS DISTINCT FROM NEW.authorization_revoked_at THEN
        RAISE EXCEPTION 'action ticket authorization revocation is one-way and immutable'
            USING ERRCODE='23514',
                  CONSTRAINT='action_ticket_controlled_intent_immutable';
    END IF;

    IF ROW(
        OLD.offer_version_snapshot,
        OLD.commercial_terms_contract_version_snapshot,
        OLD.commercial_terms_sha256_snapshot,
        OLD.action_type_snapshot, OLD.created_at, OLD.expires_at
    ) IS DISTINCT FROM ROW(
        NEW.offer_version_snapshot,
        NEW.commercial_terms_contract_version_snapshot,
        NEW.commercial_terms_sha256_snapshot,
        NEW.action_type_snapshot, NEW.created_at, NEW.expires_at
    ) THEN
        RAISE EXCEPTION 'action ticket disclosed offer binding, action type, and authorization times are immutable'
            USING ERRCODE='23514',
                  CONSTRAINT='action_ticket_controlled_intent_immutable';
    END IF;

    IF OLD.intent_redacted_at IS NULL AND NEW.intent_redacted_at IS NULL THEN
        IF ROW(
            OLD.demand_topic, OLD.region_code, OLD.budget_band,
            OLD.urgency, OLD.requirement_flags
        ) IS DISTINCT FROM ROW(
            NEW.demand_topic, NEW.region_code, NEW.budget_band,
            NEW.urgency, NEW.requirement_flags
        ) THEN
            RAISE EXCEPTION 'action ticket controlled intent is immutable before redaction'
                USING ERRCODE='23514',
                      CONSTRAINT='action_ticket_controlled_intent_immutable';
        END IF;
        RETURN NEW;
    END IF;

    IF OLD.intent_redacted_at IS NULL AND NEW.intent_redacted_at IS NOT NULL THEN
        IF NEW.demand_topic = 'redacted' AND
           NEW.region_code = '' AND
           NEW.budget_band = 'unspecified' AND
           NEW.urgency = 'unspecified' AND
           cardinality(NEW.requirement_flags) = 0 THEN
            RETURN NEW;
        END IF;
        RAISE EXCEPTION 'action ticket intent redaction must be complete and one-way'
            USING ERRCODE='23514',
                  CONSTRAINT='action_ticket_controlled_intent_immutable';
    END IF;

    IF ROW(
        OLD.demand_topic, OLD.region_code, OLD.budget_band,
        OLD.urgency, OLD.requirement_flags, OLD.intent_redacted_at
    ) IS DISTINCT FROM ROW(
        NEW.demand_topic, NEW.region_code, NEW.budget_band,
        NEW.urgency, NEW.requirement_flags, NEW.intent_redacted_at
    ) THEN
        RAISE EXCEPTION 'redacted action ticket controlled intent cannot be restored or changed'
            USING ERRCODE='23514',
                  CONSTRAINT='action_ticket_controlled_intent_immutable';
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS action_ticket_controlled_intent_immutability_enforced
    ON public.action_tickets;
CREATE TRIGGER action_ticket_controlled_intent_immutability_enforced
BEFORE UPDATE OF offer_version_snapshot,
    commercial_terms_contract_version_snapshot,
    commercial_terms_sha256_snapshot, action_type_snapshot, created_at,
    expires_at, demand_topic, region_code, budget_band, urgency,
    requirement_flags, intent_redacted_at, authorization_revoked_at, status
ON public.action_tickets
FOR EACH ROW
EXECUTE FUNCTION public.enforce_action_ticket_controlled_intent_immutability();
