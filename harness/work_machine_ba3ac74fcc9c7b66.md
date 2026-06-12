# Discovery-quality aggregate refresh closeout - 2026-06-12

WorkItem: `work_machine_ba3ac74fcc9c7b66`

No full recrawl, broad crawl, deploy, browser automation, desktop automation, public action, production data deletion, private row inspection, process-environment inspection, or secret read was performed.

## Source Boundary

The refresh used only the completed 2026-05-19 full-recrawl aggregate summary lines from `tools/recrawl.log`, written temporarily to `/tmp/nhs-2026-05-19-recrawl-summary.log` and removed after artifact generation.

Observed aggregate boundary:

- `sample_rows=9827`
- `hard_signal_rows=4114`
- `low_signal_rows=5713`
- `category_other_hard_agent_signal=752`
- `category_other_low_signal=2466`

No raw domains, URLs, row IDs, descriptions, emails, tokens, private notes, or crawler row output were committed.

## Artifact State

`harness/discovery-quality-latest.json`, `harness/discovery-quarantine-latest.json`, and `harness/discovery-quarantine-history.jsonl` were re-run through the aggregate helper path for the May 19 boundary. The committed artifacts already matched the refreshed output, so this ledger is the durable state change for the worker run.

## Decision

Post-recrawl `category=other` state remains a no-op fixed point.

- `taxonomy_rule_change=false`
- `threshold_adjustment=false`
- `no_op_fixed_point=true`

Low-signal `category=other` rows lack hard agent signals by definition. Aggregate counts alone do not prove a narrow taxonomy rule, and a threshold change would promote passive discovery rows rather than proving agent usability.

## Guard State

The following cohorts remain audit-only unless a hard agent signal is proven:

- `llms_only`: `public_search=false`, `score_fix_targeting=false`
- `schema_only`: `public_search=false`, `score_fix_targeting=false`
- `zero_score`: `public_search=false`, `score_fix_targeting=false`
- `category_other_low_signal`: `public_search=false`, `score_fix_targeting=false`

Public search remains protected by `models.AgentFirstFilter`. Score-fix targeting remains gated on `HasHardAgentSignal`.

## Follow-up State

`harness/generated-work-items.json` was not changed because this discovery-quality lane is at a true fixed point for passive and low-signal cohorts. No replacement recrawl, broad crawl, taxonomy-rule change, threshold adjustment, public-search promotion, score-fix targeting, browser task, deploy task, public action, or private-row follow-up is warranted from this aggregate cohort.
