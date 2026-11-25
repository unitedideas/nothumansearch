# Discovery-quality post-recrawl refresh

WorkItem: `work_machine_c1a78f75c63b73b8`
Observed: `2026-06-12`
Recrawl boundary: `2026-05-19`

## Sources

- `tools/recrawl-health.log`: 2026-05-19 full-recrawl started at 06:00:11 and completed at 10:26:32 with `api_status=200`, `api_ok=1`, `workers=10`, `dry_run=0`.
- `tools/recrawl.log`: 2026-05-19 bounded aggregate reports `success=9830`, `failed=396`, `total=10226`.
- Local lock check from WorkItem source: no `tools/full-recrawl.lock`.
- Aggregate helper input: date-bounded 2026-05-19 slice of `tools/recrawl.log`; no broad crawl, full recrawl, row-level output, domains, URLs, row ids, descriptions, emails, tokens, or private notes.

## Refreshed Artifacts

- `harness/discovery-quality-latest.json`
- `harness/discovery-quarantine-latest.json`
- `harness/discovery-quarantine-history.jsonl`

Aggregate counts from the bounded 2026-05-19 recrawl slice:

- sample rows: 9827
- hard-signal rows: 4114
- low-signal rows: 5713
- low-signal `category=other`: 2466
- hard-signal `category=other`: 752
- `llms-only`: 1004
- `schema-only`: 538
- `zero-score`: 2187

## Decision

Post-recrawl `category=other` state is a no-op fixed point.

- Taxonomy-rule change: no.
- Threshold adjustment: no.
- No-op fixed point: yes.

Reason: the low-signal `category=other`, `llms-only`, `schema-only`, and `zero-score` cohorts lack proven hard agent signals. Aggregate counts alone are not enough evidence for a narrow taxonomy rule or threshold change. The hard-signal `category=other` cohort remains reviewable only through sanitized targeted sampling.

## Guard State

- `llms-only`: audit-only, `public_search=false`, `score_fix_targeting=false`.
- `schema-only`: audit-only, `public_search=false`, `score_fix_targeting=false`.
- `zero-score`: audit-only, `public_search=false`, `score_fix_targeting=false`.
- Low-signal `category=other`: aggregate-review-only, `public_search=false`, `score_fix_targeting=false`.

`python3 tools/taxonomy-other-redacted-sample.py --limit 50` failed closed with `URLError` in this runtime and emitted only sanitized failure metadata.
