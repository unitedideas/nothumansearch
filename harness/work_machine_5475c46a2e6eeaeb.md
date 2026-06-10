# Discovery Quarantine Aggregate Refresh - work_machine_5475c46a2e6eeaeb

Observed at: 2026-06-10T18:11:24Z

Input path: `tools/discover.err`
Sanitized aggregate path: `harness/discovery-quality-latest.json`
Quarantine output path: `harness/discovery-quarantine-latest.json`
History path: `harness/discovery-quarantine-history.jsonl`

Command run:

```sh
./tools/refresh-discovery-quality.sh
```

Sanitized aggregate result:

- sample_rows: 879
- hard_signal_rows: 308
- low_signal_rows: 571
- hard_signal_rate: 0.3504
- category_other_low_signal: 411
- category_other_hard_agent_signal: 132
- llms_only: 84
- schema_only: 46
- zero_score: 122
- passive_only_share: 0.6496
- planner_priority: quarantine_first

Decision:

- post_recrawl_category_other_state: no_op_fixed_point
- taxonomy_rule_change: false
- threshold_adjustment: false
- no_op_fixed_point: true
- taxonomy_rule: not_from_low_signal_aggregate_cohort
- threshold_adjustment: none

Reason:

The refreshed bounded discovery sampler still shows low-signal rows exceeding hard-signal rows and category=other low-signal rows exceeding category=other hard-signal rows. That is not evidence for a new taxonomy rule, because the low-signal cohort lacks API, OpenAPI, MCP, or ai-plugin proof. A threshold adjustment would weaken the hard-agent-signal boundary. The correct state remains a no-op fixed point with bounded aggregate review only.

Guard state:

- llms_only: audit_only, public_search=false, score_fix_targeting=false
- schema_only: audit_only, public_search=false, score_fix_targeting=false
- zero_score: audit_only, public_search=false, score_fix_targeting=false
- category_other_low_signal: aggregate_review_only, public_search=false, score_fix_targeting=false

Execution boundaries:

- Used only repo-local aggregate helpers.
- Did not start a full recrawl or broad crawl.
- Did not copy raw domains, URLs, row ids, names, descriptions, emails, tokens, private notes, crawler row output, or private query logs into committed artifacts.
- Left `harness/generated-work-items.json` unchanged because this discovery-quality lane is at a true fixed point. The remaining generated work items are unrelated deploy-required or credential-required work.
