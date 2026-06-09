# Discovery Quality Fixed-Point Refresh

Work item: `work_machine_1259ddba6da25c3f`

Scope: aggregate-only discovery-quality and discovery-quarantine refresh from the completed full-recrawl boundary plus the 2026-05-22 seed-refresh aggregate.

## Boundary Evidence

- `tools/recrawl-health.log` records 2026-05-21 `seed_refresh` completion at `05:42:37` with `api_status=200`, `api_ok=1`, `workers=10`, and `dry_run=0`.
- `tools/recrawl-health.log` records 2026-05-21 `full_recrawl` completion at `10:25:48` with `api_status=200`, `api_ok=1`, `workers=10`, and `dry_run=0`.
- `tools/recrawl.log` records the matching full-recrawl aggregate: `Success=9847`, `Failed=389`, `Total=10236`.
- `tools/recrawl-health.log` records 2026-05-22 `seed_refresh` completion at `05:39:47` with `api_status=200`, `api_ok=1`, `workers=10`, and `dry_run=0`.
- `tools/seed-refresh.log` records the matching seed-refresh aggregate: `Success=476`, `Failed=7`, `Total=483`.
- Repo-local lock check found no `tools/full-recrawl.lock` or `tools/recrawl.lock`.

## Refresh

Command shape:

```bash
awk '/^2026\/05\/22 /{print}' tools/seed-refresh.log > /private/tmp/nhs-seed-refresh-2026-05-22.log
NHS_DISCOVERY_INPUT=/private/tmp/nhs-seed-refresh-2026-05-22.log ./tools/refresh-discovery-quality.sh
```

Output:

```text
discovery_quality_refresh hard_signal_rows=343 low_signal_rows=130 category_other_low_signal=9 quarantine_active=true planner_priority=quarantine_first history=harness/discovery-quarantine-history.jsonl
```

The temporary input slice was not committed. The committed artifacts are aggregate-only:

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

Hard-signal `category=other` review stays aggregate-only:

- `rows=3`
- `top_signal_sets`: `API=2`, `API,schema.org=1`
- `score_buckets`: `0_24=3`, `25_39=0`, `40_59=0`, `60_plus=0`

## Decision

Decision: `no_op_fixed_point`.

Matrix:

- `taxonomy_rule_change=false`
- `threshold_adjustment=false`
- `no_op_fixed_point=true`

Reason: the current aggregate `category=other` low-signal state does not prove a narrow taxonomy rule. The low-signal cohorts lack API, OpenAPI, MCP, or ai-plugin evidence and stay audit-only. Threshold changes would weaken the hard-agent-signal boundary, so there is no threshold adjustment.

## Guard State

The following cohorts remain audit-only unless a hard agent signal is proven by an executor-only path:

- `llms_only`: `public_search=false`, `score_fix_targeting=false`
- `schema_only`: `public_search=false`, `score_fix_targeting=false`
- `zero_score`: `public_search=false`, `score_fix_targeting=false`
- `category_other_low_signal`: `public_search=false`, `score_fix_targeting=false`

Public search remains protected by `models.AgentFirstFilter`. Score-fix targeting still requires a hard agent signal.

## Queue Decision

No new discovery-quality follow-up was added to `harness/generated-work-items.json` for this lane because the refreshed aggregate is a true fixed point: no taxonomy-rule change, no threshold adjustment, and no public or score-fix eligibility change.

## Verification

- `python3 -m json.tool harness/discovery-quality-latest.json`
- `python3 -m json.tool harness/discovery-quarantine-latest.json`
- `python3 -m json.tool harness/generated-work-items.json`
- `python3 -m unittest tools/test-discovery-quality-report.py tools/test-discovery-quarantine-report.py tools/quality-gate-discovery-test.py`
