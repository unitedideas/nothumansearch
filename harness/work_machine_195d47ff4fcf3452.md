# Discovery Quarantine Fixed-Point Refresh

WorkItem: `work_machine_195d47ff4fcf3452`

Observed refresh: 2026-06-09T11:11:20Z

Refresh command:

```sh
NHS_DISCOVERY_INPUT=tools/seed-refresh.log ./tools/refresh-discovery-quality.sh
```

Aggregate-only output:

- `sample_rows`: 11328
- `hard_signal_rows`: 8186
- `low_signal_rows`: 3142
- `category_other_low_signal`: 216
- `category_other_hard_agent_signal`: 71
- `llms_only`: 556
- `schema_only`: 571
- `zero_score`: 1242
- `planner_priority`: `quarantine_first`

Decision:

- `post_recrawl_category_other_state`: `no_op_fixed_point`
- `taxonomy_rule_change`: false
- `threshold_adjustment`: false
- `no_op_fixed_point`: true
- `taxonomy_rule`: `not_from_low_signal_aggregate_cohort`
- `threshold_adjustment`: `none`

Guard state:

- `llms_only`: audit-only, `public_search=false`, `score_fix_targeting=false`
- `schema_only`: audit-only, `public_search=false`, `score_fix_targeting=false`
- `zero_score`: audit-only, `public_search=false`, `score_fix_targeting=false`
- `category_other_low_signal`: aggregate-review-only, `public_search=false`, `score_fix_targeting=false`

No full recrawl, broad crawl, raw domains, URLs, row IDs, descriptions, emails, tokens, or private notes were used in the committed artifact. The follow-up queue was left unchanged because this lane is still a fixed point unless later aggregate evidence proves a hard agent signal or a narrower taxonomy rule.
