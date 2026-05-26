# NHS discovery-quality fixed point - 2026-05-26

Aggregate-only closeout for QLimit WorkItem `work_machine_c96a3844442cdc03`.

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
- `tools/seed-refresh.log` records the bounded 2026-05-24 seed-refresh aggregate:
  - `2026-05-24 05:39:06 NHS seed_refresh complete`
  - `Success=477`
  - `Failed=6`
  - `Total=483`
- The refresh used only the 2026-05-24 seed-refresh slice:
  - `awk '/^2026\\/05\\/24 /{print}' tools/seed-refresh.log > /tmp/nhs-seed-refresh-2026-05-24.log`
  - `NHS_DISCOVERY_INPUT=/tmp/nhs-seed-refresh-2026-05-24.log ./tools/refresh-discovery-quality.sh`
- Refresh output: `discovery_quality_refresh hard_signal_rows=343 low_signal_rows=131 category_other_low_signal=9 quarantine_active=true planner_priority=quarantine_first history=harness/discovery-quarantine-history.jsonl`.

## Refreshed Discovery Aggregates

- `sample_rows=474`
- `hard_signal_rows=343`
- `low_signal_rows=131`
- `hard_signal_rate=0.7236`
- `category_other=12`
- `category_other_hard_agent_signal=3`
- `category_other_low_signal=9`
- `llms_only=23`
- `schema_only=23`
- `zero_score=53`
- `passive_or_soft_signal=32`

`hard_signal_other_review` remains aggregate-only:

- `rows=3`
- `score_bucket_0_24=3`
- `top_signal_sets`: `API=2`, `API,schema.org=1`

## Decision

Current `category=other` state does not justify a taxonomy-rule change from the low-signal aggregate cohort.

- `category_other_low_signal`: `no_op_fixed_point`
- `taxonomy_rule`: `not_from_low_signal_aggregate_cohort`
- `threshold_adjustment`: `none`

Reason: the low-signal `category=other` cohort lacks hard agent signals by definition. It remains audit-only, and aggregate counts alone are not evidence for a narrow taxonomy rule. Taxonomy-rule work is reserved for hard-signal `category=other` rows only, and executor samples from that lane must stay out of committed planner artifacts unless summarized as aggregate proof.

## Guard State

The following cohorts remain audit-only unless a hard agent signal is proven:

- `llms_only`: `public_search=false`, `score_fix_targeting=false`
- `schema_only`: `public_search=false`, `score_fix_targeting=false`
- `zero_score`: `public_search=false`, `score_fix_targeting=false`
- `category_other_low_signal`: `public_search=false`, `score_fix_targeting=false`

Public search remains guarded by `models.AgentFirstFilter`. Score-fix targeting remains guarded by `HasHardAgentSignal`.

## Follow-up State

No replacement full-recrawl, broad crawl, taxonomy-rule-change, threshold-adjustment, public-search, or score-fix-targeting follow-up is warranted from this lane.

`harness/generated-work-items.json` is intentionally unchanged for this WorkItem. The discovery-quality lane is at a true fixed point; the file already carries separate non-discovery follow-ups for API-key commerce/admin traffic, score-fix private cleanup, and monitor quarantine reconciliation.

## Verification

- `python3 -m json.tool` passed for `harness/discovery-quality-latest.json`, `harness/discovery-quarantine-latest.json`, and `harness/generated-work-items.json`.
- `harness/discovery-quarantine-history.jsonl` parsed as JSONL.
- `PYTHONDONTWRITEBYTECODE=1 python3 -m unittest tools/test-discovery-quality-report.py tools/test-discovery-quarantine-report.py tools/quality-gate-discovery-test.py` passed: 17 tests.
- `GOCACHE=/private/tmp/nhs-go-cache go test ./internal/... ./cmd/server ./cmd/crawler` passed.
- `GOCACHE=/private/tmp/nhs-go-cache-full go test ./...` failed on pre-existing `cmd/monitor-check` test drift: undefined `firstCheckFailedQuarantineReason`, `firstCheckZeroScoreQuarantineReason`, and `firstCheckQuarantineReason`.

## Commit Blocker

Commit creation was blocked by the sandbox before staging:

`fatal: Unable to create '/Users/owlassist/foundry-businesses/nothumansearch/.git/index.lock': Operation not permitted`

Files intended for commit:

- `harness/discovery-quality-latest.json`
- `harness/discovery-quarantine-latest.json`
- `harness/discovery-quarantine-history.jsonl`
- `harness/work_machine_c96a3844442cdc03.md`
