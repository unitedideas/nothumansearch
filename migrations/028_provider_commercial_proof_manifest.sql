-- Immutable, owner-issued commercial-proof manifests for one exact closed
-- Not Human Search provider pilot.
--
-- This store is intentionally aggregate and privacy-redacted. The signed JSON
-- may contain only the fixed proof-manifest contract enforced by the Go
-- signing package; it never stores provider/company, offer, ticket, handoff,
-- callback, query, search-receipt, bearer, principal, or agent identifiers.

CREATE TABLE IF NOT EXISTS provider_commercial_proof_manifests (
    id                              UUID PRIMARY KEY,
    provider_pilot_epoch_id         UUID NOT NULL UNIQUE
                                    REFERENCES provider_pilot_epochs(id)
                                    ON DELETE RESTRICT,
    manifest_contract_version       TEXT NOT NULL CHECK (
                                        manifest_contract_version =
                                        'nhs-provider-proof-manifest-v1'
                                    ),
    proof_snapshot_sha256           TEXT NOT NULL CHECK (
                                        proof_snapshot_sha256 ~ '^[0-9a-f]{64}$'
                                    ),
    review_evidence_sha256          TEXT NOT NULL CHECK (
                                        review_evidence_sha256 ~ '^[0-9a-f]{64}$'
                                    ),
    key_id                          TEXT NOT NULL CHECK (
                                        key_id ~ '^[A-Za-z0-9][A-Za-z0-9._-]{2,63}$'
                                    ),
    signed_manifest                 TEXT NOT NULL CHECK (
                                        octet_length(signed_manifest) BETWEEN 1 AND 16384
                                    ),
    signature                       TEXT NOT NULL CHECK (
                                        signature ~ '^[A-Za-z0-9_-]{43}$'
                                    ),
    payload_sha256                  TEXT NOT NULL CHECK (
                                        payload_sha256 ~ '^[0-9a-f]{64}$'
                                    ),
    owner_reference                 TEXT NOT NULL CHECK (
                                        owner_reference ~
                                        '^[A-Za-z0-9][A-Za-z0-9._:/-]{2,199}$'
                                    ),
    evidence_reference              TEXT NOT NULL CHECK (
                                        evidence_reference ~
                                        '^[A-Za-z0-9][A-Za-z0-9._:/-]{2,199}$'
                                    ),
    issued_at                       TIMESTAMPTZ NOT NULL,
    created_at                      TIMESTAMPTZ NOT NULL,
    CHECK (issued_at = created_at)
);

REVOKE INSERT, UPDATE, DELETE, TRUNCATE
    ON provider_commercial_proof_manifests FROM PUBLIC;

CREATE INDEX IF NOT EXISTS idx_provider_proof_manifests_issued
    ON provider_commercial_proof_manifests(issued_at, id);
CREATE INDEX IF NOT EXISTS idx_provider_proof_manifests_key
    ON provider_commercial_proof_manifests(key_id, issued_at, id);

-- Database ownership of identity bindings and the issue timestamp prevents a
-- caller from storing a canonical payload under a different pilot, manifest,
-- or key label. HMAC verification remains in the application signing boundary;
-- production startup retains and re-verifies one manifest per referenced key.
CREATE OR REPLACE FUNCTION public.enforce_provider_commercial_proof_manifest()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    document JSONB;
    pilot_status TEXT;
    database_issued_at TIMESTAMPTZ;
