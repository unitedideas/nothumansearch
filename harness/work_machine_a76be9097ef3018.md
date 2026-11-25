# NHS Full Recrawl Boundary Closeout - 2026-06-12

QLimit WorkItem: `work_machine_a76be9097ef3018`

No replacement full recrawl, deploy, browser automation, desktop automation, public action, production data deletion, private row inspection, process-environment inspection, secret read, or ad hoc lock deletion was performed.

## Boundary

The source WorkItem referenced the 2026-05-21 full-recrawl boundary as active from an earlier observation. Repo-local aggregate evidence now proves that boundary closed and later boundaries have also closed.

2026-05-21 boundary evidence:

- `tools/recrawl-health.log`: seed refresh completed at `2026-05-21 05:42:37` with `api_status=200`, `api_ok=1`, `workers=10`, `dry_run=0`.
- `tools/recrawl.log`: seed refresh aggregate was `success=469`, `failed=14`, `total=483`.
- `tools/recrawl-health.log`: full recrawl completed at `2026-05-21 10:25:48` with `api_status=200`, `api_ok=1`, `workers=10`, `dry_run=0`.
- `tools/recrawl.log`: full recrawl aggregate was `success=9847`, `failed=389`, `total=10236`.
- `tools/full-recrawl.lock`: absent at this closeout.

Latest completed boundary evidence:

- `tools/recrawl-health.log`: seed refresh completed at `2026-06-11 05:39:33` with `api_status=200`, `api_ok=1`, `workers=10`, `dry_run=0`.
- `tools/recrawl-health.log`: full recrawl started at `2026-06-11 06:00:08`.
- `tools/recrawl-health.log`: full recrawl completed at `2026-06-11 10:58:31` with `api_status=200`, `api_ok=1`, `workers=10`, `dry_run=0`.
- `tools/recrawl.log`: full recrawl aggregate was `success=10065`, `failed=440`, `total=10505`.
- `tools/recrawl.log`: completion line exists: `2026-06-11 10:58:31 NHS full_recrawl complete`.
- `tools/full-recrawl.lock`: absent at this closeout.

## Public Aggregate Status

Direct public aggregate probes from this runner failed closed because local DNS could not resolve `nothumansearch.ai`.

- `curl -fsS https://nothumansearch.ai/api/v1/stats`: `Could not resolve host`.
- `curl -fsS https://nothumansearch.ai/api/v1/categories`: `Could not resolve host`.

No live public stats or category counts were invented from this failed probe. The latest repo-local wrapper postflight remains the API status evidence: `api_status=200`, `api_ok=1`.

## Discovery Quality Refresh

The discovery-quality aggregate was refreshed from a temporary bounded slice of `tools/recrawl.log` covering lines `573284..594390`, the completed 2026-06-11 full-recrawl boundary. The temporary slice was removed after the helper completed.

Refresh command:

- `NHS_DISCOVERY_INPUT=/tmp/nhs-recrawl-20260611.*.log ./tools/refresh-discovery-quality.sh`

Refresh output:

- `hard_signal_rows=4206`
- `low_signal_rows=5855`
- `category_other_low_signal=2580`
- `quarantine_active=true`
- `planner_priority=quarantine_first`

Latest refreshed aggregate artifact state:

- `sample_rows=10061`
- `hard_signal_rate=0.418`
- `category_other=3362`
- `category_other_hard_agent_signal=782`
- `category_other_low_signal=2580`
- `llms_only=998`
- `schema_only=525`
- `zero_score=2353`
- `passive_or_soft_signal=1979`

`hard_signal_other_review` remains aggregate-only:

- `rows=782`
- score buckets: `0_24=434`, `25_39=30`, `40_59=306`, `60_plus=12`
- top signal sets are aggregate counts only and contain no domains or URLs.

## Decision

The 2026-05-21 full-recrawl boundary is not active. The latest 2026-06-11 full-recrawl boundary is also closed by wrapper completion evidence.

Discovery-quality follow-up remains active but aggregate-only:

- `taxonomy_rule_change=false`
- `threshold_adjustment=false`
- `category_other_low_signal=aggregate_review_only`
- `public_search=false`
- `score_fix_targeting=false`

Reason: low-signal rows still outnumber hard-signal rows, but aggregate counts alone do not prove a narrow taxonomy rule. Rows without API, OpenAPI, MCP, or ai-plugin remain audit-only under the existing `AgentFirstFilter` and score-fix hard-signal gate.
