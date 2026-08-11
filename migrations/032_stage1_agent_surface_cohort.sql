-- Restrict monetization readiness to the discovery surfaces that actually
-- expose the controlled action-interest affordance. Web browsing remains free,
-- but cannot manufacture evidence that agent users want a downstream action.

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM public.action_interest_receipts interest
          JOIN public.search_receipts receipt
            ON receipt.id=interest.search_receipt_id
         WHERE receipt.surface NOT IN ('mcp','rest')
    ) THEN
        RAISE EXCEPTION 'non-agent search receipt already backs action-interest evidence'
            USING ERRCODE='23514',
                  CONSTRAINT='action_interest_agent_surface';
    END IF;
END;
$$;

CREATE OR REPLACE FUNCTION public.enforce_action_interest_agent_surface()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    source_surface TEXT;
BEGIN
    SELECT receipt.surface
      INTO source_surface
      FROM public.search_receipts receipt
     WHERE receipt.id=NEW.search_receipt_id
     FOR KEY SHARE;
    IF NOT FOUND OR source_surface NOT IN ('mcp','rest') THEN
        RAISE EXCEPTION 'action interest requires an MCP or REST discovery receipt'
            USING ERRCODE='23514',
                  CONSTRAINT='action_interest_agent_surface';
    END IF;
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS stage1_action_interest_agent_surface_enforced
    ON public.action_interest_receipts;
CREATE TRIGGER stage1_action_interest_agent_surface_enforced
BEFORE INSERT ON public.action_interest_receipts
FOR EACH ROW
EXECUTE FUNCTION public.enforce_action_interest_agent_surface();

-- Replace the pilot insert gate so every threshold, bucket, observation span,
-- and canonical snapshot is recomputed from MCP and REST receipts only.
CREATE OR REPLACE FUNCTION public.provider_pilot_stage1_surface_is_eligible(surface_name TEXT)
RETURNS BOOLEAN
LANGUAGE SQL
IMMUTABLE
STRICT
AS $$
    SELECT surface_name IN ('mcp','rest');
$$;

CREATE OR REPLACE FUNCTION public.enforce_provider_pilot_epoch_insert()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    authoritative_stage1_started_at TIMESTAMPTZ;
    meaningful_search_receipts INTEGER;
    result_selections INTEGER;
    search_receipts_with_selection INTEGER;
    action_interest_receipts INTEGER;
    search_receipts_with_action_interest INTEGER;
    distinct_interest_domains INTEGER;
    pilot_topic_receipts INTEGER;
    pilot_topic_eligible_domains INTEGER;
    observation_span_seconds BIGINT;
    canonical_snapshot TEXT;
    expected_snapshot_sha256 TEXT;
    bucket RECORD;
