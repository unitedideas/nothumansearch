# NHS discovery-quality fixed point - work_machine_11871d18a81fd16c

Scope: aggregate-only refresh for the completed 2026-05-21 full-recrawl boundary plus the bounded 2026-05-22 seed-refresh aggregate.

No full recrawl, broad crawl, deploy, browser automation, desktop automation, public action, credential read, production data deletion, raw domain output, raw URL output, row-id output, description output, email output, token output, private-note output, or crawler-row artifact was produced.

## Boundary Evidence

`tools/recrawl-health.log` records:

- 2026-05-21 seed-refresh completion at 05:42:37 with `api_status=200`, `api_ok=1`, `workers=10`, and `dry_run=0`.
- 2026-05-21 full-recrawl start at 06:00:12 with `api_status=200`, `api_ok=1`, and `workers=10`.
- 2026-05-21 full-recrawl completion at 10:25:48 with `api_status=200`, `api_ok=1`, `workers=10`, and `dry_run=0`.
- 2026-05-22 seed-refresh completion at 05:39:47 with `api_status=200`, `api_ok=1`, `workers=10`, and `dry_run=0`.

`tools/recrawl.log` records the 2026-05-21 full-recrawl aggregate completion:

- `success=9847`
- `failed=389`
- `total=10236`

`tools/seed-refresh.log` records the 2026-05-22 seed-refresh aggregate completion:

- `success=476`
- `failed=7`
- `total=483`

Repo-local lock check found no `tools/full-recrawl.lock` and no `tools/recrawl.lock`, so no stale-lock cleanup was needed.

## Refresh

Bounded input:

```bash
awk '/^2026\/05\/22 /{print}' tools/seed-refresh.log > /private/tmp/nhs-seed-refresh-2026-05-22.log
```

Refresh commands:

```bash
python3 tools/discovery-quality-report.py --input /private/tmp/nhs-seed-refresh-2026-05-22.log --output harness/discovery-quality-latest.json
python3 tools/discovery-quarantine-report.py --input harness/discovery-quality-latest.json --output harness/discovery-quarantine-latest.json --history-output harness/discovery-quarantine-history.jsonl --observed-at 2026-05-22T14:10:23Z
python3 tools/quality-gate-discovery.py --quarantine harness/discovery-quarantine-latest.json --repo-root .
```

The refresh updated:

- `harness/discovery-quality-latest.json`
- `harness/discovery-quarantine-latest.json`
- `harness/discovery-quarantine-history.jsonl`

## Aggregate Counts

- `sample_rows=473`
- `hard_signal_rows=343`
- `low_signal_rows=130`
- `hard_signal_rate=0.7252`
- `category_other=12`
- `category_other_hard_agent_signal=3`
- `category_other_low_signal=9`
- `llms_only=23`
- `schema_only=23`
- `zero_score=52`
- `passive_or_soft_signal=32`

Hard-signal category=other review stayed aggregate-only:

- `rows=3`
- `score_0_24=3`
- top signal sets: `API=2`, `API,schema.org=1`

## Decision

Current `category=other` state is a no-op fixed point.

- `taxonomy_rule_change=false`
- `threshold_adjustment=false`
- `no_op_fixed_point=true`
- `category_other_low_signal=no_op_fixed_point`
- `taxonomy_rule=not_from_low_signal_aggregate_cohort`
- `threshold_adjustment=none`

Reason: the low-signal `category=other` cohort lacks proven hard agent signals in this aggregate path. The aggregate counts do not identify a narrow safe taxonomy rule, and lowering thresholds would promote passive or zero-score rows without evidence that they are agent-first.

## Guard Outcome

These cohorts remain audit-only:

- `llms_only`: `public_search=false`, `score_fix_targeting=false`
- `schema_only`: `public_search=false`, `score_fix_targeting=false`
- `zero_score`: `public_search=false`, `score_fix_targeting=false`
- `category_other_low_signal`: `public_search=false`, `score_fix_targeting=false`

Public search remains protected by `models.AgentFirstFilter`. Score-fix targeting still requires a hard agent signal.

## Queue Decision

`harness/generated-work-items.json` is intentionally unchanged for this discovery-quality lane. This lane is at a true fixed point: no replacement full recrawl, broad crawl, taxonomy-rule-change, threshold-adjustment, public-search, or score-fix-targeting follow-up is warranted from this aggregate evidence.

## Verification

- `python3 tools/quality-gate-discovery.py --quarantine harness/discovery-quarantine-latest.json --repo-root .` passed.
- `python3 -m json.tool harness/discovery-quality-latest.json` passed.
- `python3 -m json.tool harness/discovery-quarantine-latest.json` passed.
- `python3 -m json.tool harness/generated-work-items.json` passed.
