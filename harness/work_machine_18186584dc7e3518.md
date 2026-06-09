# Discovery-quality aggregate fixed point

WorkItem: `work_machine_18186584dc7e3518`
Observed: `2026-06-09`

## Sources

- `tools/recrawl-health.log`: `2026-05-21 10:25:48` full-recrawl completion, `api_status=200`, `api_ok=1`, `workers=10`, `dry_run=0`.
- `tools/recrawl.log`: full-recrawl aggregate `success=9847`, `failed=389`, `total=10236`.
- `tools/recrawl-health.log`: `2026-05-22 05:39:47` seed-refresh completion, `api_status=200`, `api_ok=1`, `workers=10`, `dry_run=0`.
- `tools/seed-refresh.log`: seed-refresh aggregate `success=476`, `failed=7`, `total=483`.
- Lock check: `tools/full-recrawl.lock` absent and `tools/recrawl.lock` absent.

## Decision

`category=other` stays a no-op fixed point for this aggregate review.

- Taxonomy-rule change: no.
- Threshold adjustment: no.
- No-op fixed point: yes.

The aggregate evidence shows passive and soft-signal cohorts, but no committed hard-signal proof that would justify promoting `llms-only`, `schema-only`, `zero-score`, or low-signal `category=other` rows.

## Guard State

- `llms-only`: audit-only, `public_search=false`, `score_fix_targeting=false`.
- `schema-only`: audit-only, `public_search=false`, `score_fix_targeting=false`.
- `zero-score`: audit-only, `public_search=false`, `score_fix_targeting=false`.
- Low-signal `category=other`: audit-only, `public_search=false`, `score_fix_targeting=false`.

No broad crawl, full recrawl, row-level sampler output, domains, URLs, row ids, descriptions, emails, tokens, or private notes were committed.

`harness/generated-work-items.json` was left unchanged because this lane is at a true fixed point and existing follow-ups are for separate API-key commerce, monitor-admin, and score-fix credential-gated work.
