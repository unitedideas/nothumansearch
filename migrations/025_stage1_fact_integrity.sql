-- Make the Stage 1 observation clock and the facts counted against it
-- database-owned.  This closes the remaining direct-writer path that could
-- backdate the 020 epoch or its search-to-interest evidence and immediately
-- manufacture a fourteen-day observation window.

-- Protected migration receipts are themselves release evidence.  Later
-- migrations may append rows, but no caller may rewrite or remove an applied
-- migration -- especially the 025 trusted Stage 1 epoch anchor.
CREATE OR REPLACE RULE nhs_schema_migrations_no_update AS
ON UPDATE TO public.nhs_schema_migrations DO INSTEAD NOTHING;
CREATE OR REPLACE RULE nhs_schema_migrations_no_delete AS
ON DELETE TO public.nhs_schema_migrations DO INSTEAD NOTHING;

-- This integrity edge deliberately lives in 025 instead of rewriting the
-- already-protected 020 migration. Every selection counted after the trusted
-- Stage 1 clock begins must point to the exact domain returned by its source
-- search. Validation also fails the migration if historical contamination is
-- present, rather than silently grandfathering it into the protected schema.
ALTER TABLE public.result_selections
    ADD CONSTRAINT result_selections_returned_result_fk
    FOREIGN KEY (search_receipt_id, site_domain_snapshot)
        REFERENCES public.organic_results_returned(search_receipt_id, site_domain_snapshot)
        ON DELETE CASCADE;

-- Existing facts remain NULL: their insertion clocks were caller-owned and a
-- future timestamp cannot be used as proof that they were observed after this
-- migration. Every later insert trigger overwrites the marker to generation 1.
-- The marker is deliberately nullable only for that quarantined legacy set.
ALTER TABLE public.search_receipts
    ADD COLUMN stage1_integrity_generation SMALLINT
        CHECK (stage1_integrity_generation IS NULL OR stage1_integrity_generation=1);
ALTER TABLE public.organic_results_returned
    ADD COLUMN stage1_integrity_generation SMALLINT
        CHECK (stage1_integrity_generation IS NULL OR stage1_integrity_generation=1);
ALTER TABLE public.result_selections
    ADD COLUMN stage1_integrity_generation SMALLINT
        CHECK (stage1_integrity_generation IS NULL OR stage1_integrity_generation=1);
ALTER TABLE public.action_interest_receipts
    ADD COLUMN stage1_integrity_generation SMALLINT
        CHECK (stage1_integrity_generation IS NULL OR stage1_integrity_generation=1);

-- Search receipts have no post-insert lifecycle.  Their controlled
-- classification, result counters, synthetic marker, and observation clock
-- are one immutable fact bundle.  A trigger is used instead of a rewrite rule
-- because the returned-result writers legitimately use ON CONFLICT.
CREATE OR REPLACE FUNCTION public.enforce_search_receipt_stage1_immutability()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION 'Stage 1 search receipt facts are immutable'
        USING ERRCODE='23514',
              CONSTRAINT='search_receipt_stage1_immutable';
END;
$$;
DROP TRIGGER IF EXISTS search_receipt_stage1_immutability_enforced
    ON public.search_receipts;
CREATE TRIGGER search_receipt_stage1_immutability_enforced
BEFORE UPDATE ON public.search_receipts
FOR EACH ROW
EXECUTE FUNCTION public.enforce_search_receipt_stage1_immutability();

-- Returned-result and selection facts are immutable, except that the existing
-- sites(id) ON DELETE SET NULL referential action must remain usable.  A site
-- deletion may erase only the optional site_id pointer; the receipt/domain,
-- position/surface, score, and observed clock cannot change.
CREATE OR REPLACE FUNCTION public.enforce_organic_result_stage1_immutability()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.id IS DISTINCT FROM OLD.id OR
       NEW.search_receipt_id IS DISTINCT FROM OLD.search_receipt_id OR
       NEW.site_domain_snapshot IS DISTINCT FROM OLD.site_domain_snapshot OR
       NEW.organic_position IS DISTINCT FROM OLD.organic_position OR
       NEW.score_snapshot IS DISTINCT FROM OLD.score_snapshot OR
       NEW.returned_at IS DISTINCT FROM OLD.returned_at OR
       NEW.stage1_integrity_generation IS DISTINCT FROM OLD.stage1_integrity_generation OR
       NOT (
           NEW.site_id IS NOT DISTINCT FROM OLD.site_id OR
           (OLD.site_id IS NOT NULL AND NEW.site_id IS NULL)
       ) THEN
        RAISE EXCEPTION 'Stage 1 organic result facts are immutable'
            USING ERRCODE='23514',
                  CONSTRAINT='organic_result_stage1_immutable';
    END IF;
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS organic_result_stage1_immutability_enforced
    ON public.organic_results_returned;
