# work_machine_0c0afd045287fa5c

## Scope

Refresh sanitized aggregate discovery-quality and discovery-quarantine artifacts after the completed 2026-05-19 full recrawl. No broad crawl, no full recrawl, no raw candidate rows.

## Aggregate Result

- Sample rows: 9827
- Hard-signal rows: 4114
- Low-signal rows: 5713
- Hard-signal rate: 0.4186
- `category=other` low-signal rows: 2466
- `category=other` hard-signal rows: 752
- `llms_only`: 1004
- `schema_only`: 538
- `zero_score`: 2187

## Decision

Post-recrawl `category=other` low-signal state is a `no_op_fixed_point`.

- Taxonomy-rule change: no
- Threshold adjustment: no
- No-op fixed point: yes

Low-signal `category=other`, `llms_only`, `schema_only`, and `zero_score` cohorts remain audit-only with `public_search=false` and `score_fix_targeting=false`.

The hard-signal `category=other` cohort remains eligible only for bounded private sampling. Any future taxonomy rule needs aggregate pattern evidence plus a crawler unit test; no raw row data should enter planner artifacts.

## Verification

- `python3 tools/discovery-quarantine-report.py --input harness/discovery-quality-latest.json --output harness/discovery-quarantine-latest.json --history-output harness/discovery-quarantine-history.jsonl --observed-at 2026-05-19T17:26:32Z`
- `python3 tools/quality-gate-discovery.py --quarantine harness/discovery-quarantine-latest.json --repo-root .`
- `python3 tools/test-discovery-quality-report.py && python3 tools/test-discovery-quarantine-report.py && python3 tools/test-taxonomy-other-redacted-sample.py`
- `GOCACHE=/private/tmp/nhs-go-build go test ./...`

## Blocker

Commit was blocked because git metadata writes failed with:

```text
fatal: Unable to create '/Users/owlassist/foundry-businesses/nothumansearch/.git/index.lock': Operation not permitted
```
