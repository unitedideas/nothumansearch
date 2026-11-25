# NHS discovery-quality closeout - work_machine_cdee4533b0035233

Aggregate-only closeout for QLimit WorkItem `work_machine_cdee4533b0035233`.

No full recrawl, broad crawl, deploy, browser automation, desktop automation, public action, production data deletion, private row inspection, process-environment inspection, or secret read was performed.

## Boundary

The requested boundary is the completed 2026-05-19 full recrawl:

- `tools/recrawl-health.log`: start `2026-05-19 06:00:11`, preflight `api_status=200`, `api_ok=1`, `health_outcome=full_pressure`, `workers=10`, remote start, and completion `2026-05-19 10:26:32` with post-run `api_status=200`, `api_ok=1`, `dry_run=0`.
- `tools/recrawl.log`: `Success=9830`, `Failed=396`, `Total=10226`, completion `2026-05-19 10:26:32 NHS full_recrawl complete`.
- Repo-local lock check found no `tools/full-recrawl.lock`, `tools/recrawl.lock`, or `tools/seed-refresh.lock`.

The refresh used only the bounded aggregate parser over `tools/recrawl.log`; no raw domains, URLs, row ids, descriptions, emails, tokens, private notes, or crawler row output were committed.

## Refreshed Artifacts

`harness/discovery-quality-latest.json`, `harness/discovery-quarantine-latest.json`, and `harness/discovery-quarantine-history.jsonl` already contained the same sanitized May 19 aggregate state after regeneration:

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

## Decision

Post-recrawl `category=other` state is a no-op fixed point for this lane.

- `taxonomy_rule_change=false`
- `threshold_adjustment=false`
- `no_op_fixed_point=true`
- `category_other_low_signal=no_op_fixed_point`
- `taxonomy_rule=not_from_low_signal_aggregate_cohort`
- `threshold_adjustment=none`

Reason: low-signal `category=other` rows lack hard agent signals by definition. Aggregate counts alone are not enough evidence for a narrow taxonomy rule, and threshold adjustment would promote passive discovery rows rather than prove agent usability.

## Guard State

These cohorts remain audit-only unless a hard agent signal is proven:

- `llms_only`: `public_search=false`, `score_fix_targeting=false`
- `schema_only`: `public_search=false`, `score_fix_targeting=false`
- `zero_score`: `public_search=false`, `score_fix_targeting=false`
- `category_other_low_signal`: `public_search=false`, `score_fix_targeting=false`

Public search remains protected by `models.AgentFirstFilter`. Score-fix targeting remains gated on `HasHardAgentSignal`.

## Follow-up State

`harness/generated-work-items.json` is unchanged because this discovery-quality lane is at a true fixed point for passive and low-signal cohorts after the May 19 recrawl. No replacement recrawl, broad crawl, taxonomy-rule change, threshold adjustment, public-search promotion, score-fix targeting, browser task, deploy task, public action, or private-row follow-up is warranted from these cohorts.