CREATE TRIGGER organic_result_stage1_immutability_enforced
BEFORE UPDATE ON public.organic_results_returned
FOR EACH ROW
EXECUTE FUNCTION public.enforce_organic_result_stage1_immutability();

CREATE OR REPLACE FUNCTION public.enforce_result_selection_stage1_immutability()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.id IS DISTINCT FROM OLD.id OR
       NEW.search_receipt_id IS DISTINCT FROM OLD.search_receipt_id OR
       NEW.site_domain_snapshot IS DISTINCT FROM OLD.site_domain_snapshot OR
       NEW.surface IS DISTINCT FROM OLD.surface OR
       NEW.selected_at IS DISTINCT FROM OLD.selected_at OR
       NEW.stage1_integrity_generation IS DISTINCT FROM OLD.stage1_integrity_generation OR
       NOT (
           NEW.site_id IS NOT DISTINCT FROM OLD.site_id OR
           (OLD.site_id IS NOT NULL AND NEW.site_id IS NULL)
       ) THEN
        RAISE EXCEPTION 'Stage 1 result selection facts are immutable'
            USING ERRCODE='23514',
                  CONSTRAINT='result_selection_stage1_immutable';
    END IF;
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS result_selection_stage1_immutability_enforced
    ON public.result_selections;
CREATE TRIGGER result_selection_stage1_immutability_enforced
BEFORE UPDATE ON public.result_selections
FOR EACH ROW
EXECUTE FUNCTION public.enforce_result_selection_stage1_immutability();

CREATE OR REPLACE FUNCTION public.own_search_receipt_created_at()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.created_at := clock_timestamp();
    NEW.stage1_integrity_generation := 1;
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS stage1_search_receipt_insert_timestamp_owned
    ON public.search_receipts;
CREATE TRIGGER stage1_search_receipt_insert_timestamp_owned
BEFORE INSERT ON public.search_receipts
FOR EACH ROW
EXECUTE FUNCTION public.own_search_receipt_created_at();

CREATE OR REPLACE FUNCTION public.own_organic_result_returned_at()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.returned_at := clock_timestamp();
    NEW.stage1_integrity_generation := 1;
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS stage1_organic_result_insert_timestamp_owned
    ON public.organic_results_returned;
CREATE TRIGGER stage1_organic_result_insert_timestamp_owned
BEFORE INSERT ON public.organic_results_returned
FOR EACH ROW
EXECUTE FUNCTION public.own_organic_result_returned_at();

CREATE OR REPLACE FUNCTION public.own_result_selection_selected_at()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.selected_at := clock_timestamp();
    NEW.stage1_integrity_generation := 1;
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS stage1_result_selection_insert_timestamp_owned
    ON public.result_selections;
CREATE TRIGGER stage1_result_selection_insert_timestamp_owned
BEFORE INSERT ON public.result_selections
FOR EACH ROW
EXECUTE FUNCTION public.own_result_selection_selected_at();

CREATE OR REPLACE FUNCTION public.own_action_interest_created_at()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.created_at := clock_timestamp();
    NEW.stage1_integrity_generation := 1;
    -- Expiry is part of Stage 1 eligibility, so it is derived from the exact
    -- source-receipt retention boundary instead of trusting a caller clock.
    SELECT receipt.created_at + INTERVAL '30 days'
      INTO NEW.expires_at
      FROM public.search_receipts receipt
     WHERE receipt.id = NEW.search_receipt_id;
    IF NOT FOUND THEN
        -- Preserve a deterministic non-null row shape so the canonical
        -- foreign key, rather than a caller timestamp, rejects a missing
        -- source receipt after this BEFORE trigger.
        NEW.expires_at := NEW.created_at + INTERVAL '30 days';
    END IF;
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS stage1_action_interest_insert_timestamp_owned
    ON public.action_interest_receipts;
CREATE TRIGGER stage1_action_interest_insert_timestamp_owned
BEFORE INSERT ON public.action_interest_receipts
FOR EACH ROW
EXECUTE FUNCTION public.own_action_interest_created_at();

-- This statement-level trigger runs before every row-level pilot insert
-- trigger and holds an UPDATE-strength lock on the exact 025 receipt for the
-- entire inserting transaction. The 024 row trigger resolves the same trusted
-- anchor dynamically once 025 is receipted, so no pre-025 caller-owned fact
-- can satisfy the Stage 1 observation window.
CREATE OR REPLACE FUNCTION public.lock_provider_pilot_stage1_epoch_anchor()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    PERFORM 1
      FROM public.nhs_schema_migrations
     WHERE name = '025_stage1_fact_integrity.sql'
     FOR UPDATE;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'provider pilot requires the protected Stage 1 epoch anchor'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_pilot_stage1_epoch_anchor';
    END IF;
    RETURN NULL;
END;
$$;
DROP TRIGGER IF EXISTS aa_provider_pilot_stage1_epoch_anchor_locked
    ON public.provider_pilot_epochs;
