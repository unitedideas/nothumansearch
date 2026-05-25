# Discovery-quality aggregate fixed point - 2026-05-25

WorkItem: `work_machine_683598b9e16278e8`

Scope stayed inside `/Users/owlassist/foundry-businesses/nothumansearch`.

No full recrawl, broad crawl, deploy, browser automation, desktop automation, public action, production data deletion, process-environment inspection, or secret read was performed.

## Aggregate Inputs

- `tools/recrawl-health.log` records 2026-05-23 full-recrawl completion at `10:44:26` with `api_status=200`, `api_ok=1`, `workers=10`, and `dry_run=0`.
- `tools/recrawl.log` records the 2026-05-23 aggregate result: `Success=9867`, `Failed=379`, `Total=10246`.
- `tools/seed-refresh.log` records 2026-05-24 seed-refresh completion at `05:39:06`: `Success=477`, `Failed=6`, `Total=483`.
- `NHS_DISCOVERY_INPUT=tools/seed-refresh.log ./tools/refresh-discovery-quality.sh` refreshed the sanitized aggregate artifacts.

Refresh output:

```text
discovery_quality_refresh hard_signal_rows=4096 low_signal_rows=1557 category_other_low_signal=108 quarantine_active=true planner_priority=quarantine_first history=harness/discovery-quarantine-history.jsonl
```

## Sanitized Current State

- `sample_rows=5653`
- `hard_signal_rows=4096`
- `low_signal_rows=1557`
- `hard_signal_rate=0.7246`
- `category_other=143`
- `category_other_hard_agent_signal=35`
- `category_other_low_signal=108`
- `llms_only=279`
- `schema_only=286`
- `zero_score=620`
- `passive_or_soft_signal=372`

Hard-signal `category=other` review remains aggregate-only:

- `rows=35`
- `score_bucket_0_24=35`
- `top_signal_sets`: `API=24`, `API,schema.org=11`

## Decision

Current `category=other` state needs no taxonomy-rule change and no threshold adjustment from this aggregate lane.

- `category_other_low_signal`: `no_op_fixed_point`
- `taxonomy_rule`: `not_from_low_signal_aggregate_cohort`
- `threshold_adjustment`: `none`

Reason: low-signal `category=other` rows lack hard agent signals and remain audit-only. Aggregate counts alone are not evidence for a narrow taxonomy rule. Taxonomy-rule work stays reserved for hard-signal `category=other` review, and any executor sample must remain out of committed planner artifacts unless summarized as aggregate proof.

## Guard State

These cohorts remain audit-only unless a hard agent signal is proven:

- `llms_only`: `public_search=false`, `score_fix_targeting=false`
- `schema_only`: `public_search=false`, `score_fix_targeting=false`
- `zero_score`: `public_search=false`, `score_fix_targeting=false`
- `category_other_low_signal`: `public_search=false`, `score_fix_targeting=false`

Public search remains guarded by `models.AgentFirstFilter`. Score-fix targeting remains guarded by `HasHardAgentSignal`.

## Follow-up State

No new discovery-quality follow-up belongs in `harness/generated-work-items.json` for this lane because it is at a true fixed point. The existing generated queue still carries separate non-discovery fixed-point work for API-key commerce/admin traffic buckets and score-fix private cleanup.

## Verification

- `python3 -m json.tool harness/discovery-quality-latest.json`
- `python3 -m json.tool harness/discovery-quarantine-latest.json`
- `python3 -m json.tool harness/generated-work-items.json`
- `python3 tools/test-discovery-quality-report.py`
- `python3 tools/test-discovery-quarantine-report.py`
- `python3 tools/quality-gate-discovery-test.py`
- `python3 tools/test-taxonomy-other-redacted-sample.py`
- `GOCACHE=/private/tmp/nhs-go-cache go test ./internal/... ./cmd/server ./cmd/crawler`
