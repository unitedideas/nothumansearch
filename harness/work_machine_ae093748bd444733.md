# NHS discovery-quality aggregate refresh - 2026-06-12

Closeout for QLimit WorkItem `work_machine_ae093748bd444733`.

No full recrawl, broad crawl, deploy, browser automation, desktop automation, public action, production data deletion, process-environment inspection, or secret read was performed.

## Inputs

- The originating handoff records the completed 2026-05-19 full-recrawl boundary:
  - start: `2026-05-19 06:00:11`
  - preflight: `api_status=200`, `api_ok=1`
  - workers: `10`
  - completion: `2026-05-19 10:26:32`
  - post-run: `api_status=200`, `api_ok=1`, `dry_run=0`
  - aggregate crawler result: `Success=9830`, `Failed=396`, `Total=10226`
- Existing history already retained a sanitized post-2026-05-19 aggregate row for week `2026-05-18` without domains, URLs, row ids, descriptions, emails, tokens, or private notes.
- This run refreshed the current aggregate artifacts with:
  - `NHS_DISCOVERY_INPUT=tools/seed-refresh.log ./tools/refresh-discovery-quality.sh`
  - output: `discovery_quality_refresh hard_signal_rows=9548 low_signal_rows=3670 category_other_low_signal=252 quarantine_active=true planner_priority=quarantine_first history=harness/discovery-quarantine-history.jsonl`

## Refreshed Discovery Aggregates

- `sample_rows=13218`
- `hard_signal_rows=9548`
- `low_signal_rows=3670`
- `hard_signal_rate=0.7223`
- `category_other=335`
- `category_other_hard_agent_signal=83`
- `category_other_low_signal=252`
- `llms_only=649`
- `schema_only=673`
- `zero_score=1439`
- `passive_or_soft_signal=909`

`hard_signal_other_review` remains aggregate-only:

- `rows=83`
- `score_bucket_0_24=83`
- `top_signal_sets`: `API=56`, `API,schema.org=27`

## Decision

The post-recrawl `category=other` state remains a no-op fixed point for this lane.

- `taxonomy_rule_change=false`
- `threshold_adjustment=false`
- `no_op_fixed_point=true`

Reason: the `category=other` low-signal cohort lacks hard agent signals by definition. Aggregate counts alone do not prove a narrow taxonomy rule, and lowering thresholds would only promote passive or soft-signal rows into surfaces that require hard agent evidence. The hard-signal `category=other` lane remains aggregate-only unless a future bounded sampler proves a specific taxonomy rule without committing row-level evidence.

## Guard State

The following cohorts remain audit-only unless a hard agent signal is proven:

- `llms_only`: `public_search=false`, `score_fix_targeting=false`
- `schema_only`: `public_search=false`, `score_fix_targeting=false`
- `zero_score`: `public_search=false`, `score_fix_targeting=false`
- `category_other_low_signal`: `public_search=false`, `score_fix_targeting=false`

Public search remains guarded by `models.AgentFirstFilter`. Score-fix targeting remains guarded by `HasHardAgentSignal`.

## Follow-up State

No replacement full-recrawl, broad crawl, taxonomy-rule-change, threshold-adjustment, public-search, or score-fix-targeting follow-up is warranted from this lane.

`harness/generated-work-items.json` already contains three concrete non-fixed-point business follow-ups for credential-capable monitor review, private score-fix cleanup, and API-key subscribe HTML handoff. It is intentionally unchanged for this WorkItem because the discovery-quality quarantine lane itself is at a fixed point.
