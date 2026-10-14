-- Immutable owner-review evidence for the bounded provider pilot.
--
-- The commercial-proof gate requires an owner to inspect every qualifying
-- provider, offer, ticket, handoff, and callback in the exact pilot. A note
-- saying that review happened is not enough: each review must bind the exact
-- database snapshot that was inspected. This table stores only opaque
-- relational IDs, a SHA-256 snapshot digest, bounded non-secret evidence
-- references, and a database timestamp. It never stores a query, search
-- receipt, bearer/token hash, company-deduplication hash, principal or agent
-- identity/contact/network metadata, or free-form intent.

CREATE TABLE IF NOT EXISTS provider_pilot_review_events (
    id                         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    provider_pilot_epoch_id    UUID NOT NULL
                               REFERENCES provider_pilot_epochs(id) ON DELETE RESTRICT,
    review_contract_version    TEXT NOT NULL CHECK (
                                   review_contract_version =
                                   'nhs-provider-pilot-review-v1'
                               ),
    review_type                TEXT NOT NULL CHECK (review_type IN (
                                   'provider','offer','ticket','handoff','callback'
                               )),
    subject_id                 UUID NOT NULL,
    provider_claim_id          UUID NOT NULL
                               REFERENCES provider_claims(id) ON DELETE RESTRICT,
    provider_offer_id          UUID,
    action_ticket_id           UUID,
    handoff_receipt_id         UUID
                               REFERENCES provider_action_handoff_receipts(id)
                               ON DELETE RESTRICT,
    outcome_receipt_id         UUID
                               REFERENCES outcome_receipts(id) ON DELETE RESTRICT,
    subject_snapshot_sha256    TEXT NOT NULL CHECK (
                                   subject_snapshot_sha256 ~ '^[0-9a-f]{64}$'
                               ),
    owner_reference            TEXT NOT NULL CHECK (
                                   owner_reference ~
                                   '^[A-Za-z0-9][A-Za-z0-9._:/-]{2,199}$'
                               ),
    evidence_reference         TEXT NOT NULL CHECK (
                                   evidence_reference ~
                                   '^[A-Za-z0-9][A-Za-z0-9._:/-]{2,199}$'
                               ),
    reviewed_at                TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
    created_at                 TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
    UNIQUE (provider_pilot_epoch_id, review_type, subject_id),
    FOREIGN KEY (provider_offer_id, provider_claim_id)
        REFERENCES provider_offers(id, provider_claim_id) ON DELETE RESTRICT,
    FOREIGN KEY (action_ticket_id, provider_offer_id)
        REFERENCES action_tickets(id, provider_offer_id) ON DELETE RESTRICT,
    CHECK (reviewed_at = created_at),
    CHECK (
        (review_type='provider'
            AND subject_id=provider_claim_id
            AND provider_offer_id IS NULL
            AND action_ticket_id IS NULL
            AND handoff_receipt_id IS NULL
            AND outcome_receipt_id IS NULL) OR
        (review_type='offer'
            AND subject_id=provider_offer_id
            AND provider_offer_id IS NOT NULL
            AND action_ticket_id IS NULL
            AND handoff_receipt_id IS NULL
            AND outcome_receipt_id IS NULL) OR
        (review_type='ticket'
            AND subject_id=action_ticket_id
            AND provider_offer_id IS NOT NULL
            AND action_ticket_id IS NOT NULL
            AND handoff_receipt_id IS NULL
            AND outcome_receipt_id IS NULL) OR
        (review_type='handoff'
            AND subject_id=handoff_receipt_id
            AND provider_offer_id IS NOT NULL
            AND action_ticket_id IS NOT NULL
            AND handoff_receipt_id IS NOT NULL
            AND outcome_receipt_id IS NULL) OR
        (review_type='callback'
            AND subject_id=outcome_receipt_id
            AND provider_offer_id IS NOT NULL
            AND action_ticket_id IS NOT NULL
            AND handoff_receipt_id IS NOT NULL
            AND outcome_receipt_id IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_provider_pilot_reviews_pilot_type
    ON provider_pilot_review_events(
        provider_pilot_epoch_id, review_type, reviewed_at, id
    );
CREATE INDEX IF NOT EXISTS idx_provider_pilot_reviews_claim_type
    ON provider_pilot_review_events(
        provider_pilot_epoch_id, provider_claim_id, review_type, reviewed_at, id
    );

-- The digest input uses fixed labels, canonical PostgreSQL UUID/text forms,
-- exact microsecond timestamps, controlled enums, and immutable/bounded row
-- fields. Low-entropy owner identity material and bearer-derived values are
-- deliberately absent. The subject UUID still gives every digest a
-- high-entropy namespace.
CREATE OR REPLACE FUNCTION public.provider_pilot_review_snapshot_sha256(
    requested_pilot_id UUID,
    requested_review_type TEXT,
    requested_subject_id UUID
)
RETURNS TEXT
LANGUAGE plpgsql
STABLE
STRICT
AS $$
DECLARE
    snapshot_preimage TEXT;
BEGIN
    CASE requested_review_type
    WHEN 'provider' THEN
        SELECT array_to_string(ARRAY[
            'nhs-provider-pilot-review-snapshot-v1', 'provider',
            epoch.id::text, epoch.contract_version, epoch.demand_topic,
            enrollment.id::text, company.id::text, claim.id::text,
            claim.domain_snapshot, enrollment.stage1_eligibility_snapshot_sha256,
            accepted.id::text,
            accepted.provider_acceptance_reference,
            ((EXTRACT(EPOCH FROM accepted.provider_accepted_at) * 1000000)::bigint)::text,
            ((EXTRACT(EPOCH FROM company.owner_verified_at) * 1000000)::bigint)::text,
            ((EXTRACT(EPOCH FROM enrollment.enrolled_at) * 1000000)::bigint)::text
        ], E'\n', '<null>')
          INTO snapshot_preimage
          FROM provider_pilot_epochs epoch
          JOIN provider_pilot_enrollments enrollment
            ON enrollment.provider_pilot_epoch_id=epoch.id
           AND enrollment.provider_claim_id=requested_subject_id
          JOIN provider_pilot_companies company
            ON company.id=enrollment.provider_pilot_company_id
           AND company.provider_claim_id=enrollment.provider_claim_id
          JOIN provider_claims claim
            ON claim.id=enrollment.provider_claim_id
          JOIN provider_commercial_acceptance_events accepted
            ON accepted.id=company.provider_acceptance_event_id
           AND accepted.provider_claim_id=company.provider_claim_id
           AND accepted.event_type='pilot_company'
         WHERE epoch.id=requested_pilot_id
           AND claim.status='verified'
           AND claim.verification_last_succeeded_at >
               statement_timestamp()-INTERVAL '7 days'
           AND public.provider_pilot_enrollment_eligibility_is_current(
               epoch.id, claim.id
           );

    WHEN 'offer' THEN
        SELECT array_to_string(ARRAY[
            'nhs-provider-pilot-review-snapshot-v1', 'offer',
            epoch.id::text, epoch.contract_version, epoch.demand_topic,
            company.id::text, offer.provider_claim_id::text, offer.id::text,
            offer.version::text, offer.offer_name, offer.offer_summary,
            offer.action_type, offer.action_url, offer.disclosure_label,
            offer.charge_event, offer.bounty_cents::text, offer.currency,
            offer.principal_price_mode,
            COALESCE(offer.principal_price_cents::text, '<null>'),
            offer.principal_currency, offer.billing_mode,
            COALESCE(offer.terms_credit_limit_cents::text, '<null>'),
            COALESCE(offer.terms_period_days::text, '<null>'),
            offer.commercial_terms_contract_version,
            offer.commercial_terms_sha256, commitment.id::text,
            commitment.event_type,
            commitment.provider_acceptance_event_id::text,
            ((EXTRACT(EPOCH FROM commitment.provider_accepted_at) * 1000000)::bigint)::text,
            ((EXTRACT(EPOCH FROM commitment.valid_until) * 1000000)::bigint)::text,
            ((EXTRACT(EPOCH FROM commitment.owner_verified_at) * 1000000)::bigint)::text
        ], E'\n', '<null>')
          INTO snapshot_preimage
          FROM provider_pilot_epochs epoch
          JOIN provider_pilot_enrollments enrollment
            ON enrollment.provider_pilot_epoch_id=epoch.id
          JOIN provider_pilot_companies company
            ON company.id=enrollment.provider_pilot_company_id
           AND company.provider_claim_id=enrollment.provider_claim_id
          JOIN provider_claims claim
            ON claim.id=enrollment.provider_claim_id
          JOIN provider_offers offer
            ON offer.id=requested_subject_id
           AND offer.provider_claim_id=enrollment.provider_claim_id
           AND (offer.provider_pilot_epoch_id IS NULL OR
                offer.provider_pilot_epoch_id=epoch.id)
          JOIN LATERAL (
              SELECT commitment.*
                FROM provider_commercial_commitment_events commitment
               WHERE commitment.provider_pilot_company_id=company.id
                 AND commitment.provider_claim_id=offer.provider_claim_id
                 AND commitment.provider_offer_id=offer.id
                 AND commitment.offer_version_snapshot=offer.version
                 AND commitment.terms_contract_version=
                     offer.commercial_terms_contract_version
                 AND commitment.exact_terms_sha256=offer.commercial_terms_sha256
                 AND (
                     (offer.billing_mode='prepaid'
                      AND commitment.event_type='prepaid_fund'
                      AND commitment.amount_cents + COALESCE((
                          SELECT SUM(reversal.amount_cents)
                            FROM provider_commercial_commitment_events reversal
                           WHERE reversal.related_event_id=commitment.id
                             AND reversal.event_type='fund_reversal'
                      ),0) > 0) OR
                     (offer.billing_mode='terms'
                      AND commitment.event_type IN ('terms_acceptance','terms_renewal')
                      AND commitment.valid_until > statement_timestamp())
                 )
               ORDER BY commitment.owner_verified_at ASC,
                        commitment.provider_accepted_at ASC, commitment.id ASC
               LIMIT 1
          ) commitment ON TRUE
         WHERE epoch.id=requested_pilot_id
           AND claim.status='verified'
           AND claim.verification_last_succeeded_at >
               statement_timestamp()-INTERVAL '7 days'
           AND public.provider_pilot_enrollment_eligibility_is_current(
               epoch.id, claim.id
           )
           AND offer.commercial_terms_contract_version=
               'nhs-provider-commercial-terms-v1'
           AND offer.commercial_terms_sha256 ~ '^[0-9a-f]{64}$'
           AND NOT EXISTS (
               SELECT 1 FROM provider_budget_ledger unverified
                WHERE unverified.provider_offer_id=offer.id
                  AND unverified.entry_type IN ('fund','adjustment')
                  AND NOT EXISTS (
                      SELECT 1
                        FROM provider_commercial_commitment_events linked
                       WHERE linked.budget_ledger_entry_id=unverified.id
                         AND (
                             (linked.event_type='prepaid_fund' AND
                              unverified.entry_type='fund') OR
                             (linked.event_type='fund_reversal' AND
                              unverified.entry_type='adjustment')
                         )
                  )
           );

    WHEN 'ticket' THEN
        SELECT array_to_string(ARRAY[
            'nhs-provider-pilot-review-snapshot-v1', 'ticket',
            epoch.id::text, epoch.contract_version, epoch.demand_topic,
            ticket.provider_claim_id::text, claim.domain_snapshot,
            ticket.provider_offer_id::text,
            ticket.id::text, ticket.offer_version_snapshot::text,
            ticket.offer_name_snapshot, ticket.offer_summary_snapshot,
            ticket.action_type_snapshot, ticket.action_url_snapshot,
            ticket.disclosure_snapshot, ticket.charge_event_snapshot,
            ticket.bounty_cents_snapshot::text, ticket.currency_snapshot,
            ticket.billing_mode_snapshot,
            COALESCE(ticket.terms_credit_limit_cents_snapshot::text, '<null>'),
            COALESCE(ticket.terms_period_days_snapshot::text, '<null>'),
            ticket.commercial_terms_contract_version_snapshot,
            ticket.commercial_terms_sha256_snapshot,
            ticket.principal_price_mode_snapshot,
            COALESCE(ticket.principal_price_cents_snapshot::text, '<null>'),
            ticket.principal_currency_snapshot, ticket.demand_topic,
            ticket.region_code, ticket.budget_band, ticket.urgency,
            array_to_string(ticket.requirement_flags, ','),
            ticket.principal_consent::text, ticket.consent_version,
            ((EXTRACT(EPOCH FROM ticket.created_at) * 1000000)::bigint)::text,
            ((EXTRACT(EPOCH FROM ticket.expires_at) * 1000000)::bigint)::text
        ], E'\n', '<null>')
          INTO snapshot_preimage
          FROM provider_pilot_epochs epoch
          JOIN action_tickets ticket
            ON ticket.provider_pilot_epoch_id=epoch.id
           AND ticket.id=requested_subject_id
           AND NOT ticket.source_is_synthetic
           AND ticket.intent_redacted_at IS NULL
          JOIN provider_pilot_enrollments enrollment
            ON enrollment.provider_pilot_epoch_id=epoch.id
           AND enrollment.provider_claim_id=ticket.provider_claim_id
          JOIN provider_claims claim
            ON claim.id=ticket.provider_claim_id
         WHERE epoch.id=requested_pilot_id
           AND public.provider_pilot_enrollment_eligibility_is_current(
               epoch.id, ticket.provider_claim_id
           );

    WHEN 'handoff' THEN
        SELECT array_to_string(ARRAY[
            'nhs-provider-pilot-review-snapshot-v1', 'handoff',
            epoch.id::text, epoch.contract_version, epoch.demand_topic,
            handoff.provider_claim_id::text, claim.domain_snapshot,
            handoff.provider_offer_id::text,
            handoff.action_ticket_id::text, handoff.id::text,
            handoff.offer_version_snapshot::text,
            handoff.commercial_terms_contract_version_snapshot,
            handoff.commercial_terms_sha256_snapshot,
            ticket.action_type_snapshot,
            handoff.principal_handoff_consent::text,
            handoff.handoff_consent_version,
            handoff.principal_controlled_intent_disclosure_consent::text,
            handoff.controlled_intent_disclosure_consent_version,
            handoff.event_contract_version,
            ((EXTRACT(EPOCH FROM handoff.observed_at) * 1000000)::bigint)::text
        ], E'\n', '<null>')
          INTO snapshot_preimage
          FROM provider_pilot_epochs epoch
          JOIN action_tickets ticket
            ON ticket.provider_pilot_epoch_id=epoch.id
          JOIN provider_action_handoff_receipts handoff
            ON handoff.id=requested_subject_id
           AND handoff.action_ticket_id=ticket.id
           AND handoff.provider_claim_id=ticket.provider_claim_id
           AND handoff.provider_offer_id=ticket.provider_offer_id
          JOIN provider_claims claim
            ON claim.id=handoff.provider_claim_id
         WHERE epoch.id=requested_pilot_id
           AND public.provider_pilot_enrollment_eligibility_is_current(
               epoch.id, handoff.provider_claim_id
           );

    WHEN 'callback' THEN
        SELECT array_to_string(ARRAY[
            'nhs-provider-pilot-review-snapshot-v1', 'callback',
            epoch.id::text, epoch.contract_version, epoch.demand_topic,
            outcome.provider_claim_id::text, claim.domain_snapshot,
            outcome.provider_offer_id::text,
            outcome.action_ticket_id::text, handoff.id::text,
            outcome.id::text, outcome.nhs_event_id::text,
            outcome.provider_api_key_id::text,
            ticket.offer_version_snapshot::text,
            ticket.action_type_snapshot, ticket.charge_event_snapshot,
            ticket.commercial_terms_contract_version_snapshot,
            ticket.commercial_terms_sha256_snapshot, outcome.outcome,
            outcome.billed_cents::text, outcome.charge_status,
            outcome.currency,
            encode(sha256(convert_to(outcome.signed_receipt, 'UTF8')), 'hex'),
            encode(sha256(convert_to(outcome.signature, 'UTF8')), 'hex'),
            ((EXTRACT(EPOCH FROM outcome.provider_reported_at) * 1000000)::bigint)::text,
            ((EXTRACT(EPOCH FROM outcome.created_at) * 1000000)::bigint)::text
        ], E'\n', '<null>')
          INTO snapshot_preimage
          FROM provider_pilot_epochs epoch
          JOIN action_tickets ticket
            ON ticket.provider_pilot_epoch_id=epoch.id
          JOIN provider_action_handoff_receipts handoff
            ON handoff.action_ticket_id=ticket.id
          JOIN outcome_receipts outcome
            ON outcome.id=requested_subject_id
           AND outcome.action_ticket_id=ticket.id
           AND outcome.provider_claim_id=ticket.provider_claim_id
           AND outcome.provider_offer_id=ticket.provider_offer_id
          JOIN provider_claims claim
            ON claim.id=outcome.provider_claim_id
         WHERE epoch.id=requested_pilot_id
           AND public.provider_pilot_enrollment_eligibility_is_current(
               epoch.id, outcome.provider_claim_id
           );

    ELSE
        RAISE EXCEPTION 'unknown provider pilot review type'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_pilot_review_subject';
    END CASE;

    IF snapshot_preimage IS NULL THEN
        RAISE EXCEPTION 'provider pilot review subject is outside the exact pilot scope'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_pilot_review_subject';
    END IF;
    RETURN encode(sha256(convert_to(snapshot_preimage, 'UTF8')), 'hex');
END;
$$;

-- Derive every relational binding from the generic subject ID and reject a
-- stale or fabricated digest. Callers cannot choose a different provider,
-- offer, ticket, handoff, or callback to accompany a reviewed hash.
CREATE OR REPLACE FUNCTION public.enforce_provider_pilot_review_event()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    expected_snapshot_sha256 TEXT;
BEGIN
    IF NEW.review_contract_version IS DISTINCT FROM
       'nhs-provider-pilot-review-v1' THEN
        RAISE EXCEPTION 'provider pilot review contract is invalid'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_pilot_review_contract';
    END IF;

    CASE NEW.review_type
    WHEN 'provider' THEN
        SELECT enrollment.provider_claim_id,
               NULL::uuid, NULL::uuid, NULL::uuid, NULL::uuid
          INTO NEW.provider_claim_id, NEW.provider_offer_id,
               NEW.action_ticket_id, NEW.handoff_receipt_id,
               NEW.outcome_receipt_id
          FROM provider_pilot_enrollments enrollment
         WHERE enrollment.provider_pilot_epoch_id=NEW.provider_pilot_epoch_id
           AND enrollment.provider_claim_id=NEW.subject_id;
    WHEN 'offer' THEN
        SELECT offer.provider_claim_id, offer.id,
               NULL::uuid, NULL::uuid, NULL::uuid
          INTO NEW.provider_claim_id, NEW.provider_offer_id,
               NEW.action_ticket_id, NEW.handoff_receipt_id,
               NEW.outcome_receipt_id
          FROM provider_offers offer
          JOIN provider_pilot_enrollments enrollment
            ON enrollment.provider_pilot_epoch_id=NEW.provider_pilot_epoch_id
           AND enrollment.provider_claim_id=offer.provider_claim_id
         WHERE offer.id=NEW.subject_id
           AND (offer.provider_pilot_epoch_id IS NULL OR
                offer.provider_pilot_epoch_id=NEW.provider_pilot_epoch_id);
    WHEN 'ticket' THEN
        SELECT ticket.provider_claim_id, ticket.provider_offer_id, ticket.id,
               NULL::uuid, NULL::uuid
          INTO NEW.provider_claim_id, NEW.provider_offer_id,
               NEW.action_ticket_id, NEW.handoff_receipt_id,
               NEW.outcome_receipt_id
          FROM action_tickets ticket
         WHERE ticket.id=NEW.subject_id
           AND ticket.provider_pilot_epoch_id=NEW.provider_pilot_epoch_id;
    WHEN 'handoff' THEN
        SELECT handoff.provider_claim_id, handoff.provider_offer_id,
               handoff.action_ticket_id, handoff.id, NULL::uuid
          INTO NEW.provider_claim_id, NEW.provider_offer_id,
               NEW.action_ticket_id, NEW.handoff_receipt_id,
               NEW.outcome_receipt_id
          FROM provider_action_handoff_receipts handoff
          JOIN action_tickets ticket ON ticket.id=handoff.action_ticket_id
         WHERE handoff.id=NEW.subject_id
           AND ticket.provider_pilot_epoch_id=NEW.provider_pilot_epoch_id;
    WHEN 'callback' THEN
        SELECT outcome.provider_claim_id, outcome.provider_offer_id,
               outcome.action_ticket_id, handoff.id, outcome.id
          INTO NEW.provider_claim_id, NEW.provider_offer_id,
               NEW.action_ticket_id, NEW.handoff_receipt_id,
               NEW.outcome_receipt_id
          FROM outcome_receipts outcome
          JOIN action_tickets ticket ON ticket.id=outcome.action_ticket_id
          JOIN provider_action_handoff_receipts handoff
            ON handoff.action_ticket_id=ticket.id
         WHERE outcome.id=NEW.subject_id
           AND ticket.provider_pilot_epoch_id=NEW.provider_pilot_epoch_id;
    ELSE
        RAISE EXCEPTION 'unknown provider pilot review type'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_pilot_review_subject';
    END CASE;

    IF NEW.provider_claim_id IS NULL THEN
        RAISE EXCEPTION 'provider pilot review subject is outside the exact pilot scope'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_pilot_review_subject';
    END IF;

    expected_snapshot_sha256 := provider_pilot_review_snapshot_sha256(
        NEW.provider_pilot_epoch_id, NEW.review_type, NEW.subject_id
    );
    IF NEW.subject_snapshot_sha256 IS DISTINCT FROM expected_snapshot_sha256 THEN
        RAISE EXCEPTION 'provider pilot review snapshot hash does not match database facts'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_pilot_review_snapshot_hash';
    END IF;

    NEW.reviewed_at := statement_timestamp();
    NEW.created_at := NEW.reviewed_at;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS provider_pilot_review_event_enforced
    ON public.provider_pilot_review_events;
CREATE TRIGGER provider_pilot_review_event_enforced
BEFORE INSERT ON public.provider_pilot_review_events
FOR EACH ROW
EXECUTE FUNCTION public.enforce_provider_pilot_review_event();

-- Make the event-relative review chronology an execution gate, not a runbook
-- convention. These three transitions are irreversible proof boundaries: an
-- epoch activation fixes the provider-review deadline, offer activation fixes
-- the offer-review deadline, and handoff observation fixes the ticket-review
-- deadline. A direct SQL writer receives the same fail-closed rule as the
-- application model.
CREATE OR REPLACE FUNCTION public.enforce_provider_pilot_epoch_provider_reviews()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    missing_reviews INTEGER;
BEGIN
    IF OLD.status='draft' AND NEW.status='active' THEN
        SELECT COUNT(*)::integer
          INTO missing_reviews
          FROM provider_pilot_enrollments enrollment
         WHERE enrollment.provider_pilot_epoch_id=OLD.id
           AND NOT EXISTS (
               SELECT 1
                 FROM provider_pilot_review_events review
                WHERE review.provider_pilot_epoch_id=OLD.id
                  AND review.review_type='provider'
                  AND review.subject_id=enrollment.provider_claim_id
                  AND review.review_contract_version=
                      'nhs-provider-pilot-review-v1'
                  AND review.subject_snapshot_sha256=
                      provider_pilot_review_snapshot_sha256(
                          OLD.id, 'provider', enrollment.provider_claim_id
                      )
           );
        IF missing_reviews <> 0 THEN
            RAISE EXCEPTION 'provider pilot activation requires current provider reviews'
                USING ERRCODE='23514',
                      CONSTRAINT='provider_pilot_activation_provider_reviews';
        END IF;
    END IF;
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS provider_pilot_activation_provider_reviews
    ON public.provider_pilot_epochs;
CREATE TRIGGER provider_pilot_activation_provider_reviews
BEFORE UPDATE OF status ON public.provider_pilot_epochs
FOR EACH ROW
EXECUTE FUNCTION public.enforce_provider_pilot_epoch_provider_reviews();

CREATE OR REPLACE FUNCTION public.enforce_provider_offer_pre_activation_review()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.status='active' AND OLD.status IS DISTINCT FROM 'active' THEN
        IF NEW.provider_pilot_epoch_id IS NULL OR NOT EXISTS (
            SELECT 1
              FROM provider_pilot_review_events review
             WHERE review.provider_pilot_epoch_id=NEW.provider_pilot_epoch_id
               AND review.review_type='offer'
               AND review.subject_id=OLD.id
               AND review.review_contract_version=
                   'nhs-provider-pilot-review-v1'
               AND review.subject_snapshot_sha256=
                   provider_pilot_review_snapshot_sha256(
                       NEW.provider_pilot_epoch_id, 'offer', OLD.id
                   )
        ) THEN
            RAISE EXCEPTION 'provider offer activation requires a current offer review'
                USING ERRCODE='23514',
                      CONSTRAINT='provider_offer_activation_review';
        END IF;
    END IF;
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS provider_offer_activation_review
    ON public.provider_offers;
CREATE TRIGGER provider_offer_activation_review
BEFORE UPDATE OF status, provider_pilot_epoch_id ON public.provider_offers
FOR EACH ROW
EXECUTE FUNCTION public.enforce_provider_offer_pre_activation_review();

CREATE OR REPLACE FUNCTION public.enforce_provider_handoff_ticket_review()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    pilot_id UUID;
BEGIN
    SELECT ticket.provider_pilot_epoch_id
      INTO pilot_id
      FROM action_tickets ticket
     WHERE ticket.id=NEW.action_ticket_id
     FOR KEY SHARE;
    IF pilot_id IS NULL OR NOT EXISTS (
        SELECT 1
          FROM provider_pilot_review_events review
         WHERE review.provider_pilot_epoch_id=pilot_id
           AND review.review_type='ticket'
           AND review.subject_id=NEW.action_ticket_id
           AND review.review_contract_version='nhs-provider-pilot-review-v1'
           AND review.subject_snapshot_sha256=
               provider_pilot_review_snapshot_sha256(
                   pilot_id, 'ticket', NEW.action_ticket_id
               )
    ) THEN
        RAISE EXCEPTION 'provider handoff requires a current ticket review'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_handoff_ticket_review';
    END IF;
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS provider_handoff_ticket_review
    ON public.provider_action_handoff_receipts;
CREATE TRIGGER provider_handoff_ticket_review
BEFORE INSERT ON public.provider_action_handoff_receipts
FOR EACH ROW
EXECUTE FUNCTION public.enforce_provider_handoff_ticket_review();

CREATE OR REPLACE RULE provider_pilot_review_events_no_update AS
ON UPDATE TO provider_pilot_review_events DO INSTEAD NOTHING;
CREATE OR REPLACE RULE provider_pilot_review_events_no_delete AS
ON DELETE TO provider_pilot_review_events DO INSTEAD NOTHING;
