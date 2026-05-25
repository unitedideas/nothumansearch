# NHS discovery-quality fixed point - 2026-05-25

Aggregate-only closeout for QLimit WorkItem `work_machine_524371fec87a56f4`.

No full recrawl, broad crawl, deploy, browser automation, desktop automation, public action, production data deletion, process-environment inspection, or secret read was performed.

## Inputs

- `tools/recrawl-health.log` records the completed 2026-05-23 full-recrawl boundary:
  - start: `2026-05-23 06:00:08`
  - preflight: `api_status=200`, `api_ok=1`
  - workers: `10`
  - completion: `2026-05-23 10:44:26`
  - post-run: `api_status=200`, `api_ok=1`, `dry_run=0`
- `tools/recrawl.log` records the matching aggregate crawler result:
  - `Success=9867`
  - `Failed=379`
  - `Total=10246`
- `tools/recrawl-health.log` records the latest completed seed-refresh boundary:
  - start: `2026-05-24 05:30:07`
  - preflight: `api_status=200`, `api_ok=1`
  - workers: `10`
  - completion: `2026-05-24 05:39:06`
  - post-run: `api_status=200`, `api_ok=1`, `dry_run=0`
- `tools/seed-refresh.log` records the matching aggregate seed-refresh result:
  - `Success=477`
  - `Failed=6`
  - `Total=483`

## Refresh Proof

Command:

```bash
NHS_DISCOVERY_INPUT=tools/seed-refresh.log ./tools/refresh-discovery-quality.sh
```

Output:

```text
discovery_quality_refresh hard_signal_rows=4096 low_signal_rows=1557 category_other_low_signal=108 quarantine_active=true planner_priority=quarantine_first history=harness/discovery-quarantine-history.jsonl
```

The command refreshed `harness/discovery-quality-latest.json`, `harness/discovery-quarantine-latest.json`, and `harness/discovery-quarantine-history.jsonl`. The first two are ignored aggregate working artifacts; this tracked note is the committed proof.

## Refreshed Discovery Aggregates

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

`hard_signal_other_review` remains aggregate-only:

- `rows=35`
- `score_bucket_0_24=35`
- `top_signal_sets`: `API=24`, `API,schema.org=11`

## Decision

Current `category=other` state does not justify a taxonomy-rule change from the low-signal aggregate cohort.

- `category_other_low_signal`: `no_op_fixed_point`
- `taxonomy_rule`: `not_from_low_signal_aggregate_cohort`
- `threshold_adjustment`: `none`

Reason: the low-signal `category=other` cohort has no hard agent signal by definition. It remains audit-only, and aggregate counts alone are not evidence for a narrow taxonomy rule. Taxonomy-rule work stays limited to hard-signal `category=other` rows, with executor samples kept out of committed planner artifacts unless summarized as aggregate proof.

## Guard State

The following cohorts remain audit-only unless a hard agent signal is proven:

- `llms_only`: `public_search=false`, `score_fix_targeting=false`
- `schema_only`: `public_search=false`, `score_fix_targeting=false`
- `zero_score`: `public_search=false`, `score_fix_targeting=false`
- `category_other_low_signal`: `public_search=false`, `score_fix_targeting=false`

Public search remains guarded by `models.AgentFirstFilter`. Score-fix targeting remains guarded by `HasHardAgentSignal`.

## Follow-up State

No replacement full-recrawl, broad crawl, taxonomy-rule-change, threshold-adjustment, public-search, or score-fix-targeting follow-up is warranted from this lane.

`harness/generated-work-items.json` already carries the remaining concrete follow-ups for API-key commerce/admin traffic buckets and score-fix private cleanup, so no new discovery-quality follow-up was added.

## Verification

- `python3 tools/test-discovery-quality-report.py && python3 tools/test-discovery-quarantine-report.py && python3 tools/quality-gate-discovery-test.py` passed.
- `GOCACHE=/private/tmp/nothumansearch-gocache go test ./internal/... ./cmd/server ./cmd/crawler` passed.
- `GOCACHE=/private/tmp/nothumansearch-gocache go test ./...` did not pass because `cmd/monitor-check` still references missing helper symbols: `firstCheckFailedQuarantineReason`, `firstCheckZeroScoreQuarantineReason`, and `firstCheckQuarantineReason`. This is outside the discovery-quality artifact lane.

## Commit State

Commit creation was attempted with:

```bash
git add harness/work_machine_524371fec87a56f4.md && git commit -m "Record discovery quality fixed point"
```

It failed before staging with:

```text
fatal: Unable to create '/Users/owlassist/foundry-businesses/nothumansearch/.git/index.lock': Operation not permitted
```

Next action: run the same `git add` and `git commit` from a git-writable worker. No content or state retry requires a crawl.
