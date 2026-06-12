# WorkItem closeout: work_machine_b1f06338a035319e

Business: `nothumansearch`

Scope: aggregate-only closeout for the completed 2026-05-21 seed refresh and full recrawl boundary.

No full recrawl, broad crawl, deploy, browser automation, desktop automation, public action, production data deletion, private row inspection, process-environment inspection, or secret read was performed.

## Boundary proof

Seed refresh wrapper:

- start: `2026-05-21 05:30:09`
- preflight: `api_status=200`, `api_ok=1`
- command: `/app/crawler -seed -workers 10`
- completion: `2026-05-21 05:42:37`, `api_status=200`, `api_ok=1`, `workers=10`, `dry_run=0`
- aggregate completion: `success=469`, `failed=14`, `total=483`

Full recrawl wrapper:

- start: `2026-05-21 06:00:12`
- preflight: `api_status=200`, `api_ok=1`
- health outcome: `full_pressure`, `workers=10`
- command: `/app/crawler -recrawl -workers 10`
- completion: `2026-05-21 10:25:48`, `api_status=200`, `api_ok=1`, `workers=10`, `dry_run=0`
- aggregate completion: `success=9847`, `failed=389`, `total=10236`
- lock check: no `tools/full-recrawl.lock` or `tools/recrawl.lock`

Post-boundary aggregate state from the WorkItem:

- `total_sites=4171`
- `avg_score=35`
- `top_category=developer`
- `category_other=777`
- `spam=1`
- 2026-05-18 discovery found `0` new domains.

## Discovery-quality refresh

Command:

- `NHS_DISCOVERY_INPUT=tools/seed-refresh.log ./tools/refresh-discovery-quality.sh`

Refreshed aggregate output:

- `hard_signal_rows=9548`
- `low_signal_rows=3670`
- `category_other_low_signal=252`
- `quarantine_active=true`
- `planner_priority=quarantine_first`

Refreshed sanitized artifact state:

- `sample_rows=13218`
- `hard_signal_rate=0.7223`
- `category_other=335`
- `category_other_hard_agent_signal=83`
- `category_other_low_signal=252`
- `llms_only=649`
- `schema_only=673`
- `zero_score=1439`
- `passive_or_soft_signal=909`

## Decision

Current `category=other` state is a no-op fixed point for this lane.

- `taxonomy_rule_change=false`
- `threshold_adjustment=false`
- `no_op_fixed_point=true`
- `category_other_low_signal=no_op_fixed_point`
- `taxonomy_rule=not_from_low_signal_aggregate_cohort`
- `threshold_adjustment=none`

Reason: low-signal `category=other` rows lack hard agent signals by definition. Aggregate counts alone are not evidence for a narrow taxonomy rule, and threshold changes would promote passive discovery rows rather than proving agent usability.

## Guard state

These cohorts remain audit-only unless a hard agent signal is proven:

- `llms_only`: `public_search=false`, `score_fix_targeting=false`
- `schema_only`: `public_search=false`, `score_fix_targeting=false`
- `zero_score`: `public_search=false`, `score_fix_targeting=false`
- `category_other_low_signal`: `public_search=false`, `score_fix_targeting=false`

Public search remains protected by `models.AgentFirstFilter`. Score-fix targeting remains gated on `HasHardAgentSignal`.

## Follow-up state

`harness/generated-work-items.json` remains unchanged because this discovery-quality lane is at a true fixed point. No recrawl, broad crawl, taxonomy-rule change, threshold adjustment, public-search promotion, or score-fix targeting follow-up is warranted from this aggregate boundary.

Commit attempt failed because the runner cannot write `.git/index.lock` in this sandbox. The worktree proof is present for a commit-capable runner.
