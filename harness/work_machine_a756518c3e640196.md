# NHS discovery-quality fixed point - 2026-06-11

QLimit WorkItem: `work_machine_a756518c3e640196`

No full recrawl, broad crawl, deploy, browser automation, desktop automation, public action, production data deletion, private row inspection, process-environment inspection, or secret read was performed.

## Boundary

The refresh used the completed 2026-05-19 full-recrawl boundary already recorded in repo-local logs:

- start: `2026-05-19 06:00:11`
- completion: `2026-05-19 10:26:32`
- preflight/post-run API status: `api_status=200`, `api_ok=1`
- outcome: `Success=9830`, `Failed=396`, `Total=10226`
- lock check from source WorkItem: no `tools/full-recrawl.lock` or `tools/recrawl.lock`

The aggregate refresh was run against a temporary bounded slice of `tools/recrawl.log` for that run only, then the temporary file was removed. No raw domains, URLs, row IDs, descriptions, emails, tokens, private notes, or candidate row output were committed.

## Aggregate Artifact State

`harness/discovery-quality-latest.json`, `harness/discovery-quarantine-latest.json`, and `harness/discovery-quarantine-history.jsonl` were refreshed through the aggregate-only helper. The latest tracked artifacts already represented the 2026-05-19 recrawl state, so no JSON content changed in this closeout.

Current sanitized aggregate counts:

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

`hard_signal_other_review` remains aggregate-only:

- `rows=752`
- score buckets: `0_24=431`, `25_39=29`, `40_59=288`, `60_plus=4`
- top signal sets are aggregate counts only and contain no domains or URLs.

## Decision

Post-recrawl `category=other` state is a no-op fixed point for this lane.

- `taxonomy_rule_change=false`
- `threshold_adjustment=false`
- `no_op_fixed_point=true`
- `category_other_low_signal=no_op_fixed_point`
- `taxonomy_rule=not_from_low_signal_aggregate_cohort`
- `threshold_adjustment=none`

Reason: low-signal `category=other` rows lack hard agent signals by definition. Aggregate counts alone are not evidence for a narrow taxonomy rule, and threshold changes would promote passive discovery rows rather than proving agent usability. A taxonomy change would require a separate hard-signal-only executor review with sanitized proof and crawler tests.

The optional targeted sampler failed closed with a sanitized `URLError` artifact and emitted no raw fields. That does not change the decision because this WorkItem can be completed from the bounded aggregate helper output.

## Guard State

The following cohorts remain audit-only unless a hard agent signal is proven:

- `llms_only`: `public_search=false`, `score_fix_targeting=false`
- `schema_only`: `public_search=false`, `score_fix_targeting=false`
- `zero_score`: `public_search=false`, `score_fix_targeting=false`
- `category_other_low_signal`: `public_search=false`, `score_fix_targeting=false`

Public search remains protected by `models.AgentFirstFilter`. Score-fix targeting remains gated on `HasHardAgentSignal`.

## Follow-up State

`harness/generated-work-items.json` is intentionally unchanged. This discovery-quality lane is at a true fixed point for passive and low-signal cohorts after the 2026-05-19 recrawl: no replacement recrawl, broad crawl, taxonomy-rule change, threshold adjustment, public-search promotion, score-fix targeting, browser task, deploy task, public action, or private-row follow-up is warranted from these cohorts.
