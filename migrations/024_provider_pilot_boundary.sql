-- One owner-authorized, receipt-bounded Stage 2 pilot.
--
-- This migration stores only controlled topic/configuration values, provider
-- claim/company identifiers already present in the exchange, and bounded
-- non-secret owner evidence references. It contains no query, prompt, contact,
-- agent/principal identity, network, or free-form intent fields.

CREATE TABLE IF NOT EXISTS provider_pilot_epochs (
    id                         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    contract_version           TEXT NOT NULL CHECK (
                                   contract_version = 'nhs-provider-pilot-v1'
                               ),
    demand_topic               TEXT NOT NULL CHECK (demand_topic IN (
                                   'payments','commerce','jobs','data','search',
                                   'weather','maps','email','messaging','image',
                                   'video','audio','documents','security','finance',
                                   'health','education','news','analytics','automation',
                                   'productivity','identity','storage','ai-tools',
                                   'developer-tools'
                               )),
    stage1_started_at          TIMESTAMPTZ NOT NULL,
    stage1_evidence_as_of      TIMESTAMPTZ NOT NULL,
    stage1_evidence_sha256     TEXT NOT NULL CHECK (
                                   stage1_evidence_sha256 ~ '^[0-9a-f]{64}$'
                               ),
    cohort_limit               INTEGER NOT NULL CHECK (cohort_limit BETWEEN 3 AND 20),
    provider_ticket_cap        INTEGER NOT NULL CHECK (provider_ticket_cap BETWEEN 1 AND 100),
    total_ticket_cap           INTEGER NOT NULL CHECK (total_ticket_cap BETWEEN 5 AND 2000),
    status                     TEXT NOT NULL DEFAULT 'draft' CHECK (
                                   status IN ('draft','active','closed')
                               ),
    owner_reference            TEXT NOT NULL CHECK (
                                   owner_reference ~ '^[A-Za-z0-9][A-Za-z0-9._:/-]{2,199}$'
                               ),
    evidence_reference         TEXT NOT NULL CHECK (
                                   evidence_reference ~ '^[A-Za-z0-9][A-Za-z0-9._:/-]{2,199}$'
                               ),
    activated_at               TIMESTAMPTZ,
    closed_at                  TIMESTAMPTZ,
    created_at                 TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
    updated_at                 TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
    CHECK (stage1_evidence_as_of >= stage1_started_at + INTERVAL '14 days'),
    CHECK (total_ticket_cap >= cohort_limit),
    CHECK (
        (status='draft' AND activated_at IS NULL AND closed_at IS NULL) OR
        (status='active' AND activated_at IS NOT NULL AND closed_at IS NULL) OR
        (status='closed' AND activated_at IS NOT NULL AND closed_at IS NOT NULL
                         AND closed_at >= activated_at)
    ),
    CHECK (created_at <= updated_at)
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_provider_pilot_one_open_epoch
    ON provider_pilot_epochs ((true)) WHERE status IN ('draft','active');

-- Epoch lifecycle timestamps are database facts, not owner-supplied evidence.
-- Historical Stage 1 evidence timestamps above remain explicit inputs, but a
-- caller cannot manufacture an already-active epoch or backdate its creation.
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
        'nhs-provider-pilot-stage1-snapshot-v1',
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
DROP TRIGGER IF EXISTS provider_pilot_epoch_insert_enforced
    ON public.provider_pilot_epochs;
CREATE TRIGGER provider_pilot_epoch_insert_enforced
BEFORE INSERT ON public.provider_pilot_epochs
FOR EACH ROW
EXECUTE FUNCTION public.enforce_provider_pilot_epoch_insert();

CREATE TABLE IF NOT EXISTS provider_pilot_enrollments (
    id                         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    provider_pilot_epoch_id    UUID NOT NULL REFERENCES provider_pilot_epochs(id) ON DELETE RESTRICT,
    provider_pilot_company_id  UUID NOT NULL,
    provider_claim_id          UUID NOT NULL,
    stage1_eligibility_contract_version TEXT NOT NULL CHECK (
                                   stage1_eligibility_contract_version =
                                   'nhs-provider-pilot-stage1-eligibility-v1'
                               ),
    stage1_eligibility_site_id_snapshot UUID NOT NULL
                                   REFERENCES sites(id) ON DELETE RESTRICT,
    stage1_eligibility_domain_sha256 TEXT NOT NULL CHECK (
                                   stage1_eligibility_domain_sha256 ~ '^[0-9a-f]{64}$'
                               ),
    stage1_eligibility_snapshot_sha256 TEXT NOT NULL CHECK (
                                   stage1_eligibility_snapshot_sha256 ~ '^[0-9a-f]{64}$'
                               ),
    stage1_eligibility_bound_at TIMESTAMPTZ NOT NULL,
    owner_reference            TEXT NOT NULL CHECK (
                                   owner_reference ~ '^[A-Za-z0-9][A-Za-z0-9._:/-]{2,199}$'
                               ),
    evidence_reference         TEXT NOT NULL CHECK (
                                   evidence_reference ~ '^[A-Za-z0-9][A-Za-z0-9._:/-]{2,199}$'
                               ),
    enrolled_at                TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
    UNIQUE (provider_pilot_epoch_id, provider_claim_id),
    UNIQUE (provider_pilot_epoch_id, provider_pilot_company_id),
    FOREIGN KEY (provider_pilot_company_id, provider_claim_id)
        REFERENCES provider_pilot_companies(id, provider_claim_id) ON DELETE RESTRICT
);
CREATE INDEX IF NOT EXISTS idx_provider_pilot_enrollments_claim
    ON provider_pilot_enrollments(provider_claim_id, provider_pilot_epoch_id);

-- This canonical helper accepts only controlled values, opaque UUIDs, a domain
-- digest, and database timestamps. It never accepts or emits the Stage 1
-- receipt, organic rank, raw domain, query, or a domain list/count.
CREATE OR REPLACE FUNCTION public.provider_pilot_stage1_eligibility_snapshot_sha256(
    eligibility_contract_version TEXT,
    pilot_id UUID,
    demand_topic TEXT,
    stage1_started_at TIMESTAMPTZ,
    stage1_evidence_as_of TIMESTAMPTZ,
    pilot_company_id UUID,
    claim_id UUID,
    site_id_snapshot UUID,
    domain_sha256 TEXT,
    eligibility_bound_at TIMESTAMPTZ
)
RETURNS TEXT
LANGUAGE SQL
IMMUTABLE
STRICT
AS $$
    SELECT encode(sha256(convert_to(concat_ws(E'\n',
        eligibility_contract_version,
        pilot_id::text,
        demand_topic,
        ((EXTRACT(EPOCH FROM stage1_started_at) * 1000000)::bigint)::text,
        ((EXTRACT(EPOCH FROM stage1_evidence_as_of) * 1000000)::bigint)::text,
        pilot_company_id::text,
        claim_id::text,
        site_id_snapshot::text,
        domain_sha256,
        ((EXTRACT(EPOCH FROM eligibility_bound_at) * 1000000)::bigint)::text
    ), 'UTF8')), 'hex')
$$;

-- Later pilot actions revalidate only this immutable enrollment binding against
-- current epoch/claim/site state. The expired Stage 1 relations are deliberately
-- absent, so retention cleanup can never erase a legitimate historical bind.
CREATE OR REPLACE FUNCTION public.provider_pilot_enrollment_eligibility_is_current(
    requested_pilot_id UUID,
    requested_claim_id UUID
)
RETURNS BOOLEAN
LANGUAGE SQL
STABLE
STRICT
AS $$
    SELECT EXISTS (
        SELECT 1
          FROM provider_pilot_enrollments enrollment
          JOIN provider_pilot_epochs epoch
            ON epoch.id=enrollment.provider_pilot_epoch_id
          JOIN provider_pilot_companies company
            ON company.id=enrollment.provider_pilot_company_id
           AND company.provider_claim_id=enrollment.provider_claim_id
          JOIN provider_claims claim
            ON claim.id=enrollment.provider_claim_id
          JOIN sites site
            ON site.id=claim.site_id
           AND site.id=enrollment.stage1_eligibility_site_id_snapshot
           AND site.domain=claim.domain_snapshot
           AND site.category<>'spam'
         WHERE enrollment.provider_pilot_epoch_id=requested_pilot_id
           AND enrollment.provider_claim_id=requested_claim_id
           AND enrollment.stage1_eligibility_contract_version=
               'nhs-provider-pilot-stage1-eligibility-v1'
           AND enrollment.stage1_eligibility_domain_sha256=
               encode(sha256(convert_to(claim.domain_snapshot, 'UTF8')), 'hex')
           AND enrollment.stage1_eligibility_snapshot_sha256=
               public.provider_pilot_stage1_eligibility_snapshot_sha256(
                   enrollment.stage1_eligibility_contract_version,
                   epoch.id, epoch.demand_topic, epoch.stage1_started_at,
                   epoch.stage1_evidence_as_of, company.id, claim.id,
                   enrollment.stage1_eligibility_site_id_snapshot,
                   enrollment.stage1_eligibility_domain_sha256,
                   enrollment.stage1_eligibility_bound_at
               )
    )
$$;

-- Locking the epoch row makes the cohort count a transaction-safe gate: two
-- concurrent enrollments cannot both observe the same final slot. The company
-- and claim rows are also locked while their exact, fresh binding is checked.
CREATE OR REPLACE FUNCTION public.enforce_provider_pilot_enrollment()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    epoch_status TEXT;
    epoch_cohort_limit INTEGER;
    epoch_demand_topic TEXT;
    epoch_stage1_started_at TIMESTAMPTZ;
    epoch_stage1_evidence_as_of TIMESTAMPTZ;
    bound_site_id UUID;
    bound_domain TEXT;
    binding_is_fresh BOOLEAN;
    topic_is_eligible BOOLEAN;
    enrolled_count INTEGER;
BEGIN
    SELECT status, cohort_limit, demand_topic, stage1_started_at,
           stage1_evidence_as_of
      INTO epoch_status, epoch_cohort_limit, epoch_demand_topic,
           epoch_stage1_started_at, epoch_stage1_evidence_as_of
      FROM provider_pilot_epochs
     WHERE id=NEW.provider_pilot_epoch_id
     FOR UPDATE;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'provider pilot enrollment requires an existing epoch'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_pilot_enrollment_epoch';
    END IF;
    IF epoch_status IS DISTINCT FROM 'draft' THEN
        RAISE EXCEPTION 'provider pilot enrollment requires a draft epoch'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_pilot_enrollment_epoch_draft';
    END IF;

    SELECT claim.site_id, claim.domain_snapshot, TRUE
      INTO bound_site_id, bound_domain, binding_is_fresh
      FROM provider_pilot_companies company
      JOIN provider_claims claim
        ON claim.id=company.provider_claim_id
      JOIN sites site
        ON site.id=claim.site_id
       AND site.domain=claim.domain_snapshot
       AND site.category<>'spam'
     WHERE company.id=NEW.provider_pilot_company_id
       AND company.provider_claim_id=NEW.provider_claim_id
       AND claim.status='verified'
       AND claim.verification_last_succeeded_at >
           statement_timestamp() - INTERVAL '7 days'
       AND claim.verification_last_succeeded_at <= statement_timestamp()
     FOR KEY SHARE OF company, claim, site;
    IF NOT COALESCE(binding_is_fresh, FALSE) THEN
        RAISE EXCEPTION 'provider pilot enrollment requires the exact fresh verified company claim'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_pilot_enrollment_fresh_company_claim';
    END IF;

    -- The generation-1 guarantee for this frozen epoch window is owned by
    -- migration 025's epoch-insert trigger. Keeping this 024 function free of
    -- not-yet-created columns lets the protected 024 boundary work during the
    -- ordered migration sequence; no later insert can acquire a pre-cutoff
    -- database timestamp.
    SELECT EXISTS (
        SELECT 1
          FROM search_receipts receipt
          JOIN organic_results_returned returned
            ON returned.search_receipt_id=receipt.id
           AND returned.site_id=bound_site_id
           AND returned.site_domain_snapshot=bound_domain
           AND returned.returned_at >= GREATEST(
               epoch_stage1_started_at,
               epoch_stage1_evidence_as_of - INTERVAL '30 days'
           )
           AND returned.returned_at <= epoch_stage1_evidence_as_of
         WHERE NOT receipt.is_synthetic
           AND epoch_demand_topic=ANY(receipt.demand_topics)
           AND receipt.created_at >= GREATEST(
               epoch_stage1_started_at,
               epoch_stage1_evidence_as_of - INTERVAL '30 days'
           )
           AND receipt.created_at <= epoch_stage1_evidence_as_of
    ) INTO topic_is_eligible;
    IF NOT COALESCE(topic_is_eligible, FALSE) THEN
        RAISE EXCEPTION 'provider pilot enrollment requires exact Stage 1 topic organic eligibility'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_pilot_enrollment_stage1_topic_eligibility';
    END IF;

    SELECT COUNT(*)::integer
      INTO enrolled_count
      FROM provider_pilot_enrollments
     WHERE provider_pilot_epoch_id=NEW.provider_pilot_epoch_id;
    IF enrolled_count >= epoch_cohort_limit THEN
        RAISE EXCEPTION 'provider pilot cohort limit reached'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_pilot_enrollment_cohort_cap';
    END IF;

    NEW.enrolled_at := statement_timestamp();
    NEW.stage1_eligibility_contract_version :=
        'nhs-provider-pilot-stage1-eligibility-v1';
    NEW.stage1_eligibility_site_id_snapshot := bound_site_id;
    NEW.stage1_eligibility_domain_sha256 := encode(
        sha256(convert_to(bound_domain, 'UTF8')), 'hex'
    );
    NEW.stage1_eligibility_bound_at := NEW.enrolled_at;
    NEW.stage1_eligibility_snapshot_sha256 :=
        public.provider_pilot_stage1_eligibility_snapshot_sha256(
            NEW.stage1_eligibility_contract_version,
            NEW.provider_pilot_epoch_id, epoch_demand_topic,
            epoch_stage1_started_at, epoch_stage1_evidence_as_of,
            NEW.provider_pilot_company_id, NEW.provider_claim_id,
            NEW.stage1_eligibility_site_id_snapshot,
            NEW.stage1_eligibility_domain_sha256,
            NEW.stage1_eligibility_bound_at
        );
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS provider_pilot_enrollment_enforced
    ON public.provider_pilot_enrollments;
CREATE TRIGGER provider_pilot_enrollment_enforced
BEFORE INSERT ON public.provider_pilot_enrollments
FOR EACH ROW
EXECUTE FUNCTION public.enforce_provider_pilot_enrollment();
CREATE OR REPLACE RULE provider_pilot_enrollments_no_update AS
ON UPDATE TO provider_pilot_enrollments DO INSTEAD NOTHING;
CREATE OR REPLACE RULE provider_pilot_enrollments_no_delete AS
ON DELETE TO provider_pilot_enrollments DO INSTEAD NOTHING;

CREATE TABLE IF NOT EXISTS provider_pilot_epoch_events (
    id                         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    provider_pilot_epoch_id    UUID NOT NULL REFERENCES provider_pilot_epochs(id) ON DELETE RESTRICT,
    event_type                 TEXT NOT NULL CHECK (event_type IN (
                                   'created','provider_enrolled','activated','closed'
                               )),
    provider_claim_id          UUID REFERENCES provider_claims(id) ON DELETE RESTRICT,
    event_snapshot_sha256      TEXT NOT NULL CHECK (
                                   event_snapshot_sha256 ~ '^[0-9a-f]{64}$'
                               ),
    owner_reference            TEXT NOT NULL CHECK (
                                   owner_reference ~ '^[A-Za-z0-9][A-Za-z0-9._:/-]{2,199}$'
                               ),
    evidence_reference         TEXT NOT NULL CHECK (
                                   evidence_reference ~ '^[A-Za-z0-9][A-Za-z0-9._:/-]{2,199}$'
                               ),
    created_at                 TIMESTAMPTZ NOT NULL DEFAULT statement_timestamp(),
    CHECK (
        (event_type='provider_enrolled' AND provider_claim_id IS NOT NULL) OR
        (event_type<>'provider_enrolled' AND provider_claim_id IS NULL)
    )
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_provider_pilot_epoch_singleton_events
    ON provider_pilot_epoch_events(provider_pilot_epoch_id, event_type)
    WHERE event_type IN ('created','activated','closed');
CREATE UNIQUE INDEX IF NOT EXISTS idx_provider_pilot_epoch_enrollment_events
    ON provider_pilot_epoch_events(provider_pilot_epoch_id, provider_claim_id, event_type)
    WHERE event_type='provider_enrolled';

-- Audit rows are append-only below. This trigger also makes their recorded time
-- a database fact and prevents a lifecycle label that has no matching state.
CREATE OR REPLACE FUNCTION public.enforce_provider_pilot_epoch_event()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    epoch_status TEXT;
    expected_event_snapshot_sha256 TEXT;
BEGIN
    SELECT status
      INTO epoch_status
      FROM provider_pilot_epochs
     WHERE id=NEW.provider_pilot_epoch_id
     FOR KEY SHARE;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'provider pilot event requires an existing epoch'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_pilot_epoch_event_epoch';
    END IF;

    IF NEW.event_type='created' AND epoch_status IS DISTINCT FROM 'draft' THEN
        RAISE EXCEPTION 'provider pilot created event requires a draft epoch'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_pilot_epoch_event_state';
    ELSIF NEW.event_type='provider_enrolled' THEN
        IF epoch_status IS DISTINCT FROM 'draft' OR NOT EXISTS (
            SELECT 1
              FROM provider_pilot_enrollments enrollment
             WHERE enrollment.provider_pilot_epoch_id=NEW.provider_pilot_epoch_id
               AND enrollment.provider_claim_id=NEW.provider_claim_id
        ) THEN
            RAISE EXCEPTION 'provider pilot enrollment event requires the exact draft enrollment'
                USING ERRCODE='23514',
                      CONSTRAINT='provider_pilot_epoch_event_enrollment';
        END IF;
    ELSIF NEW.event_type='activated' AND epoch_status IS DISTINCT FROM 'active' THEN
        RAISE EXCEPTION 'provider pilot activated event requires an active epoch'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_pilot_epoch_event_state';
    ELSIF NEW.event_type='closed' AND epoch_status IS DISTINCT FROM 'closed' THEN
        RAISE EXCEPTION 'provider pilot closed event requires a closed epoch'
            USING ERRCODE='23514',
                      CONSTRAINT='provider_pilot_epoch_event_state';
    END IF;

    expected_event_snapshot_sha256 := encode(sha256(convert_to(concat_ws(E'\n',
        'nhs-provider-pilot-event-snapshot-v1', NEW.event_type,
        NEW.provider_pilot_epoch_id::text,
        COALESCE(NEW.provider_claim_id::text, '')
    ), 'UTF8')), 'hex');
    IF NEW.event_snapshot_sha256 IS DISTINCT FROM expected_event_snapshot_sha256 THEN
        RAISE EXCEPTION 'provider pilot event snapshot hash does not match database facts'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_pilot_event_snapshot_hash';
    END IF;

    NEW.created_at := statement_timestamp();
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS provider_pilot_epoch_event_enforced
    ON public.provider_pilot_epoch_events;
CREATE TRIGGER provider_pilot_epoch_event_enforced
BEFORE INSERT ON public.provider_pilot_epoch_events
FOR EACH ROW
EXECUTE FUNCTION public.enforce_provider_pilot_epoch_event();
CREATE OR REPLACE RULE provider_pilot_epoch_events_no_update AS
ON UPDATE TO provider_pilot_epoch_events DO INSTEAD NOTHING;
CREATE OR REPLACE RULE provider_pilot_epoch_events_no_delete AS
ON DELETE TO provider_pilot_epoch_events DO INSTEAD NOTHING;

-- Bind every new public sidecar and ticket to the exact pilot epoch. Historical
-- rows remain NULL and are therefore ineligible for bounded-pilot proof.
ALTER TABLE provider_offers
    ADD COLUMN IF NOT EXISTS provider_pilot_epoch_id UUID
        REFERENCES provider_pilot_epochs(id) ON DELETE RESTRICT;
ALTER TABLE provider_offers_returned
    ADD COLUMN IF NOT EXISTS provider_pilot_epoch_id_snapshot UUID
        REFERENCES provider_pilot_epochs(id) ON DELETE RESTRICT;
ALTER TABLE action_tickets
    ADD COLUMN IF NOT EXISTS provider_pilot_epoch_id UUID
        REFERENCES provider_pilot_epochs(id) ON DELETE RESTRICT;
CREATE INDEX IF NOT EXISTS idx_action_tickets_pilot_epoch_claim
    ON action_tickets(provider_pilot_epoch_id, provider_claim_id)
    WHERE provider_pilot_epoch_id IS NOT NULL;

CREATE OR REPLACE FUNCTION public.enforce_provider_pilot_offer_binding()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    epoch_status TEXT;
    enrollment_is_fresh BOOLEAN;
BEGIN
    IF TG_OP='INSERT' THEN
        IF NEW.provider_pilot_epoch_id IS NOT NULL THEN
            RAISE EXCEPTION 'provider offer pilot binding may be assigned only while activating a draft'
                USING ERRCODE='23514',
                      CONSTRAINT='provider_offer_pilot_binding_transition';
        END IF;
        RETURN NEW;
    END IF;

    IF OLD.provider_pilot_epoch_id IS NOT NULL AND
       OLD.provider_pilot_epoch_id IS DISTINCT FROM NEW.provider_pilot_epoch_id THEN
        RAISE EXCEPTION 'provider offer pilot binding is immutable once assigned'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_offer_pilot_binding_immutable';
    END IF;
    IF OLD.provider_pilot_epoch_id IS NULL AND
       NEW.provider_pilot_epoch_id IS NOT NULL AND NOT (
           OLD.status='draft' AND NEW.status='active'
       ) THEN
        RAISE EXCEPTION 'provider offer pilot binding may be assigned only at activation'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_offer_pilot_binding_transition';
    END IF;
    IF OLD.provider_pilot_epoch_id IS NOT NULL AND
       OLD.provider_claim_id IS DISTINCT FROM NEW.provider_claim_id THEN
        RAISE EXCEPTION 'provider offer pilot claim is immutable once assigned'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_offer_pilot_claim_immutable';
    END IF;

    IF OLD.provider_pilot_epoch_id IS NOT NULL AND
       OLD.activated_at IS DISTINCT FROM NEW.activated_at AND NOT (
           OLD.status='draft' AND NEW.status='active'
       ) THEN
        RAISE EXCEPTION 'provider offer pilot activation time is immutable'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_offer_pilot_activation_immutable';
    END IF;

    IF NEW.status='active' AND NEW.provider_pilot_epoch_id IS NOT NULL THEN
        SELECT status
          INTO epoch_status
          FROM provider_pilot_epochs
         WHERE id=NEW.provider_pilot_epoch_id
         FOR UPDATE;
        IF NOT FOUND OR epoch_status IS DISTINCT FROM 'active' THEN
            RAISE EXCEPTION 'provider offer activation requires an active pilot epoch'
                USING ERRCODE='23514',
                      CONSTRAINT='provider_offer_pilot_epoch_active';
        END IF;

        SELECT TRUE
          INTO enrollment_is_fresh
          FROM provider_pilot_enrollments enrollment
          JOIN provider_claims claim ON claim.id=enrollment.provider_claim_id
          JOIN sites site ON site.id=claim.site_id
         WHERE enrollment.provider_pilot_epoch_id=NEW.provider_pilot_epoch_id
           AND enrollment.provider_claim_id=NEW.provider_claim_id
           AND claim.status='verified'
           AND claim.verification_last_succeeded_at >
               statement_timestamp() - INTERVAL '7 days'
           AND claim.verification_last_succeeded_at <= statement_timestamp()
           AND public.provider_pilot_enrollment_eligibility_is_current(
               NEW.provider_pilot_epoch_id, NEW.provider_claim_id
           )
         FOR KEY SHARE OF claim, site;
        IF NOT COALESCE(enrollment_is_fresh, FALSE) THEN
            RAISE EXCEPTION 'provider offer activation requires the exact fresh enrolled claim'
                USING ERRCODE='23514',
                      CONSTRAINT='provider_offer_pilot_enrollment_claim';
        END IF;

        IF OLD.status='draft' THEN
            NEW.activated_at := statement_timestamp();
        ELSIF OLD.status='paused' AND
              NEW.activated_at IS DISTINCT FROM OLD.activated_at THEN
            RAISE EXCEPTION 'reactivating a pilot offer must preserve its activation time'
                USING ERRCODE='23514',
                      CONSTRAINT='provider_offer_pilot_activation_immutable';
        END IF;
    END IF;
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS provider_offer_pilot_binding_enforced
    ON public.provider_offers;
CREATE TRIGGER provider_offer_pilot_binding_enforced
BEFORE INSERT OR UPDATE OF provider_pilot_epoch_id, provider_claim_id, status, activated_at
ON public.provider_offers
FOR EACH ROW
EXECUTE FUNCTION public.enforce_provider_pilot_offer_binding();

-- The sidecar snapshot is evidence only when it is an exact copy of an active,
-- enrolled offer for a real receipt in the same epoch topic and time window.
CREATE OR REPLACE FUNCTION public.enforce_provider_pilot_returned_offer()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    bound_offer provider_offers%ROWTYPE;
    pilot_epoch provider_pilot_epochs%ROWTYPE;
    receipt search_receipts%ROWTYPE;
    enrollment_is_fresh BOOLEAN;
BEGIN
    SELECT *
      INTO bound_offer
      FROM provider_offers
     WHERE id=NEW.provider_offer_id
     FOR KEY SHARE;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'returned provider offer requires an existing offer'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_returned_offer_binding';
    END IF;

    NEW.returned_at := statement_timestamp();
    IF bound_offer.provider_pilot_epoch_id IS NULL THEN
        IF NEW.provider_pilot_epoch_id_snapshot IS NOT NULL THEN
            RAISE EXCEPTION 'non-pilot offer cannot claim a pilot epoch snapshot'
                USING ERRCODE='23514',
                      CONSTRAINT='provider_returned_offer_pilot_epoch';
        END IF;
        RETURN NEW;
    END IF;

    IF NEW.provider_pilot_epoch_id_snapshot IS DISTINCT FROM
       bound_offer.provider_pilot_epoch_id OR
       NEW.provider_claim_id IS DISTINCT FROM bound_offer.provider_claim_id OR
       ROW(
           NEW.offer_version_snapshot, NEW.offer_name_snapshot,
           NEW.action_type_snapshot, NEW.disclosure_snapshot,
           NEW.bounty_cents_snapshot, NEW.currency_snapshot,
           NEW.charge_event_snapshot,
           NEW.commercial_terms_contract_version_snapshot,
           NEW.commercial_terms_sha256_snapshot
       ) IS DISTINCT FROM ROW(
           bound_offer.version, bound_offer.offer_name,
           bound_offer.action_type, bound_offer.disclosure_label,
           bound_offer.bounty_cents, bound_offer.currency,
           bound_offer.charge_event,
           bound_offer.commercial_terms_contract_version,
           bound_offer.commercial_terms_sha256
       ) THEN
        RAISE EXCEPTION 'returned provider offer snapshot does not exactly match its pilot offer'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_returned_offer_exact_snapshot';
    END IF;

    SELECT *
      INTO pilot_epoch
      FROM provider_pilot_epochs
     WHERE id=bound_offer.provider_pilot_epoch_id
     FOR UPDATE;
    IF NOT FOUND OR pilot_epoch.status IS DISTINCT FROM 'active' OR
       bound_offer.status IS DISTINCT FROM 'active' OR
       bound_offer.activated_at < pilot_epoch.activated_at OR
       bound_offer.activated_at > NEW.returned_at THEN
        RAISE EXCEPTION 'returned provider offer requires an active offer in an active pilot epoch'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_returned_offer_active_epoch';
    END IF;

    SELECT TRUE
      INTO enrollment_is_fresh
      FROM provider_pilot_enrollments enrollment
      JOIN provider_claims claim ON claim.id=enrollment.provider_claim_id
      JOIN sites site ON site.id=claim.site_id
     WHERE enrollment.provider_pilot_epoch_id=pilot_epoch.id
       AND enrollment.provider_claim_id=bound_offer.provider_claim_id
       AND claim.status='verified'
       AND claim.verification_last_succeeded_at >
           statement_timestamp() - INTERVAL '7 days'
       AND claim.verification_last_succeeded_at <= statement_timestamp()
       AND public.provider_pilot_enrollment_eligibility_is_current(
           pilot_epoch.id, bound_offer.provider_claim_id
       )
     FOR KEY SHARE OF claim, site;
    IF NOT COALESCE(enrollment_is_fresh, FALSE) THEN
        RAISE EXCEPTION 'returned provider offer requires the exact fresh enrolled claim'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_returned_offer_enrollment_claim';
    END IF;

    IF NOT EXISTS (
        SELECT 1
          FROM provider_claims claim
          JOIN organic_results_returned organic
            ON organic.search_receipt_id=NEW.search_receipt_id
           AND organic.site_id=claim.site_id
           AND organic.site_domain_snapshot=claim.domain_snapshot
         WHERE claim.id=bound_offer.provider_claim_id
           AND claim.id=NEW.provider_claim_id
    ) THEN
        RAISE EXCEPTION 'returned provider offer requires the exact persisted organic result'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_returned_offer_organic_result';
    END IF;

    SELECT *
      INTO receipt
      FROM search_receipts
     WHERE id=NEW.search_receipt_id
     FOR KEY SHARE;
    IF NOT FOUND OR receipt.is_synthetic OR
       NOT (pilot_epoch.demand_topic=ANY(receipt.demand_topics)) OR
       receipt.created_at < pilot_epoch.activated_at OR
       receipt.created_at > NEW.returned_at THEN
        RAISE EXCEPTION 'returned provider offer receipt is outside the active pilot topic window'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_returned_offer_receipt_window';
    END IF;
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS provider_pilot_returned_offer_enforced
    ON public.provider_offers_returned;
CREATE TRIGGER provider_pilot_returned_offer_enforced
BEFORE INSERT ON public.provider_offers_returned
FOR EACH ROW
EXECUTE FUNCTION public.enforce_provider_pilot_returned_offer();

-- The epoch row is the serialization point for both caps. Because every pilot
-- ticket insert takes that row lock before counting, concurrent writers cannot
-- overrun either the per-provider or total limit under READ COMMITTED.
CREATE OR REPLACE FUNCTION public.enforce_action_ticket_pilot_insert()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    bound_offer provider_offers%ROWTYPE;
    pilot_epoch provider_pilot_epochs%ROWTYPE;
    returned_offer provider_offers_returned%ROWTYPE;
    receipt search_receipts%ROWTYPE;
    enrollment_is_fresh BOOLEAN;
    provider_ticket_count INTEGER;
    total_ticket_count INTEGER;
BEGIN
    SELECT *
      INTO bound_offer
      FROM provider_offers
     WHERE id=NEW.provider_offer_id
     FOR KEY SHARE;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'action ticket requires an existing provider offer'
            USING ERRCODE='23514',
                  CONSTRAINT='action_ticket_pilot_offer';
    END IF;

    IF bound_offer.provider_pilot_epoch_id IS NULL THEN
        IF NEW.provider_pilot_epoch_id IS NOT NULL THEN
            RAISE EXCEPTION 'non-pilot offer cannot mint a pilot ticket'
                USING ERRCODE='23514',
                      CONSTRAINT='action_ticket_pilot_epoch_binding';
        END IF;
        RETURN NEW;
    END IF;
    IF NEW.provider_pilot_epoch_id IS DISTINCT FROM
       bound_offer.provider_pilot_epoch_id THEN
        RAISE EXCEPTION 'action ticket must carry the exact offer pilot epoch'
            USING ERRCODE='23514',
                  CONSTRAINT='action_ticket_pilot_epoch_binding';
    END IF;

    SELECT *
      INTO pilot_epoch
      FROM provider_pilot_epochs
     WHERE id=NEW.provider_pilot_epoch_id
     FOR UPDATE;
    IF NOT FOUND OR pilot_epoch.status IS DISTINCT FROM 'active' OR
       bound_offer.status IS DISTINCT FROM 'active' OR
       bound_offer.provider_claim_id IS DISTINCT FROM NEW.provider_claim_id OR
       bound_offer.activated_at < pilot_epoch.activated_at THEN
        RAISE EXCEPTION 'action ticket requires the exact active pilot offer and epoch'
            USING ERRCODE='23514',
                  CONSTRAINT='action_ticket_pilot_active_offer';
    END IF;

    SELECT TRUE
      INTO enrollment_is_fresh
      FROM provider_pilot_enrollments enrollment
      JOIN provider_claims claim ON claim.id=enrollment.provider_claim_id
      JOIN sites site ON site.id=claim.site_id
     WHERE enrollment.provider_pilot_epoch_id=pilot_epoch.id
       AND enrollment.provider_claim_id=NEW.provider_claim_id
       AND claim.status='verified'
       AND claim.verification_last_succeeded_at >
           statement_timestamp() - INTERVAL '7 days'
       AND claim.verification_last_succeeded_at <= statement_timestamp()
       AND public.provider_pilot_enrollment_eligibility_is_current(
           pilot_epoch.id, NEW.provider_claim_id
       )
     FOR KEY SHARE OF claim, site;
    IF NOT COALESCE(enrollment_is_fresh, FALSE) THEN
        RAISE EXCEPTION 'action ticket requires the exact fresh enrolled claim'
            USING ERRCODE='23514',
                  CONSTRAINT='action_ticket_pilot_enrollment_claim';
    END IF;

    IF ROW(
        NEW.offer_version_snapshot, NEW.offer_name_snapshot,
        NEW.offer_summary_snapshot, NEW.action_type_snapshot,
        NEW.action_url_snapshot, NEW.disclosure_snapshot,
        NEW.charge_event_snapshot, NEW.bounty_cents_snapshot,
        NEW.currency_snapshot, NEW.billing_mode_snapshot,
        NEW.terms_evidence_reference_snapshot,
        NEW.commercial_terms_contract_version_snapshot,
        NEW.commercial_terms_sha256_snapshot,
        NEW.terms_credit_limit_cents_snapshot, NEW.terms_period_days_snapshot,
        NEW.terms_period_anchor_at_snapshot,
        NEW.principal_price_mode_snapshot, NEW.principal_price_cents_snapshot,
        NEW.principal_currency_snapshot
    ) IS DISTINCT FROM ROW(
        bound_offer.version, bound_offer.offer_name,
        bound_offer.offer_summary, bound_offer.action_type,
        bound_offer.action_url, bound_offer.disclosure_label,
        bound_offer.charge_event, bound_offer.bounty_cents,
        bound_offer.currency, bound_offer.billing_mode,
        bound_offer.terms_evidence_reference,
        bound_offer.commercial_terms_contract_version,
        bound_offer.commercial_terms_sha256,
        bound_offer.terms_credit_limit_cents, bound_offer.terms_period_days,
        bound_offer.terms_period_anchor_at,
        bound_offer.principal_price_mode, bound_offer.principal_price_cents,
        bound_offer.principal_currency
    ) THEN
        RAISE EXCEPTION 'action ticket snapshot does not exactly match its pilot offer'
            USING ERRCODE='23514',
                  CONSTRAINT='action_ticket_pilot_exact_snapshot';
    END IF;

    IF NEW.search_receipt_id IS NULL THEN
        RAISE EXCEPTION 'pilot action ticket requires a returned-offer receipt'
            USING ERRCODE='23514',
                  CONSTRAINT='action_ticket_pilot_returned_offer';
    END IF;
    SELECT *
      INTO returned_offer
      FROM provider_offers_returned
     WHERE search_receipt_id=NEW.search_receipt_id
       AND provider_offer_id=NEW.provider_offer_id
     FOR KEY SHARE;
    IF NOT FOUND OR
       returned_offer.provider_claim_id IS DISTINCT FROM NEW.provider_claim_id OR
       returned_offer.provider_pilot_epoch_id_snapshot IS DISTINCT FROM
           NEW.provider_pilot_epoch_id OR
       ROW(
           returned_offer.offer_version_snapshot,
           returned_offer.offer_name_snapshot,
           returned_offer.action_type_snapshot,
           returned_offer.disclosure_snapshot,
           returned_offer.bounty_cents_snapshot,
           returned_offer.currency_snapshot,
           returned_offer.charge_event_snapshot,
           returned_offer.commercial_terms_contract_version_snapshot,
           returned_offer.commercial_terms_sha256_snapshot
       ) IS DISTINCT FROM ROW(
           NEW.offer_version_snapshot, NEW.offer_name_snapshot,
           NEW.action_type_snapshot, NEW.disclosure_snapshot,
           NEW.bounty_cents_snapshot, NEW.currency_snapshot,
           NEW.charge_event_snapshot,
           NEW.commercial_terms_contract_version_snapshot,
           NEW.commercial_terms_sha256_snapshot
       ) THEN
        RAISE EXCEPTION 'action ticket must match the exact returned pilot offer snapshot'
            USING ERRCODE='23514',
                  CONSTRAINT='action_ticket_pilot_returned_offer';
    END IF;

    SELECT *
      INTO receipt
      FROM search_receipts
     WHERE id=NEW.search_receipt_id
     FOR KEY SHARE;
    IF NOT FOUND OR receipt.is_synthetic OR NEW.source_is_synthetic OR
       NEW.source_is_synthetic IS DISTINCT FROM receipt.is_synthetic OR
       NEW.demand_topic IS DISTINCT FROM pilot_epoch.demand_topic OR
       NOT (NEW.demand_topic=ANY(receipt.demand_topics)) OR
       receipt.created_at < pilot_epoch.activated_at OR
       receipt.created_at > returned_offer.returned_at OR
       NEW.created_at + INTERVAL '1 second' < returned_offer.returned_at OR
       NEW.created_at < statement_timestamp() - INTERVAL '1 minute' OR
       NEW.created_at > statement_timestamp() + INTERVAL '1 second' OR
       NEW.updated_at IS DISTINCT FROM NEW.created_at OR
       NEW.expires_at > NEW.created_at + INTERVAL '90 days' THEN
        RAISE EXCEPTION 'action ticket is outside the active pilot topic or authorization window'
            USING ERRCODE='23514',
                  CONSTRAINT='action_ticket_pilot_topic_window';
    END IF;

    SELECT COUNT(*)::integer,
           COUNT(*) FILTER (
               WHERE provider_claim_id=NEW.provider_claim_id
           )::integer
      INTO total_ticket_count, provider_ticket_count
      FROM action_tickets
     WHERE provider_pilot_epoch_id=pilot_epoch.id;
    IF total_ticket_count >= pilot_epoch.total_ticket_cap THEN
        RAISE EXCEPTION 'provider pilot total ticket cap reached'
            USING ERRCODE='23514',
                  CONSTRAINT='action_ticket_pilot_total_cap';
    END IF;
    IF provider_ticket_count >= pilot_epoch.provider_ticket_cap THEN
        RAISE EXCEPTION 'provider pilot provider ticket cap reached'
            USING ERRCODE='23514',
                  CONSTRAINT='action_ticket_pilot_provider_cap';
    END IF;
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS action_ticket_pilot_insert_enforced
    ON public.action_tickets;
CREATE TRIGGER action_ticket_pilot_insert_enforced
BEFORE INSERT ON public.action_tickets
FOR EACH ROW
EXECUTE FUNCTION public.enforce_action_ticket_pilot_insert();

CREATE OR REPLACE FUNCTION public.enforce_action_ticket_pilot_binding()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF ROW(
        OLD.provider_pilot_epoch_id, OLD.provider_claim_id,
        OLD.provider_offer_id, OLD.source_is_synthetic
    ) IS DISTINCT FROM ROW(
        NEW.provider_pilot_epoch_id, NEW.provider_claim_id,
        NEW.provider_offer_id, NEW.source_is_synthetic
    ) THEN
        RAISE EXCEPTION 'action ticket pilot binding is immutable'
            USING ERRCODE='23514',
                  CONSTRAINT='action_ticket_pilot_binding_immutable';
    END IF;

    IF OLD.provider_pilot_epoch_id IS NOT NULL AND ROW(
        OLD.token_hash, OLD.token_nonce, OLD.creation_request_hash,
        OLD.offer_version_snapshot, OLD.offer_name_snapshot,
        OLD.offer_summary_snapshot, OLD.action_type_snapshot,
        OLD.action_url_snapshot, OLD.disclosure_snapshot,
        OLD.charge_event_snapshot, OLD.bounty_cents_snapshot,
        OLD.currency_snapshot, OLD.billing_mode_snapshot,
        OLD.terms_evidence_reference_snapshot,
        OLD.terms_credit_limit_cents_snapshot, OLD.terms_period_days_snapshot,
        OLD.terms_period_anchor_at_snapshot,
        OLD.commercial_terms_contract_version_snapshot,
        OLD.commercial_terms_sha256_snapshot,
        OLD.attribution_key_id_snapshot,
        OLD.principal_price_mode_snapshot, OLD.principal_price_cents_snapshot,
        OLD.principal_currency_snapshot, OLD.principal_consent,
        OLD.consent_version, OLD.created_at, OLD.expires_at
    ) IS DISTINCT FROM ROW(
        NEW.token_hash, NEW.token_nonce, NEW.creation_request_hash,
        NEW.offer_version_snapshot, NEW.offer_name_snapshot,
        NEW.offer_summary_snapshot, NEW.action_type_snapshot,
        NEW.action_url_snapshot, NEW.disclosure_snapshot,
        NEW.charge_event_snapshot, NEW.bounty_cents_snapshot,
        NEW.currency_snapshot, NEW.billing_mode_snapshot,
        NEW.terms_evidence_reference_snapshot,
        NEW.terms_credit_limit_cents_snapshot, NEW.terms_period_days_snapshot,
        NEW.terms_period_anchor_at_snapshot,
        NEW.commercial_terms_contract_version_snapshot,
        NEW.commercial_terms_sha256_snapshot,
        NEW.attribution_key_id_snapshot,
        NEW.principal_price_mode_snapshot, NEW.principal_price_cents_snapshot,
        NEW.principal_currency_snapshot, NEW.principal_consent,
        NEW.consent_version, NEW.created_at, NEW.expires_at
    ) THEN
        RAISE EXCEPTION 'action ticket pilot authorization and commercial snapshot are immutable'
            USING ERRCODE='23514',
                  CONSTRAINT='action_ticket_pilot_snapshot_immutable';
    END IF;

    IF OLD.search_receipt_id IS DISTINCT FROM NEW.search_receipt_id AND NOT (
        OLD.search_receipt_id IS NOT NULL AND
        NEW.search_receipt_id IS NULL AND
        NEW.intent_redacted_at IS NOT NULL AND
        NEW.demand_topic='redacted'
    ) THEN
        RAISE EXCEPTION 'action ticket returned-offer receipt binding is immutable'
            USING ERRCODE='23514',
                  CONSTRAINT='action_ticket_pilot_binding_immutable';
    END IF;
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS action_ticket_pilot_binding_enforced
    ON public.action_tickets;
CREATE TRIGGER action_ticket_pilot_binding_enforced
BEFORE UPDATE OF provider_pilot_epoch_id, provider_claim_id, provider_offer_id,
    search_receipt_id, source_is_synthetic, token_hash, token_nonce,
    creation_request_hash, offer_version_snapshot, offer_name_snapshot,
    offer_summary_snapshot, action_type_snapshot, action_url_snapshot,
    disclosure_snapshot, charge_event_snapshot, bounty_cents_snapshot,
    currency_snapshot, billing_mode_snapshot, terms_evidence_reference_snapshot,
    terms_credit_limit_cents_snapshot, terms_period_days_snapshot,
    terms_period_anchor_at_snapshot,
    commercial_terms_contract_version_snapshot,
    commercial_terms_sha256_snapshot, attribution_key_id_snapshot,
    principal_price_mode_snapshot, principal_price_cents_snapshot,
    principal_currency_snapshot, principal_consent, consent_version,
    created_at, expires_at ON public.action_tickets
FOR EACH ROW
EXECUTE FUNCTION public.enforce_action_ticket_pilot_binding();

-- A ticket minted while the pilot was active may resolve after close, but the
-- principal may not begin a new handoff after the owner closes the epoch. This
-- trigger runs after migration 022's exact claim/offer/ticket trigger (the zz
-- name is deliberate), preserving the shared claim -> offer -> ticket -> epoch
-- lock order and serializing the final decision against close.
CREATE OR REPLACE FUNCTION public.enforce_provider_pilot_handoff_insert()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    ticket_pilot_epoch_id UUID;
    ticket_provider_claim_id UUID;
    ticket_created_at TIMESTAMPTZ;
    epoch_status TEXT;
    epoch_activated_at TIMESTAMPTZ;
BEGIN
    SELECT provider_pilot_epoch_id, provider_claim_id, created_at
      INTO ticket_pilot_epoch_id, ticket_provider_claim_id, ticket_created_at
      FROM action_tickets
     WHERE id=NEW.action_ticket_id
     FOR KEY SHARE;
    IF NOT FOUND OR ticket_pilot_epoch_id IS NULL THEN
        RETURN NEW;
    END IF;

    SELECT status, activated_at
      INTO epoch_status, epoch_activated_at
      FROM provider_pilot_epochs
     WHERE id=ticket_pilot_epoch_id
     FOR UPDATE;
    IF NOT FOUND OR epoch_status IS DISTINCT FROM 'active' OR
       epoch_activated_at IS NULL OR
       ticket_created_at < date_trunc('second', epoch_activated_at) THEN
        RAISE EXCEPTION 'provider pilot handoff requires the exact active ticket epoch'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_pilot_handoff_active_epoch';
    END IF;
    PERFORM 1
      FROM provider_pilot_enrollments enrollment
      JOIN provider_claims claim ON claim.id=enrollment.provider_claim_id
      JOIN sites site ON site.id=claim.site_id
     WHERE enrollment.provider_pilot_epoch_id=ticket_pilot_epoch_id
       AND enrollment.provider_claim_id=ticket_provider_claim_id
     FOR KEY SHARE OF claim, site;
    IF NOT FOUND OR NOT public.provider_pilot_enrollment_eligibility_is_current(
        ticket_pilot_epoch_id, ticket_provider_claim_id
    ) THEN
        RAISE EXCEPTION 'provider pilot handoff requires the current Stage 1 eligibility binding'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_pilot_handoff_enrollment_eligibility';
    END IF;
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS zz_provider_action_handoff_pilot_boundary_enforced
    ON public.provider_action_handoff_receipts;
CREATE TRIGGER zz_provider_action_handoff_pilot_boundary_enforced
BEFORE INSERT ON public.provider_action_handoff_receipts
FOR EACH ROW
EXECUTE FUNCTION public.enforce_provider_pilot_handoff_insert();

CREATE OR REPLACE FUNCTION public.enforce_provider_pilot_epoch_transition()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    enrolled_count INTEGER;
    invalid_enrollment_count INTEGER;
BEGIN
    IF ROW(
        OLD.id, OLD.contract_version, OLD.demand_topic, OLD.stage1_started_at,
        OLD.stage1_evidence_as_of, OLD.stage1_evidence_sha256,
        OLD.cohort_limit, OLD.provider_ticket_cap, OLD.total_ticket_cap,
        OLD.owner_reference, OLD.evidence_reference, OLD.created_at
    ) IS DISTINCT FROM ROW(
        NEW.id, NEW.contract_version, NEW.demand_topic, NEW.stage1_started_at,
        NEW.stage1_evidence_as_of, NEW.stage1_evidence_sha256,
        NEW.cohort_limit, NEW.provider_ticket_cap, NEW.total_ticket_cap,
        NEW.owner_reference, NEW.evidence_reference, NEW.created_at
    ) THEN
        RAISE EXCEPTION 'provider pilot epoch configuration is immutable'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_pilot_epoch_configuration_immutable';
    END IF;

    IF OLD.status='draft' AND NEW.status='active' THEN
        -- Keep claim verification from changing between cohort validation and
        -- the active-state write.
        PERFORM 1
          FROM provider_pilot_enrollments enrollment
          JOIN provider_claims claim ON claim.id=enrollment.provider_claim_id
          JOIN sites site ON site.id=claim.site_id
         WHERE enrollment.provider_pilot_epoch_id=OLD.id
         FOR KEY SHARE OF claim, site;
        SELECT COUNT(*)::integer,
               COUNT(*) FILTER (
                   WHERE claim.status IS DISTINCT FROM 'verified' OR
                         claim.verification_last_succeeded_at IS NULL OR
                         claim.verification_last_succeeded_at <=
                             statement_timestamp() - INTERVAL '7 days' OR
                         claim.verification_last_succeeded_at > statement_timestamp() OR
                         NOT public.provider_pilot_enrollment_eligibility_is_current(
                             OLD.id, enrollment.provider_claim_id
                         )
               )::integer
          INTO enrolled_count, invalid_enrollment_count
          FROM provider_pilot_enrollments enrollment
          JOIN provider_claims claim ON claim.id=enrollment.provider_claim_id
         WHERE enrollment.provider_pilot_epoch_id=OLD.id;
        IF enrolled_count < 3 OR enrolled_count > OLD.cohort_limit THEN
            RAISE EXCEPTION 'provider pilot cohort must contain 3 through the configured limit'
                USING ERRCODE='23514',
                      CONSTRAINT='provider_pilot_epoch_cohort_not_ready';
        END IF;
        IF invalid_enrollment_count <> 0 THEN
            RAISE EXCEPTION 'provider pilot activation requires every enrolled claim to remain fresh and verified'
                USING ERRCODE='23514',
                      CONSTRAINT='provider_pilot_epoch_enrollment_freshness';
        END IF;
        NEW.activated_at := statement_timestamp();
        NEW.closed_at := NULL;
    ELSIF OLD.status='active' AND NEW.status='closed' THEN
        NEW.activated_at := OLD.activated_at;
        NEW.closed_at := statement_timestamp();
    ELSIF OLD.status IS DISTINCT FROM NEW.status THEN
        RAISE EXCEPTION 'provider pilot status transition is not monotonic'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_pilot_epoch_transition';
    ELSIF OLD.activated_at IS DISTINCT FROM NEW.activated_at OR
          OLD.closed_at IS DISTINCT FROM NEW.closed_at THEN
        RAISE EXCEPTION 'provider pilot lifecycle timestamps are immutable'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_pilot_epoch_timestamp_immutable';
    END IF;

    NEW.updated_at := statement_timestamp();
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS provider_pilot_epoch_transition_enforced
    ON public.provider_pilot_epochs;
CREATE TRIGGER provider_pilot_epoch_transition_enforced
BEFORE UPDATE ON public.provider_pilot_epochs
FOR EACH ROW
EXECUTE FUNCTION public.enforce_provider_pilot_epoch_transition();

CREATE OR REPLACE RULE provider_pilot_epochs_no_delete AS
ON DELETE TO provider_pilot_epochs DO INSTEAD NOTHING;
