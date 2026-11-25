# Discovery Quarantine Aggregate Refresh

Work item: `work_machine_28f847afb01f2a99`

Date: 2026-06-09 local / 2026-06-10 UTC

Scope: aggregate-only refresh of sanitized discovery-quality and discovery-quarantine artifacts after the completed post-recrawl state. No full recrawl, broad crawl, raw domains, URLs, row ids, descriptions, emails, tokens, private notes, or row-level candidate output were committed.

Command run:
- `NHS_DISCOVERY_INPUT=tools/seed-refresh.log ./tools/refresh-discovery-quality.sh`

Aggregate output:
- `hard_signal_rows`: 8526
- `low_signal_rows`: 3273
- `category_other_low_signal`: 225
- `category_other_hard_agent_signal`: 74
- `quarantine_active`: true
- `planner_priority`: `quarantine_first`

Decision:
- Post-recrawl `category=other` state remains `no_op_fixed_point`.
- Taxonomy rule change: none from the low-signal aggregate cohort.
- Threshold adjustment: none.
- Reason: the low-signal `category=other` cohort still lacks hard agent signals. Aggregate counts alone do not prove a narrow taxonomy-rule change.

Guard state:
- `llms_only`, `schema_only`, `zero_score`, and `category_other_low_signal` remain audit-only.
- `public_search=false` for passive/soft-signal cohorts.
- `score_fix_targeting=false` for passive/soft-signal cohorts.
- Public search remains protected by `models.AgentFirstFilter`; score-fix targeting still requires a hard agent signal.

Generated-work queue decision:
- `harness/generated-work-items.json` was left unchanged because this discovery-quality lane is at a true fixed point. Existing queue entries are unrelated concrete follow-ups for API-key commerce, monitor quarantine review, and score-fix private cleanup.