CREATE TRIGGER aa_provider_pilot_stage1_epoch_anchor_locked
BEFORE INSERT ON public.provider_pilot_epochs
FOR EACH STATEMENT
EXECUTE FUNCTION public.lock_provider_pilot_stage1_epoch_anchor();

-- Migration 024 computes its canonical snapshot from the Stage 1 relations.
-- Before that row trigger runs, reject the pilot if any fact in the exact
-- eligible cohort lacks generation 1. This makes the 024 aggregate and the
-- generation-filtered application report equivalent without grandfathering a
-- pre-025 row whose caller supplied a future timestamp.
CREATE OR REPLACE FUNCTION public.enforce_provider_pilot_stage1_generation()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    authoritative_stage1_started_at TIMESTAMPTZ;
    untrusted_fact_exists BOOLEAN;
BEGIN
    SELECT applied_at
      INTO authoritative_stage1_started_at
      FROM public.nhs_schema_migrations
     WHERE name='025_stage1_fact_integrity.sql'
     FOR KEY SHARE;
    IF NOT FOUND OR
       NEW.stage1_started_at IS DISTINCT FROM authoritative_stage1_started_at THEN
        RAISE EXCEPTION 'provider pilot requires the trusted Stage 1 integrity generation'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_pilot_stage1_integrity_generation';
    END IF;

    WITH eligible_searches AS (
        SELECT receipt.id, receipt.stage1_integrity_generation
          FROM public.search_receipts receipt
         WHERE NOT receipt.is_synthetic
           AND receipt.created_at >= GREATEST(
               authoritative_stage1_started_at,
               NEW.stage1_evidence_as_of - INTERVAL '30 days'
           )
           AND receipt.created_at <= NEW.stage1_evidence_as_of
           AND EXISTS (
               SELECT 1
                 FROM public.organic_results_returned returned
                WHERE returned.search_receipt_id=receipt.id
                  AND returned.returned_at >= GREATEST(
                      authoritative_stage1_started_at,
                      NEW.stage1_evidence_as_of - INTERVAL '30 days'
                  )
                  AND returned.returned_at <= NEW.stage1_evidence_as_of
           )
    )
    SELECT EXISTS (
        SELECT 1
          FROM eligible_searches receipt
         WHERE receipt.stage1_integrity_generation IS DISTINCT FROM 1
            OR EXISTS (
                SELECT 1
                 FROM public.organic_results_returned returned
                 WHERE returned.search_receipt_id=receipt.id
                   AND returned.returned_at >= GREATEST(
                       authoritative_stage1_started_at,
                       NEW.stage1_evidence_as_of - INTERVAL '30 days'
                   )
                   AND returned.returned_at <= NEW.stage1_evidence_as_of
                   AND returned.stage1_integrity_generation IS DISTINCT FROM 1
            )
            OR EXISTS (
                SELECT 1
                  FROM public.result_selections selection
                  JOIN public.organic_results_returned returned
                    ON returned.search_receipt_id=selection.search_receipt_id
                   AND returned.site_domain_snapshot=selection.site_domain_snapshot
                 WHERE selection.search_receipt_id=receipt.id
                   AND selection.selected_at >= GREATEST(
                       authoritative_stage1_started_at,
                       NEW.stage1_evidence_as_of - INTERVAL '30 days'
                   )
                   AND selection.selected_at <= NEW.stage1_evidence_as_of
                   AND returned.returned_at >= GREATEST(
                       authoritative_stage1_started_at,
                       NEW.stage1_evidence_as_of - INTERVAL '30 days'
                   )
                   AND returned.returned_at <= NEW.stage1_evidence_as_of
                   AND (
                       selection.stage1_integrity_generation IS DISTINCT FROM 1 OR
                       returned.stage1_integrity_generation IS DISTINCT FROM 1
                   )
            )
            OR EXISTS (
                SELECT 1
                  FROM public.action_interest_receipts interest
                 WHERE interest.search_receipt_id=receipt.id
                   AND interest.created_at >= GREATEST(
                       authoritative_stage1_started_at,
                       NEW.stage1_evidence_as_of - INTERVAL '30 days'
                   )
                   AND interest.created_at <= NEW.stage1_evidence_as_of
                   AND interest.expires_at > NEW.stage1_evidence_as_of
                   AND interest.stage1_integrity_generation IS DISTINCT FROM 1
            )
    ) INTO untrusted_fact_exists;

    IF untrusted_fact_exists THEN
        RAISE EXCEPTION 'provider pilot cohort contains a pre-integrity Stage 1 fact'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_pilot_stage1_integrity_generation';
    END IF;
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS ab_provider_pilot_stage1_generation_enforced
    ON public.provider_pilot_epochs;
CREATE TRIGGER ab_provider_pilot_stage1_generation_enforced
BEFORE INSERT ON public.provider_pilot_epochs
FOR EACH ROW
EXECUTE FUNCTION public.enforce_provider_pilot_stage1_generation();
