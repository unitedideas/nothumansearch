# Discovery Quarantine Refresh Closeout

WorkItem: `work_machine_adf9b50adc289aa8`
Observed: 2026-06-12

## Action

Ran `./tools/refresh-discovery-quality.sh`, which selected the bounded aggregate seed-refresh log and regenerated:

- `harness/discovery-quality-latest.json`
- `harness/discovery-quarantine-latest.json`
- `harness/discovery-quarantine-history.jsonl`

The refresh emitted aggregate counts only:

- `hard_signal_rows=9208`
- `low_signal_rows=3537`
- `category_other_low_signal=243`
- `quarantine_active=true`
- `planner_priority=quarantine_first`

No full recrawl, broad crawl, row sampler, raw domain export, URL export, row ID export, email export, token read, or private-note read was started.

## Decision

The post-recrawl `category=other` state remains a **no-op fixed point**.

This is not a taxonomy-rule change because the aggregate cohort does not prove a narrow category rule. It is not a threshold adjustment because low-signal rows still lack hard agent signals. `category=other` low-signal rows remain audit-only.

## Guards

The generated quarantine artifact keeps these cohorts out of public search and score-fix targeting unless a hard agent signal is proven:

- `llms_only`
- `schema_only`
- `zero_score`
- `category_other_low_signal`

The public guards remain:

- `public_search`: `protected_by_models.AgentFirstFilter`
- `score_fix_targeting`: `requires_has_hard_agent_signal`

## Result

The regenerated artifacts were byte-identical to the committed state, so the committed state already reflected the current bounded aggregate refresh.
