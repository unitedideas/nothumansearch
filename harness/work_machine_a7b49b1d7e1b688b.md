# Discovery Quarantine Refresh

Work item: `work_machine_a7b49b1d7e1b688b`
Observed at: 2026-06-12T03:11:02Z

## Inputs

- Used `./tools/refresh-discovery-quality.sh`.
- The wrapper selected the bounded aggregate seed-refresh input and regenerated:
  - `harness/discovery-quality-latest.json`
  - `harness/discovery-quarantine-latest.json`
  - `harness/discovery-quarantine-history.jsonl`
- No full recrawl, broad crawl, browser automation, public posting, or production mutation was started.
- Committed artifacts remain aggregate-only. They contain no raw domains, URLs, row IDs, descriptions, emails, tokens, or private notes.

## Aggregate Result

- `sample_rows`: 12745
- `hard_signal_rows`: 9208
- `low_signal_rows`: 3537
- `category_other_low_signal`: 243
- `category_other_hard_agent_signal`: 80
- `llms_only`: 625
- `schema_only`: 647
- `zero_score`: 1390
- `passive_only_share`: 0.2775
- `quarantine_active`: true

## Decision

The post-recrawl `category=other` state is a **no-op fixed point** for this work item.

- `taxonomy_rule_change`: false
- `threshold_adjustment`: false
- `no_op_fixed_point`: true

Reason: aggregate-only evidence still does not prove a narrow taxonomy rule. The hard-signal `category=other` cohort is small and only summarized by signal-set counts. The low-signal `category=other` cohort lacks proven hard agent signals and stays audit-only.

## Guard State

These cohorts remain audit-only:

- `llms_only`: `public_search=false`, `score_fix_targeting=false`
- `schema_only`: `public_search=false`, `score_fix_targeting=false`
- `zero_score`: `public_search=false`, `score_fix_targeting=false`
- `category_other_low_signal`: `public_search=false`, `score_fix_targeting=false`

Promotion requires a proven hard agent signal through the existing `AgentFirstFilter` and `HasHardAgentSignal` boundaries.

## Follow-Up

Updated `harness/generated-work-items.json` with bounded aggregate-only follow-ups. The next useful work is a redacted/aggregate hard-signal `category=other` taxonomy review, not another full recrawl.

## Commit Blocker

Commit was attempted, but this runner cannot write the Git index:

`fatal: Unable to create '/Users/owlassist/foundry-businesses/nothumansearch/.git/index.lock': Operation not permitted`

Observed permission checks returned `git_dir_not_writable` and `index_not_writable`. The repo files were updated locally, but the commit must be made by a runner with writable `.git` metadata.
