# 2026-05-19 Discovery Quality Refresh

WorkItem: `work_machine_f0365180206c419a`

Input boundary used: completed 2026-05-19 full recrawl in `tools/recrawl.log`, bounded to `2026/05/19 13:00:00` through `2026/05/19 17:26:32`. This maps to the wrapper completion at `2026-05-19 10:26:32` local time.

Sanitized aggregate artifacts:
- `harness/discovery-quality-latest.json`
- `harness/discovery-quarantine-latest.json`
- `harness/discovery-quarantine-history.jsonl`

Aggregate result:
- sample_rows: 9827
- hard_signal_rows: 4114
- low_signal_rows: 5713
- category_other_low_signal: 2466
- category_other_hard_agent_signal: 752
- quality_gate: review, `low_signal_rows_exceed_hard_signal_rows`

Decision: no-op fixed point. This is not a taxonomy-rule change and not a threshold adjustment. Low-signal `category=other` rows lack hard agent signals, so they remain audit-only. `llms_only`, `schema_only`, `zero_score`, and low-signal `category=other` cohorts keep `public_search=false` and `score_fix_targeting=false` unless a hard agent signal is proven in a private executor review.

No full recrawl, broad crawl, public post, email, payment action, raw domain output, URL output, row ID output, description output, email output, token output, or private-note output was produced for this work item.
