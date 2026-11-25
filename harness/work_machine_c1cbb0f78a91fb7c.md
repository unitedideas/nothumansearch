# Discovery Quality Aggregate Refresh

WorkItem: `work_machine_c1cbb0f78a91fb7c`
Observed at: 2026-06-12T22:11:17Z

## Boundary

- No local full-recrawl, recrawl, or seed-refresh lock was present before refresh.
- Used the repo aggregate helper: `./tools/refresh-discovery-quality.sh`.
- Input path selected by the helper was the newest bounded aggregate log, not a broad crawl command.
- No full recrawl, broad crawl, browser automation, public post, email, spend, account creation, or production data mutation was run.

## Aggregate Output

- `hard_signal_rows`: 9548
- `low_signal_rows`: 3670
- `category_other_low_signal`: 252
- `category_other_hard_agent_signal`: 83
- `sample_rows`: 13218
- `passive_only_share`: 0.2777
- `quarantine_active`: true
- `planner_priority`: quarantine_first

## Decision

Current `category=other` state remains a no-op fixed point for taxonomy and threshold purposes.

- `taxonomy_rule_change`: false
- `threshold_adjustment`: false
- `no_op_fixed_point`: true

Reason: aggregate `category=other` low-signal rows do not prove a narrow taxonomy rule. The low-signal cohorts remain audit-only unless a hard agent signal is proven.

## Guard State

- `llms_only`: `public_search=false`, `score_fix_targeting=false`
- `schema_only`: `public_search=false`, `score_fix_targeting=false`
- `zero_score`: `public_search=false`, `score_fix_targeting=false`
- `category_other_low_signal`: `public_search=false`, `score_fix_targeting=false`

## Updated Artifacts

- `harness/discovery-quality-latest.json`
- `harness/discovery-quarantine-latest.json`
- `harness/discovery-quarantine-history.jsonl`

All committed proof is aggregate-only. It excludes raw domains, URLs, row IDs, descriptions, emails, tokens, private notes, and crawler row output.
