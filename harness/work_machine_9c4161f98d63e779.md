# WorkItem closeout: discovery-quality refresh after 2026-05-19 recrawl

WorkItem: `work_machine_9c4161f98d63e779`

No full recrawl, broad crawl, deploy, browser automation, desktop automation, public action, production data deletion, private row inspection, process-environment inspection, or secret read was performed.

## Aggregate Refresh

Refreshed these local aggregate artifacts from the completed 2026-05-19 recrawl boundary:

- `harness/discovery-quality-latest.json`
- `harness/discovery-quarantine-latest.json`
- `harness/discovery-quarantine-history.jsonl`

The refresh used a temporary `/tmp` slice of the May 19 recrawl log and removed it afterward. No raw domains, URLs, row ids, descriptions, emails, tokens, private notes, or crawler row output were committed.

## Boundary Evidence

- `tools/recrawl-health.log`: 2026-05-19 full recrawl started `06:00:11`, preflight `api_status=200 api_ok=1`, health outcome `full_pressure workers=10`, remote start `/app/crawler -recrawl -workers 10`, completion `10:26:32`, post-run `api_status=200 api_ok=1 dry_run=0`.
- `tools/recrawl.log`: `Done. Success: 9830, Failed: 396, Total: 10226` and `2026-05-19 10:26:32 NHS full_recrawl complete`.

## Refreshed Aggregate Counts

- `sample_rows=9827`
- `hard_signal_rows=4114`
- `low_signal_rows=5713`
- `hard_signal_rate=0.4186`
- `category_other=3218`
- `category_other_hard_agent_signal=752`
- `category_other_low_signal=2466`
- `llms_only=1004`
- `schema_only=538`
- `zero_score=2187`
- `passive_or_soft_signal=1984`

`hard_signal_other_review` remains aggregate-only: `rows=752`; score buckets `0_24=431`, `25_39=29`, `40_59=288`, `60_plus=4`.

## Decision

Post-recrawl `category=other` state is a no-op fixed point for this lane.

- `taxonomy_rule_change=false`
- `threshold_adjustment=false`
- `no_op_fixed_point=true`
- `category_other_low_signal=no_op_fixed_point`
- `taxonomy_rule=not_from_low_signal_aggregate_cohort`
- `threshold_adjustment=none`

Reason: low-signal `category=other` rows lack hard agent signals by definition. Aggregate counts alone are not enough evidence for a narrow taxonomy rule, and changing thresholds would promote passive discovery rows rather than proving agent usability.

## Guard State

These cohorts remain audit-only unless a hard agent signal is proven:

- `llms_only`: `public_search=false`, `score_fix_targeting=false`
- `schema_only`: `public_search=false`, `score_fix_targeting=false`
- `zero_score`: `public_search=false`, `score_fix_targeting=false`
- `category_other_low_signal`: `public_search=false`, `score_fix_targeting=false`

Public search remains protected by `models.AgentFirstFilter`. Score-fix targeting remains gated on `HasHardAgentSignal`.

## Follow-up State

`harness/generated-work-items.json` is intentionally unchanged because this lane is at a true fixed point for passive and low-signal cohorts after the May 19 recrawl. No replacement recrawl, broad crawl, taxonomy-rule change, threshold adjustment, public-search promotion, score-fix targeting, browser task, deploy task, public action, or private-row follow-up is warranted from these cohorts.

## Verification

- `python3 tools/test-discovery-quality-report.py`
- `python3 tools/test-discovery-quarantine-report.py`
- `python3 tools/test-refresh-discovery-quality.py`
- `GOCACHE=/tmp/nhs-go-cache go test ./...`

Initial `go test ./...` failed because the default Go cache path under `/Users/owlassist/Library/Caches/go-build` is not writable in this sandbox. The rerun with `GOCACHE=/tmp/nhs-go-cache` passed.

## Commit Blocker

Normal git index writes are blocked in this runner:

`fatal: Unable to create '/Users/owlassist/foundry-businesses/nothumansearch/.git/index.lock': Operation not permitted`

The runner also cannot create any file under `.git`, so staging and committing are not locally feasible in this session.
