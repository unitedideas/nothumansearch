# Discovery Quarantine Aggregate Refresh - work_machine_51c7a9a70a4f6403

Observed at: 2026-06-10T16:11:39Z

Input path: `harness/discovery-quality-latest.json`
Output path: `harness/discovery-quarantine-latest.json`
History path: `harness/discovery-quarantine-history.jsonl`

Command run:

```sh
python3 tools/discovery-quarantine-report.py --input harness/discovery-quality-latest.json --output harness/discovery-quarantine-latest.json --history-output harness/discovery-quarantine-history.jsonl --observed-at 2026-06-10T16:11:39Z
```

Sanitized aggregate result:

- sample_rows: 12271
- hard_signal_rows: 8866
- low_signal_rows: 3405
- hard_signal_rate: 0.7225
- category_other_low_signal: 234
- category_other_hard_agent_signal: 77
- llms_only: 601
- schema_only: 621
- zero_score: 1342
- passive_only_share: 0.2775
- planner_priority: quarantine_first

Decision:

- post_recrawl_category_other_state: no_op_fixed_point
- taxonomy_rule_change: false
- threshold_adjustment: false
- no_op_fixed_point: true
- taxonomy_rule: not_from_low_signal_aggregate_cohort
- threshold_adjustment: none

Reason:

The category=other low-signal cohort lacks hard agent signals in the sanitized aggregate artifact. Aggregate counts alone do not prove a reusable taxonomy rule, and a threshold adjustment would weaken the hard-signal boundary. The correct state is a no-op fixed point with bounded aggregate review only.

Guard state:

- llms_only: audit_only, public_search=false, score_fix_targeting=false
- schema_only: audit_only, public_search=false, score_fix_targeting=false
- zero_score: audit_only, public_search=false, score_fix_targeting=false
- category_other_low_signal: aggregate_review_only, public_search=false, score_fix_targeting=false

Execution boundaries:

- No raw domains, URLs, row ids, names, descriptions, emails, tokens, private notes, crawler row output, full recrawl, broad crawl, browser automation, public posting, or paid action were used.
- `harness/generated-work-items.json` was inspected and left unchanged because this discovery-quality lane is at a true fixed point. The existing rows are unrelated deploy-required or credential-required work.