BEGIN
    IF NEW.status IS DISTINCT FROM 'draft' OR
       NEW.activated_at IS NOT NULL OR NEW.closed_at IS NOT NULL THEN
        RAISE EXCEPTION 'provider pilot epochs must be created as drafts'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_pilot_epoch_insert_draft';
    END IF;

    -- 024 can be installed before the clock-owning 025 migration in the same
    -- release sequence, but every later pilot must anchor at 025. Resolving the
    -- latest trusted receipt at execution time prevents caller-mutable facts
    -- written between 020 and 025 from satisfying the fourteen-day gate.
    SELECT applied_at
      INTO authoritative_stage1_started_at
      FROM public.nhs_schema_migrations
     WHERE name=CASE
         WHEN EXISTS (
             SELECT 1 FROM public.nhs_schema_migrations
              WHERE name='025_stage1_fact_integrity.sql'
         ) THEN '025_stage1_fact_integrity.sql'
         ELSE '020_action_interest_receipts.sql'
     END
     FOR KEY SHARE;
    IF NOT FOUND OR
       NEW.stage1_started_at IS DISTINCT FROM authoritative_stage1_started_at OR
       NEW.stage1_evidence_as_of < authoritative_stage1_started_at + INTERVAL '14 days' OR
       NEW.stage1_evidence_as_of > statement_timestamp() THEN
        RAISE EXCEPTION 'provider pilot requires the exact completed Stage 1 observation window'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_pilot_stage1_evidence_window';
    END IF;

    -- Recompute the Stage 1 release thresholds at the supplied evidence clock.
    -- The application stores a canonical aggregate SHA-256 snapshot as the
    -- review receipt, while this database gate prevents a direct SQL writer
    -- from pairing an arbitrary hash with thresholds that were never reached.
    WITH eligible_searches AS (
        SELECT receipt.id, receipt.demand_topics, receipt.created_at
          FROM search_receipts receipt
         WHERE NOT receipt.is_synthetic
           AND public.provider_pilot_stage1_surface_is_eligible(receipt.surface)
           AND receipt.created_at >= GREATEST(
               authoritative_stage1_started_at,
               NEW.stage1_evidence_as_of - INTERVAL '30 days'
           )
           AND receipt.created_at <= NEW.stage1_evidence_as_of
           AND EXISTS (
               SELECT 1
                FROM organic_results_returned returned
                WHERE returned.search_receipt_id=receipt.id
                  AND returned.returned_at >= GREATEST(
                      authoritative_stage1_started_at,
                      NEW.stage1_evidence_as_of - INTERVAL '30 days'
                  )
                  AND returned.returned_at <= NEW.stage1_evidence_as_of
           )
    ), eligible_selections AS (
        SELECT selection.*
          FROM result_selections selection
          JOIN eligible_searches receipt
            ON receipt.id=selection.search_receipt_id
          JOIN organic_results_returned returned
            ON returned.search_receipt_id=selection.search_receipt_id
           AND returned.site_domain_snapshot=selection.site_domain_snapshot
         WHERE selection.selected_at >= GREATEST(
               authoritative_stage1_started_at,
               NEW.stage1_evidence_as_of - INTERVAL '30 days'
           )
           AND selection.selected_at <= NEW.stage1_evidence_as_of
           AND returned.returned_at >= GREATEST(
               authoritative_stage1_started_at,
               NEW.stage1_evidence_as_of - INTERVAL '30 days'
           )
           AND returned.returned_at <= NEW.stage1_evidence_as_of
    ), eligible_interests AS (
        SELECT interest.*
          FROM action_interest_receipts interest
          JOIN eligible_searches receipt
            ON receipt.id=interest.search_receipt_id
         WHERE interest.created_at >= GREATEST(
               authoritative_stage1_started_at,
               NEW.stage1_evidence_as_of - INTERVAL '30 days'
           )
           AND interest.created_at <= NEW.stage1_evidence_as_of
           AND interest.expires_at > NEW.stage1_evidence_as_of
    )
    SELECT
        (SELECT COUNT(*)::integer FROM eligible_searches),
        (SELECT COUNT(*)::integer FROM eligible_selections),
        (SELECT COUNT(DISTINCT selection.search_receipt_id)::integer
           FROM eligible_selections selection),
        (SELECT COUNT(*)::integer FROM eligible_interests),
        (SELECT COUNT(DISTINCT interest.search_receipt_id)::integer
           FROM eligible_interests interest),
        (SELECT COUNT(DISTINCT interest.site_domain_snapshot)::integer
           FROM eligible_interests interest),
        -- Migration 024 cannot name the generation columns that 025 adds.
        -- Once 025 is installed, its earlier epoch-insert trigger rejects the
        -- entire frozen evidence window if any search/result is not generation
        -- 1; the database-owned clocks also prevent facts being backfilled
        -- into that cutoff after the epoch is created.
        (SELECT COUNT(DISTINCT receipt.id)::integer
           FROM eligible_searches receipt
           JOIN organic_results_returned returned
             ON returned.search_receipt_id=receipt.id
            AND returned.returned_at >= GREATEST(
                authoritative_stage1_started_at,
                NEW.stage1_evidence_as_of - INTERVAL '30 days'
            )
            AND returned.returned_at <= NEW.stage1_evidence_as_of
           JOIN sites site
             ON site.id=returned.site_id
            AND site.domain=returned.site_domain_snapshot
            AND site.category<>'spam'
          WHERE NEW.demand_topic=ANY(receipt.demand_topics)),
        (SELECT COUNT(DISTINCT returned.site_domain_snapshot)::integer
           FROM eligible_searches receipt
           JOIN organic_results_returned returned
             ON returned.search_receipt_id=receipt.id
            AND returned.returned_at >= GREATEST(
                authoritative_stage1_started_at,
                NEW.stage1_evidence_as_of - INTERVAL '30 days'
            )
            AND returned.returned_at <= NEW.stage1_evidence_as_of
           JOIN sites site
             ON site.id=returned.site_id
            AND site.domain=returned.site_domain_snapshot
            AND site.category<>'spam'
          WHERE NEW.demand_topic=ANY(receipt.demand_topics)),
        (SELECT COALESCE(FLOOR(EXTRACT(EPOCH FROM
             (MAX(receipt.created_at)-MIN(receipt.created_at)))),0)::bigint
           FROM eligible_searches receipt)
      INTO meaningful_search_receipts, result_selections,
           search_receipts_with_selection, action_interest_receipts,
           search_receipts_with_action_interest, distinct_interest_domains,
           pilot_topic_receipts, pilot_topic_eligible_domains,
           observation_span_seconds;
    IF meaningful_search_receipts < 100 OR
       search_receipts_with_selection < 20 OR
       search_receipts_with_action_interest < 10 OR
       pilot_topic_receipts < 20 OR
       pilot_topic_eligible_domains < 10 OR
       observation_span_seconds < 14 * 24 * 60 * 60 THEN
        RAISE EXCEPTION 'provider pilot requires the exact achieved Stage 1 aggregate thresholds'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_pilot_stage1_thresholds';
    END IF;

    -- Every canonical value below is a fixed label, an allowlisted enum, a
    -- database count/boolean, or a database timestamp represented as Unix
    -- microseconds. Newline is therefore an unambiguous field separator and
    -- lets PostgreSQL verify the exact same snapshot produced by the model.
    canonical_snapshot := concat_ws(E'\n',
        'nhs-provider-pilot-stage1-snapshot-v2',
        'days', '30',
        'retention_days', '30',
        'as_of', (
            EXTRACT(EPOCH FROM NEW.stage1_evidence_as_of) * 1000000
        )::bigint::text,
        'stage1_started_at', (
            EXTRACT(EPOCH FROM authoritative_stage1_started_at) * 1000000
        )::bigint::text,
        'stage1_epoch_enforced', 'true',
        'synthetic_excluded', 'true',
        'eligible_surface', 'mcp',
        'eligible_surface', 'rest',
        'counts_are_receipts_not_unique_agents', 'true',
        'commercial_proof', 'false',
        'meaningful_search_receipts', meaningful_search_receipts::text,
        'result_selections', result_selections::text,
        'search_receipts_with_selection', search_receipts_with_selection::text,
        'action_interest_receipts', action_interest_receipts::text,
        'search_receipts_with_action_interest',
            search_receipts_with_action_interest::text,
        'distinct_interest_domains', distinct_interest_domains::text,
        'bucket_receipt_threshold', '20',
        'topic_buckets_may_overlap', 'true',
        'pilot_candidate_topic_available', 'true',
        'observation_window_days', '14',
        'observation_span_seconds', observation_span_seconds::text,
        'observation_span_days', (observation_span_seconds / 86400)::text,
        'observation_window_met', 'true',
        'stage1_ready', 'true'
    );

    FOR bucket IN
        WITH eligible_searches AS (
            SELECT receipt.id, receipt.demand_topics
              FROM search_receipts receipt
             WHERE NOT receipt.is_synthetic
           AND public.provider_pilot_stage1_surface_is_eligible(receipt.surface)
               AND receipt.created_at >= GREATEST(
                   authoritative_stage1_started_at,
                   NEW.stage1_evidence_as_of - INTERVAL '30 days'
               )
               AND receipt.created_at <= NEW.stage1_evidence_as_of
               AND EXISTS (
                   SELECT 1 FROM organic_results_returned returned
                    WHERE returned.search_receipt_id=receipt.id
                      AND returned.returned_at >= GREATEST(
                          authoritative_stage1_started_at,
                          NEW.stage1_evidence_as_of - INTERVAL '30 days'
                      )
                      AND returned.returned_at <= NEW.stage1_evidence_as_of
               )
        )
        SELECT topic AS value, COUNT(DISTINCT receipt.id)::integer AS receipt_count
          FROM eligible_searches receipt
          CROSS JOIN LATERAL unnest(receipt.demand_topics) AS topic
         WHERE topic=ANY(ARRAY[
             'payments','commerce','jobs','data','search','weather','maps',
             'email','messaging','image','video','audio','documents','security',
             'finance','health','education','news','analytics','automation',
             'productivity','identity','storage','ai-tools','developer-tools','other'
         ]::text[])
         GROUP BY topic
        HAVING COUNT(DISTINCT receipt.id) >= 20
         ORDER BY topic
    LOOP
        canonical_snapshot := concat_ws(E'\n', canonical_snapshot,
            'demand_topic', bucket.value, bucket.receipt_count::text);
    END LOOP;

    FOR bucket IN
        WITH eligible_searches AS (
            SELECT receipt.id, receipt.demand_topics
              FROM search_receipts receipt
             WHERE NOT receipt.is_synthetic
           AND public.provider_pilot_stage1_surface_is_eligible(receipt.surface)
               AND receipt.created_at >= GREATEST(
                   authoritative_stage1_started_at,
                   NEW.stage1_evidence_as_of - INTERVAL '30 days'
               )
               AND receipt.created_at <= NEW.stage1_evidence_as_of
               AND EXISTS (
                   SELECT 1 FROM organic_results_returned returned
                    WHERE returned.search_receipt_id=receipt.id
                      AND returned.returned_at >= GREATEST(
                          authoritative_stage1_started_at,
                          NEW.stage1_evidence_as_of - INTERVAL '30 days'
                      )
                      AND returned.returned_at <= NEW.stage1_evidence_as_of
               )
        )
        SELECT topic AS value, COUNT(DISTINCT receipt.id)::integer AS receipt_count
          FROM eligible_searches receipt
          CROSS JOIN LATERAL unnest(receipt.demand_topics) AS topic
          JOIN organic_results_returned returned
            ON returned.search_receipt_id=receipt.id
           AND returned.returned_at >= GREATEST(
               authoritative_stage1_started_at,
               NEW.stage1_evidence_as_of - INTERVAL '30 days'
           )
           AND returned.returned_at <= NEW.stage1_evidence_as_of
          JOIN sites site
            ON site.id=returned.site_id
           AND site.domain=returned.site_domain_snapshot
           AND site.category<>'spam'
         WHERE topic=ANY(ARRAY[
             'payments','commerce','jobs','data','search','weather','maps',
             'email','messaging','image','video','audio','documents','security',
             'finance','health','education','news','analytics','automation',
             'productivity','identity','storage','ai-tools','developer-tools'
         ]::text[])
         GROUP BY topic
        HAVING COUNT(DISTINCT receipt.id) >= 20
           AND COUNT(DISTINCT returned.site_domain_snapshot) >= 10
         ORDER BY topic
    LOOP
        canonical_snapshot := concat_ws(E'\n', canonical_snapshot,
            'pilot_candidate_topic', bucket.value, bucket.receipt_count::text);
    END LOOP;

    FOR bucket IN
        WITH eligible_searches AS (
            SELECT receipt.id
              FROM search_receipts receipt
             WHERE NOT receipt.is_synthetic
           AND public.provider_pilot_stage1_surface_is_eligible(receipt.surface)
               AND receipt.created_at >= GREATEST(
                   authoritative_stage1_started_at,
                   NEW.stage1_evidence_as_of - INTERVAL '30 days'
               )
               AND receipt.created_at <= NEW.stage1_evidence_as_of
               AND EXISTS (
                   SELECT 1 FROM organic_results_returned returned
                    WHERE returned.search_receipt_id=receipt.id
                      AND returned.returned_at >= GREATEST(
                          authoritative_stage1_started_at,
                          NEW.stage1_evidence_as_of - INTERVAL '30 days'
                      )
                      AND returned.returned_at <= NEW.stage1_evidence_as_of
               )
        )
        SELECT interest.action_type AS value,
               COUNT(DISTINCT interest.search_receipt_id)::integer AS receipt_count
          FROM action_interest_receipts interest
          JOIN eligible_searches receipt ON receipt.id=interest.search_receipt_id
         WHERE interest.created_at >= GREATEST(
               authoritative_stage1_started_at,
               NEW.stage1_evidence_as_of - INTERVAL '30 days'
           )
           AND interest.created_at <= NEW.stage1_evidence_as_of
           AND interest.expires_at > NEW.stage1_evidence_as_of
         GROUP BY interest.action_type
        HAVING COUNT(DISTINCT interest.search_receipt_id) >= 20
         ORDER BY interest.action_type
    LOOP
        canonical_snapshot := concat_ws(E'\n', canonical_snapshot,
            'action_type', bucket.value, bucket.receipt_count::text);
    END LOOP;

    canonical_snapshot := concat_ws(E'\n', canonical_snapshot,
        'target', 'meaningful_search_receipts', '100', 'true',
        'target', 'search_receipts_with_selection', '20', 'true',
        'target', 'search_receipts_with_action_interest', '10', 'true',
        'target', 'pilot_candidate_topic_receipts', '20', 'true',
        'target', 'observation_window_days', '14', 'true'
    );
    expected_snapshot_sha256 := encode(
        sha256(convert_to(canonical_snapshot, 'UTF8')), 'hex'
    );
    IF NEW.stage1_evidence_sha256 IS DISTINCT FROM expected_snapshot_sha256 THEN
        RAISE EXCEPTION 'provider pilot Stage 1 snapshot hash does not match database facts'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_pilot_stage1_snapshot_hash';
    END IF;

    NEW.created_at := statement_timestamp();
    NEW.updated_at := NEW.created_at;
    RETURN NEW;
END;
$$;
