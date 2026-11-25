# NHS discovery-quality fixed point - 2026-06-13

Aggregate-only closeout for QLimit WorkItem `work_machine_fd806e0bcdaa9319`.

No full recrawl, broad crawl, deploy, browser automation, desktop automation, public action, production data deletion, secret read, private row export, raw candidate export, or row-level sampler artifact was performed for this WorkItem.

## Inputs

- The originating handoff cited the completed 2026-05-19 full-recrawl boundary:
  - start: `2026-05-19 06:00:11`
  - completion: `2026-05-19 10:26:32`
  - preflight and post-run API status: `200`
  - workers: `10`
  - aggregate crawler result: `Success=9830`, `Failed=396`, `Total=10226`
- The repo README requires completed full-recrawl aggregate refreshes to run through `tools/discovery-quality-report.py` with an explicit `--since` / `--until` window, then rebuild quarantine from the sanitized quality artifact.
- The 2026-05-19 crawler rows are UTC-stamped in `tools/recrawl.log`, so the aggregate refresh used the bounded crawler-row window `2026/05/19 13:00:11` through `2026/05/19 17:26:32`.

## Refreshed Aggregate Artifacts

Refreshed through sanitized aggregate helpers only:

```bash
python3 tools/discovery-quality-report.py --input tools/recrawl.log --since '2026/05/19 13:00:11' --until '2026/05/19 17:26:32' --output harness/discovery-quality-latest.json
python3 tools/discovery-quarantine-report.py --input harness/discovery-quality-latest.json --output harness/discovery-quarantine-latest.json --history-output harness/discovery-quarantine-history.jsonl --observed-at 2026-05-19T17:26:32Z
python3 tools/quality-gate-discovery.py --quarantine harness/discovery-quarantine-latest.json --repo-root .
```

The refreshed aggregate state is:

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
- `planner_priority=quarantine_first`

`hard_signal_other_review` remains aggregate-only:

- `rows=752`
- `score_bucket_0_24=431`
- `top_signal_sets`: `API=305`, `API,llms.txt=144`, `API,llms.txt,schema.org=86`, `API,schema.org=72`, `AI-bots,API=38`, `MCP,llms.txt,schema.org=28`, `AI-bots,API,llms.txt=10`, `API,OpenAPI=9`

## Decision

Post-recrawl `category=other` state is a no-op fixed point for this lane.

- `taxonomy_rule_change=false`
- `threshold_adjustment=false`
- `no_op_fixed_point=true`
- `category_other_low_signal=no_op_fixed_point`
- `taxonomy_rule=not_from_low_signal_aggregate_cohort`
- `threshold_adjustment=none`

Reason: the low-signal `category=other` cohort lacks hard agent signals by definition, and aggregate counts alone are not evidence for a narrow taxonomy rule. Threshold adjustment would promote passive or soft discovery rows rather than proving agent usability. Taxonomy-rule work remains reserved for hard-signal `category=other` rows only, with any future proof committed as aggregate evidence.

## Guard State

The following cohorts remain audit-only unless a hard agent signal is proven:

- `llms_only`: `public_search=false`, `score_fix_targeting=false`
- `schema_only`: `public_search=false`, `score_fix_targeting=false`
- `zero_score`: `public_search=false`, `score_fix_targeting=false`
- `category_other_low_signal`: `public_search=false`, `score_fix_targeting=false`

Public search remains protected by `models.AgentFirstFilter`. Score-fix targeting remains gated on `HasHardAgentSignal`.

## Follow-up State

No replacement full-recrawl, broad crawl, taxonomy-rule-change, threshold-adjustment, public-search, score-fix-targeting, deploy, browser, or public-action follow-up is warranted from this lane.

`harness/generated-work-items.json` is intentionally unchanged. This discovery-quality lane is at a true fixed point for passive and low-signal cohorts after the May 19 recrawl.