BEGIN
    IF NEW.manifest_contract_version IS DISTINCT FROM
       'nhs-provider-proof-manifest-v1' THEN
        RAISE EXCEPTION 'provider proof manifest contract is invalid'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_proof_manifest_contract';
    END IF;

    SELECT status INTO pilot_status
      FROM provider_pilot_epochs
     WHERE id=NEW.provider_pilot_epoch_id
     FOR SHARE;
    IF pilot_status IS DISTINCT FROM 'closed' THEN
        RAISE EXCEPTION 'provider proof manifest requires an exact closed pilot'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_proof_manifest_closed_pilot';
    END IF;
    IF EXISTS (
        SELECT 1
          FROM provider_pilot_enrollments enrollment
         WHERE enrollment.provider_pilot_epoch_id=NEW.provider_pilot_epoch_id
           AND NOT public.provider_pilot_enrollment_eligibility_is_current(
               enrollment.provider_pilot_epoch_id, enrollment.provider_claim_id
           )
    ) THEN
        RAISE EXCEPTION 'provider proof manifest requires current Stage 1 eligibility bindings'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_proof_manifest_enrollment_eligibility';
    END IF;

    BEGIN
        document := NEW.signed_manifest::jsonb;
    EXCEPTION WHEN others THEN
        RAISE EXCEPTION 'provider proof manifest JSON is invalid'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_proof_manifest_json';
    END;

    -- Reject extensible or type-confused documents before the append-only
    -- unique row can be consumed. HMAC verification remains application-owned,
    -- but direct SQL cannot smuggle private fields into the immutable payload.
    IF jsonb_typeof(document) IS DISTINCT FROM 'object' THEN
        RAISE EXCEPTION 'provider proof manifest JSON shape is invalid'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_proof_manifest_json_shape';
    END IF;

    IF (SELECT COUNT(*) FROM jsonb_object_keys(document)) <> 36 OR
       NOT document ?& ARRAY[
           'v','kid','signature_verification_scope',
           'manifest_contract_version','manifest_id',
           'provider_pilot_epoch_id','provider_pilot_contract_version',
           'review_contract_version','review_evidence_contract_version',
           'market_policy_contract_version','proof_snapshot_sha256',
           'review_evidence_sha256','pilot_demand_topic','pilot_status',
           'issued_at','outcome_receipt_integrity_valid',
           'review_integrity_valid','verified_outcome_receipts',
           'rejected_outcome_receipts','verified_outcome_ledger_entries',
           'rejected_outcome_ledger_entries','verified_provider_companies',
           'verified_provider_accepted_handoffs',
           'verified_provider_confirmed_activations',
           'verified_provider_renewals',
           'verified_provider_confirmed_conversions','review_coverage',
           'monetary_amounts_withheld_for_privacy',
           'verified_prepaid_settled','verified_prepaid_net_debited',
           'verified_terms_net_receivable','pilot_thresholds_met',
           'organic_rank_sold','raw_queries_sold','agent_identities_sold',
           'evidence_scope'
       ]::text[] THEN
        RAISE EXCEPTION 'provider proof manifest JSON shape is invalid'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_proof_manifest_json_shape';
    END IF;

    IF jsonb_typeof(document->'kid') IS DISTINCT FROM 'string' OR
       jsonb_typeof(document->'signature_verification_scope') IS DISTINCT FROM 'string' OR
       jsonb_typeof(document->'manifest_contract_version') IS DISTINCT FROM 'string' OR
       jsonb_typeof(document->'manifest_id') IS DISTINCT FROM 'string' OR
       jsonb_typeof(document->'provider_pilot_epoch_id') IS DISTINCT FROM 'string' OR
       jsonb_typeof(document->'provider_pilot_contract_version') IS DISTINCT FROM 'string' OR
       jsonb_typeof(document->'review_contract_version') IS DISTINCT FROM 'string' OR
       jsonb_typeof(document->'review_evidence_contract_version') IS DISTINCT FROM 'string' OR
       jsonb_typeof(document->'market_policy_contract_version') IS DISTINCT FROM 'string' OR
       jsonb_typeof(document->'proof_snapshot_sha256') IS DISTINCT FROM 'string' OR
       jsonb_typeof(document->'review_evidence_sha256') IS DISTINCT FROM 'string' OR
       jsonb_typeof(document->'pilot_demand_topic') IS DISTINCT FROM 'string' OR
       jsonb_typeof(document->'pilot_status') IS DISTINCT FROM 'string' OR
       jsonb_typeof(document->'evidence_scope') IS DISTINCT FROM 'string' OR
       jsonb_typeof(document->'v') IS DISTINCT FROM 'number' OR
       jsonb_typeof(document->'issued_at') IS DISTINCT FROM 'number' OR
       jsonb_typeof(document->'verified_outcome_receipts') IS DISTINCT FROM 'number' OR
       jsonb_typeof(document->'rejected_outcome_receipts') IS DISTINCT FROM 'number' OR
       jsonb_typeof(document->'verified_outcome_ledger_entries') IS DISTINCT FROM 'number' OR
       jsonb_typeof(document->'rejected_outcome_ledger_entries') IS DISTINCT FROM 'number' OR
       jsonb_typeof(document->'verified_provider_companies') IS DISTINCT FROM 'number' OR
       jsonb_typeof(document->'verified_provider_accepted_handoffs') IS DISTINCT FROM 'number' OR
       jsonb_typeof(document->'verified_provider_confirmed_activations') IS DISTINCT FROM 'number' OR
       jsonb_typeof(document->'verified_provider_renewals') IS DISTINCT FROM 'number' OR
       jsonb_typeof(document->'verified_provider_confirmed_conversions') IS DISTINCT FROM 'number' OR
       jsonb_typeof(document->'outcome_receipt_integrity_valid') IS DISTINCT FROM 'boolean' OR
       jsonb_typeof(document->'review_integrity_valid') IS DISTINCT FROM 'boolean' OR
       jsonb_typeof(document->'monetary_amounts_withheld_for_privacy') IS DISTINCT FROM 'boolean' OR
       jsonb_typeof(document->'pilot_thresholds_met') IS DISTINCT FROM 'boolean' OR
       jsonb_typeof(document->'organic_rank_sold') IS DISTINCT FROM 'boolean' OR
       jsonb_typeof(document->'raw_queries_sold') IS DISTINCT FROM 'boolean' OR
       jsonb_typeof(document->'agent_identities_sold') IS DISTINCT FROM 'boolean' OR
       jsonb_typeof(document->'review_coverage') IS DISTINCT FROM 'object' OR
       jsonb_typeof(document->'verified_prepaid_settled') IS DISTINCT FROM 'array' OR
       jsonb_typeof(document->'verified_prepaid_net_debited') IS DISTINCT FROM 'array' OR
       jsonb_typeof(document->'verified_terms_net_receivable') IS DISTINCT FROM 'array' THEN
        RAISE EXCEPTION 'provider proof manifest JSON types are invalid'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_proof_manifest_json_types';
    END IF;

    IF document->'verified_prepaid_settled' <> '[]'::jsonb OR
       document->'verified_prepaid_net_debited' <> '[]'::jsonb OR
       document->'verified_terms_net_receivable' <> '[]'::jsonb OR
       document->>'monetary_amounts_withheld_for_privacy' IS DISTINCT FROM 'true' OR
       (SELECT COUNT(*) FROM jsonb_object_keys(document->'review_coverage')) <> 5 OR
       NOT (document->'review_coverage') ?&
           ARRAY['providers','offers','tickets','handoffs','callbacks']::text[] THEN
        RAISE EXCEPTION 'provider proof manifest privacy shape is invalid'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_proof_manifest_privacy_shape';
    END IF;

    IF document->>'verified_outcome_receipts' !~ '^(0|[1-9][0-9]{0,9})$' OR
       document->>'rejected_outcome_receipts' !~ '^(0|[1-9][0-9]{0,9})$' OR
       document->>'verified_outcome_ledger_entries' !~ '^(0|[1-9][0-9]{0,9})$' OR
       document->>'rejected_outcome_ledger_entries' !~ '^(0|[1-9][0-9]{0,9})$' OR
       document->>'verified_provider_companies' !~ '^(0|[1-9][0-9]{0,9})$' OR
       document->>'verified_provider_accepted_handoffs' !~ '^(0|[1-9][0-9]{0,9})$' OR
       document->>'verified_provider_confirmed_activations' !~ '^(0|[1-9][0-9]{0,9})$' OR
       document->>'verified_provider_renewals' !~ '^(0|[1-9][0-9]{0,9})$' OR
       document->>'verified_provider_confirmed_conversions' !~ '^(0|[1-9][0-9]{0,9})$' THEN
        RAISE EXCEPTION 'provider proof manifest counters are invalid'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_proof_manifest_counter_shape';
    END IF;

    IF EXISTS (
        SELECT 1
         FROM jsonb_each(document->'review_coverage') AS coverage(review_type, counts)
         WHERE jsonb_typeof(counts) IS DISTINCT FROM 'object'
            OR (SELECT COUNT(*) FROM jsonb_object_keys(counts)) <> 2
            OR NOT counts ?& ARRAY['required','valid']::text[]
            OR jsonb_typeof(counts->'required') IS DISTINCT FROM 'number'
            OR jsonb_typeof(counts->'valid') IS DISTINCT FROM 'number'
            OR counts->>'required' !~ '^(0|[1-9][0-9]{0,9})$'
            OR counts->>'valid' !~ '^(0|[1-9][0-9]{0,9})$'
    ) THEN
        RAISE EXCEPTION 'provider proof manifest review coverage is invalid'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_proof_manifest_review_shape';
    END IF;

    IF (document->>'verified_provider_companies')::bigint < 3 OR
       (document->>'verified_provider_accepted_handoffs')::bigint < 5 OR
       (document->>'verified_provider_confirmed_activations')::bigint < 2 OR
       (document->>'verified_provider_renewals')::bigint < 1 OR
       (document->>'rejected_outcome_receipts')::bigint <> 0 OR
       (document->>'rejected_outcome_ledger_entries')::bigint <> 0 OR
       (document->>'verified_provider_renewals')::bigint >
           (document->>'verified_provider_companies')::bigint OR
       (document->>'verified_provider_accepted_handoffs')::bigint >
           (document->>'verified_outcome_receipts')::bigint OR
       (document->>'verified_provider_confirmed_activations')::bigint >
           (document->>'verified_provider_accepted_handoffs')::bigint OR
       (document->>'verified_provider_confirmed_conversions')::bigint >
           (document->>'verified_provider_confirmed_activations')::bigint OR
       (document->>'verified_outcome_ledger_entries')::bigint >
           (document->>'verified_outcome_receipts')::bigint OR
       (document->'review_coverage'->'providers'->>'required')::bigint < 1 OR
       (document->'review_coverage'->'providers'->>'required')::bigint <>
           (document->'review_coverage'->'providers'->>'valid')::bigint OR
       (document->'review_coverage'->'providers'->>'required')::bigint <>
           (document->>'verified_provider_companies')::bigint OR
       (document->'review_coverage'->'offers'->>'required')::bigint <
           (document->>'verified_provider_companies')::bigint OR
       (document->'review_coverage'->'offers'->>'required')::bigint <>
           (document->'review_coverage'->'offers'->>'valid')::bigint OR
       (document->'review_coverage'->'tickets'->>'required')::bigint <
           (document->>'verified_provider_accepted_handoffs')::bigint OR
       (document->'review_coverage'->'tickets'->>'required')::bigint <>
           (document->'review_coverage'->'tickets'->>'valid')::bigint OR
       (document->'review_coverage'->'handoffs'->>'required')::bigint <>
           (document->'review_coverage'->'tickets'->>'required')::bigint OR
       (document->'review_coverage'->'handoffs'->>'required')::bigint <>
           (document->'review_coverage'->'handoffs'->>'valid')::bigint OR
       (document->'review_coverage'->'callbacks'->>'required')::bigint <>
           (document->>'verified_outcome_receipts')::bigint OR
       (document->'review_coverage'->'callbacks'->>'required')::bigint <>
           (document->'review_coverage'->'callbacks'->>'valid')::bigint THEN
        RAISE EXCEPTION 'provider proof manifest aggregate relationships are invalid'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_proof_manifest_aggregate_relationships';
    END IF;

    IF document->>'v' IS DISTINCT FROM '1' OR
       document->>'manifest_contract_version' IS DISTINCT FROM
           NEW.manifest_contract_version OR
       document->>'signature_verification_scope' IS DISTINCT FROM
           'nhs-private-keyring' OR
       document->>'provider_pilot_contract_version' IS DISTINCT FROM
           'nhs-provider-pilot-v1' OR
       document->>'review_contract_version' IS DISTINCT FROM
           'nhs-provider-pilot-review-v1' OR
       document->>'review_evidence_contract_version' IS DISTINCT FROM
           'nhs-provider-proof-review-root-v1' OR
       document->>'market_policy_contract_version' IS DISTINCT FROM
           'nhs-free-organic-provider-funded-v1' OR
       document->>'manifest_id' IS DISTINCT FROM NEW.id::text OR
       document->>'provider_pilot_epoch_id' IS DISTINCT FROM
           NEW.provider_pilot_epoch_id::text OR
       document->>'kid' IS DISTINCT FROM NEW.key_id OR
       document->>'proof_snapshot_sha256' IS DISTINCT FROM
           NEW.proof_snapshot_sha256 OR
       document->>'review_evidence_sha256' IS DISTINCT FROM
           NEW.review_evidence_sha256 OR
       document->>'pilot_status' IS DISTINCT FROM 'closed' OR
       document->>'outcome_receipt_integrity_valid' IS DISTINCT FROM 'true' OR
       document->>'review_integrity_valid' IS DISTINCT FROM 'true' OR
       document->>'pilot_thresholds_met' IS DISTINCT FROM 'true' OR
       document->>'organic_rank_sold' IS DISTINCT FROM 'false' OR
       document->>'raw_queries_sold' IS DISTINCT FROM 'false' OR
       document->>'agent_identities_sold' IS DISTINCT FROM 'false' OR
       document->>'evidence_scope' IS DISTINCT FROM
           'NHS-recorded exact closed-pilot aggregate; HMAC-signed and verifiable only by NHS; not independent proof of provider truth, cash collection, or agent identity.' THEN
        RAISE EXCEPTION 'provider proof manifest JSON bindings are invalid'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_proof_manifest_json_binding';
    END IF;

    database_issued_at := date_trunc('second', transaction_timestamp());
    IF document->>'issued_at' IS DISTINCT FROM
       (EXTRACT(EPOCH FROM database_issued_at)::bigint)::text THEN
        RAISE EXCEPTION 'provider proof manifest issue time is not database owned'
            USING ERRCODE='23514',
                  CONSTRAINT='provider_proof_manifest_issued_at';
    END IF;

    NEW.issued_at := database_issued_at;
    NEW.created_at := database_issued_at;
    NEW.payload_sha256 := encode(
        sha256(convert_to(NEW.signed_manifest, 'UTF8')), 'hex'
    );
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS provider_commercial_proof_manifest_enforced
    ON public.provider_commercial_proof_manifests;
CREATE TRIGGER provider_commercial_proof_manifest_enforced
BEFORE INSERT ON public.provider_commercial_proof_manifests
FOR EACH ROW
EXECUTE FUNCTION public.enforce_provider_commercial_proof_manifest();

CREATE OR REPLACE RULE provider_commercial_proof_manifests_no_update AS
ON UPDATE TO provider_commercial_proof_manifests DO INSTEAD NOTHING;
CREATE OR REPLACE RULE provider_commercial_proof_manifests_no_delete AS
ON DELETE TO provider_commercial_proof_manifests DO INSTEAD NOTHING;
