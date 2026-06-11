# NHS discovery-quality post-recrawl closeout - 2026-06-11

Scope: aggregate-only closeout for QLimit WorkItem `work_machine_813aa97e4b4ff1e9`.

No full recrawl, broad crawl, deploy, browser automation, desktop automation, public action, production data deletion, private row inspection, process-environment inspection, or secret read was performed.

## Boundary

This WorkItem follows the already-completed 2026-05-19 full-recrawl boundary recorded by prior local closeout evidence:

- full recrawl start: `2026-05-19 06:00:11`
- preflight: `api_status=200`, `api_ok=1`
- health outcome: `full_pressure`
- workers: `10`
- full recrawl completion: `2026-05-19 10:26:32`
- post-run: `api_status=200`, `api_ok=1`, `dry_run=0`
- aggregate crawler result: `Success=9830`, `Failed=396`, `Total=10226`

This run did not reopen that recrawl boundary.

## Refresh

Bounded helper used:

- `NHS_DISCOVERY_INPUT=tools/seed-refresh.log ./tools/refresh-discovery-quality.sh`

Refresh output:

- `hard_signal_rows=8866`
- `low_signal_rows=3405`
- `category_other_low_signal=234`
- `quarantine_active=true`
- `planner_priority=quarantine_first`
- `history=harness/discovery-quarantine-history.jsonl`

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

`hard_signal_other_review` remains aggregate-only:

- `rows=77`
- `score_bucket_0_24=77`
- `top_signal_sets`: `API=52`, `API,schema.org=25`

Targeted sampler:

- `python3 tools/taxonomy-other-redacted-sample.py --limit 50`
- result: `sample_status=failed`, `reason=URLError`
- artifact policy confirmed no raw fields, domains, URLs, row ids, names, or descriptions were output.

## Decision

Post-recrawl `category=other` state is a no-op fixed point for this lane.

- `taxonomy_rule_change=false`
- `threshold_adjustment=false`
- `no_op_fixed_point=true`
- `category_other_low_signal=no_op_fixed_point`
- `taxonomy_rule=not_from_low_signal_aggregate_cohort`
- `threshold_adjustment=none`

Reason: the low-signal `category=other` cohort lacks hard agent signals by definition. Aggregate counts alone do not prove a narrow taxonomy rule, and threshold changes would promote passive discovery rows instead of proving agent usability.

## Guard State

The following cohorts remain audit-only unless a hard agent signal is proven:

- `llms_only`: `public_search=false`, `score_fix_targeting=false`
- `schema_only`: `public_search=false`, `score_fix_targeting=false`
- `zero_score`: `public_search=false`, `score_fix_targeting=false`
- `category_other_low_signal`: `public_search=false`, `score_fix_targeting=false`

Public search remains protected by `models.AgentFirstFilter`. Score-fix targeting remains gated on `HasHardAgentSignal`.

## Follow-up State

`harness/generated-work-items.json` is intentionally unchanged for this WorkItem. This discovery-quality lane is at a true fixed point: the artifacts are refreshed through the bounded aggregate helper path, the low-signal cohorts remain audit-only, and there is no warranted taxonomy-rule-change, threshold-adjustment, public-search, score-fix-targeting, broad-crawl, full-recrawl, browser, deploy, or private-row follow-up from this lane.
