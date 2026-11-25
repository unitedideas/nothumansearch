# work_machine_c0a3fb427acf6904 closeout

Observed at: 2026-06-11T08:10:57Z

## Action

Ran the aggregate-only discovery-quality refresh through `tools/refresh-discovery-quality.sh`, which uses `tools/discovery-quality-report.py` and `tools/discovery-quarantine-report.py`.

Input source remained bounded aggregate crawler summary logs only. No full recrawl, broad crawl, raw domain sample, raw URL sample, row ID sample, descriptions, emails, tokens, or private notes were used in committed artifacts.

## Aggregate Result

- `sample_rows`: 12271
- `hard_signal_rows`: 8866
- `low_signal_rows`: 3405
- `hard_signal_rate`: 0.7225
- `category_other`: 311
- `category_other_low_signal`: 234
- `category_other_hard_agent_signal`: 77
- `llms_only`: 601
- `schema_only`: 621
- `zero_score`: 1342

## Decision

Post-recrawl `category=other` state is a no-op fixed point.

- Taxonomy-rule change: false
- Threshold adjustment: false
- No-op fixed point: true

Reason: the low-signal `category=other` aggregate cohort does not prove a narrow taxonomy rule. Rows without API, OpenAPI, MCP, or ai-plugin remain audit-only.

## Guard State

The refreshed artifacts keep all passive and low-signal cleanup cohorts out of public ranking and score-fix targeting:

- `llms_only`: `public_search=false`, `score_fix_targeting=false`
- `schema_only`: `public_search=false`, `score_fix_targeting=false`
- `zero_score`: `public_search=false`, `score_fix_targeting=false`
- `category_other_low_signal`: `public_search=false`, `score_fix_targeting=false`

## Follow-Up

`harness/generated-work-items.json` now keeps the remaining bounded private review focused on the `category=other` hard-agent-signal low-score cohort. That work must use targeted sampler or aggregate helper output only and must not start a full recrawl or broad crawl.

## Verification

- `./tools/refresh-discovery-quality.sh`
- `python3 tools/test-discovery-quality-report.py`
- `python3 tools/test-discovery-quarantine-report.py`
- `python3 tools/test-taxonomy-other-redacted-sample.py`
- `GOCACHE="$PWD/.cache/go-build" go test ./...`
- Sanitized artifact scan for URL-like text, email-like text, and high-signal secret patterns across refreshed committed harness artifacts
