# Seed Refresh Aggregate Handoff - 2026-05-12

Source: `tools/seed-refresh.log`

Policy: aggregate-only. This ledger must not include domains, URLs, row IDs, names, descriptions, emails, tokens, or review notes.

## Bounded Refresh

- `sample_rows`: 1881
- `hard_signal_rows`: 1363
- `low_signal_rows`: 518
- `passive_only_share`: 0.2754
- `llms_only`: 92
- `schema_only`: 96
- `zero_score`: 207
- `category_other_low_signal`: 36
- `category_other_hard_agent_signal`: 12
- `quality_gate.status`: review
- `quality_gate.trigger`: category_other_low_signal_exceeds_hard_signal

## Handoff

- `business_local_handoff.required`: true
- `business_local_handoff.kind`: bounded_aggregate_review
- `business_local_handoff.public`: false
- `business_local_handoff.domain_output`: false
- `bounded_action`: write business-local aggregate handoff row; do not trigger broad crawl

## Guard State

- Public search remains protected by `models.AgentFirstFilter`.
- Score-fix targeting remains hard-signal-only through `HasHardAgentSignal`.
- `llms_only`, `schema_only`, `zero_score`, and `category_other_low_signal` remain audit-only and excluded from public search and score-fix targeting.

## Verification

- `python3 -m unittest tools/test-discovery-quality-report.py tools/test-discovery-quarantine-report.py tools/quality-gate-discovery-test.py`
- `NHS_DISCOVERY_INPUT=tools/seed-refresh.log ./tools/refresh-discovery-quality.sh`

No full manual recrawl was run.
