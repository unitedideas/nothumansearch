# NHS May 21 Full Recrawl Closeout Refresh

WorkItem: `work_machine_599575d7c92c15fd`

No full recrawl, broad crawl, deploy, browser automation, desktop automation, public action, production data deletion, private row inspection, process-environment inspection, or secret read was performed.

## Aggregate Boundary Proof

Completed 2026-05-21 full-recrawl boundary, from repo-local wrapper logs:

- Seed refresh completed at `2026-05-21 05:42:37` with `api_status=200`, `api_ok=1`, `workers=10`, `dry_run=0`.
- Seed refresh aggregate result was `success=469`, `failed=14`, `total=483`.
- Full recrawl started at `2026-05-21 06:00:12` with preflight `api_status=200`, `api_ok=1`, `health_outcome=full_pressure`, `workers=10`.
- Full recrawl remote command was `/app/crawler -recrawl -workers 10`.
- Full recrawl completed at `2026-05-21 10:25:48` with post-run `api_status=200`, `api_ok=1`, `workers=10`, `dry_run=0`.
- Full recrawl aggregate result was `success=9847`, `failed=389`, `total=10236`.
- `tools/full-recrawl.lock` and `tools/recrawl.lock` were absent during this closeout refresh.

Planner-provided aggregate after the completed boundary:

- `total_sites=4171`
- `avg_score=35`
- `top_category=developer`
- `category_other=777`
- `spam=1`
- 2026-05-18 discovery found `0` new domains.

## Refreshed Discovery Artifacts

Command:

- `NHS_DISCOVERY_INPUT=tools/seed-refresh.log ./tools/refresh-discovery-quality.sh`

Sanitized helper output:

- `hard_signal_rows=8866`
- `low_signal_rows=3405`
- `category_other_low_signal=234`
- `quarantine_active=true`
- `planner_priority=quarantine_first`

Current aggregate artifact state:

- `sample_rows=12271`
- `hard_signal_rate=0.7225`
- `category_other=311`
- `category_other_hard_agent_signal=77`
- `category_other_low_signal=234`
- `llms_only=601`
- `schema_only=621`
- `zero_score=1342`
- `passive_or_soft_signal=841`

The refreshed artifacts are:

- `harness/discovery-quality-latest.json`
- `harness/discovery-quarantine-latest.json`
- `harness/discovery-quarantine-history.jsonl`

## Decision

Current `category=other` state is a no-op fixed point for this lane.

- `taxonomy_rule_change=false`
- `threshold_adjustment=false`
- `no_op_fixed_point=true`
- `category_other_low_signal=no_op_fixed_point`
- `taxonomy_rule=not_from_low_signal_aggregate_cohort`
- `threshold_adjustment=none`

Reason: low-signal `category=other` rows lack hard agent signals by definition. Aggregate counts alone are not evidence for a narrow taxonomy rule, and threshold changes would promote passive discovery rows rather than proving agent usability.

## Guard State

The following cohorts remain audit-only unless a hard agent signal is proven:

- `llms_only`: `public_search=false`, `score_fix_targeting=false`
- `schema_only`: `public_search=false`, `score_fix_targeting=false`
- `zero_score`: `public_search=false`, `score_fix_targeting=false`
- `category_other_low_signal`: `public_search=false`, `score_fix_targeting=false`

Public search remains protected by `models.AgentFirstFilter`. Score-fix targeting remains guarded by `HasHardAgentSignal`.

## Follow-up State

`harness/generated-work-items.json` is intentionally unchanged. This WorkItem is at a true fixed point and does not justify a replacement recrawl, broad crawl, taxonomy-rule change, threshold adjustment, public-search promotion, score-fix targeting, public action, browser action, deploy, or private-row review.
