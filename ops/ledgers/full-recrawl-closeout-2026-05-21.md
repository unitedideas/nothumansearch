# NHS Full Recrawl Closeout - 2026-05-21

Scope: aggregate-only closeout for QLimit WorkItem `work_machine_2355ca57b79a710b`.

No full recrawl, broad crawl, deploy, browser automation, desktop automation, public action, production data deletion, private row inspection, process-environment inspection, or secret read was performed.

## Boundary Proof

Seed refresh wrapper:

- `tools/recrawl-health.log` records start at `2026-05-21 05:30:09`.
- Preflight health was `api_status=200`, `api_ok=1`.
- Remote command was `/app/crawler -seed -workers 10`.
- Completion was `2026-05-21 05:42:37` with `api_status=200`, `api_ok=1`, `workers=10`, `dry_run=0`.

Seed refresh aggregate:

- `success=469`
- `failed=14`
- `total=483`
- Completion line: `2026-05-21 05:42:37 NHS seed_refresh complete`

Full recrawl wrapper:

- `tools/recrawl-health.log` records start at `2026-05-21 06:00:12`.
- Preflight health was `api_status=200`, `api_ok=1`.
- Health outcome was `full_pressure`, `workers=10`.
- Remote command was `/app/crawler -recrawl -workers 10`.
- Completion was `2026-05-21 10:25:48` with `api_status=200`, `api_ok=1`, `workers=10`, `dry_run=0`.
- `tools/full-recrawl.lock` and `tools/recrawl.lock` were absent at closeout.

Full recrawl aggregate:

- `success=9847`
- `failed=389`
- `total=10236`
- Completion line: `2026-05-21 10:25:48 NHS full_recrawl complete`

Planner-provided public aggregate after the completed boundary:

- `total_sites=4171`
- `avg_score=35`
- `top_category=developer`
- `category_other=777`
- `spam=1`
- 2026-05-18 discovery found `0` new domains.

## Discovery Quality Refresh

Refresh command:

- `NHS_DISCOVERY_INPUT=tools/seed-refresh.log ./tools/refresh-discovery-quality.sh`

Refresh output:

- `hard_signal_rows=8526`
- `low_signal_rows=3273`
- `category_other_low_signal=225`
- `quarantine_active=true`
- `planner_priority=quarantine_first`

Refreshed aggregate artifact state:

- `sample_rows=11799`
- `hard_signal_rate=0.7226`
- `category_other=299`
- `category_other_hard_agent_signal=74`
- `category_other_low_signal=225`
- `llms_only=578`
- `schema_only=596`
- `zero_score=1292`
- `passive_or_soft_signal=807`

`hard_signal_other_review` remains aggregate-only. Executor-only samples must not enter committed planner artifacts.

## Decision

Current `category=other` state is a no-op fixed point for this lane.

- `taxonomy_rule_change=false`
- `threshold_adjustment=false`
- `no_op_fixed_point=true`
- `category_other_low_signal=no_op_fixed_point`
- `taxonomy_rule=not_from_low_signal_aggregate_cohort`
- `threshold_adjustment=none`

Reason: low-signal `category=other` rows lack hard agent signals by definition. Aggregate counts alone are not enough evidence for a narrow taxonomy rule, and changing thresholds would promote passive discovery rows rather than proving agent usability.

## Guard State

The following cohorts remain audit-only unless a hard agent signal is proven:

- `llms_only`: `public_search=false`, `score_fix_targeting=false`
- `schema_only`: `public_search=false`, `score_fix_targeting=false`
- `zero_score`: `public_search=false`, `score_fix_targeting=false`
- `category_other_low_signal`: `public_search=false`, `score_fix_targeting=false`

Public search remains protected by `models.AgentFirstFilter`. Score-fix targeting remains gated on `HasHardAgentSignal`.

## Follow-up State

`harness/generated-work-items.json` is intentionally unchanged for this WorkItem. This lane is at a true fixed point: the completed May 21 recrawl is closed, discovery-quality artifacts were refreshed from bounded aggregate helper output, and the current low-signal `category=other` state does not justify a taxonomy-rule change, threshold adjustment, public-search promotion, score-fix targeting, or replacement crawl.
