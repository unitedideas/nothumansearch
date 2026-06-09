# Work machine 11b79c5ae852530f

Date: 2026-06-08

Scope: aggregate-only NHS discovery-quality refresh.

Input used: `tools/seed-refresh.log` through `tools/refresh-discovery-quality.sh`.
No full recrawl or broad crawl was started.

Refreshed artifacts:

- `harness/discovery-quality-latest.json`
- `harness/discovery-quarantine-latest.json`
- `harness/discovery-quarantine-history.jsonl`

Aggregate refresh result:

- sample_rows: 11328
- hard_signal_rows: 8186
- low_signal_rows: 3142
- hard_signal_rate: 0.7226
- category_other_low_signal: 216
- category_other_hard_agent_signal: 71
- llms_only: 556
- schema_only: 571
- zero_score: 1242

Decision:

- post-recrawl category=other state: no-op fixed point
- taxonomy-rule change: false
- threshold adjustment: false
- no-op fixed point: true

Reason: the low-signal category=other cohort remains aggregate-review-only.
Aggregate counts alone do not prove a narrow taxonomy rule, and the passive
cohorts do not prove a hard agent signal.

Guard state:

- llms-only: audit-only, public_search=false, score_fix_targeting=false
- schema-only: audit-only, public_search=false, score_fix_targeting=false
- zero-score: audit-only, public_search=false, score_fix_targeting=false
- category=other low-signal: aggregate-review-only, public_search=false, score_fix_targeting=false

Follow-up queue decision:

`harness/generated-work-items.json` was left unchanged because this lane is at
a true fixed point. The existing generated queue already carries higher-value
non-discovery-quality ops follow-ups, and this refresh did not create a new
taxonomy, threshold, crawler, public-search, or score-fix work item.

Verification:

- `NHS_DISCOVERY_INPUT=tools/seed-refresh.log ./tools/refresh-discovery-quality.sh`
- `python3 -m json.tool harness/discovery-quality-latest.json`
- `python3 -m json.tool harness/discovery-quarantine-latest.json`
- `python3 -m json.tool harness/generated-work-items.json`
- `python3 tools/test-discovery-quality-report.py`
- `python3 tools/test-discovery-quarantine-report.py`
- `python3 tools/quality-gate-discovery-test.py`
- `python3 tools/test-taxonomy-other-redacted-sample.py`
- `GOCACHE=/private/tmp/nhs-go-cache go test ./internal/... ./cmd/server ./cmd/crawler`
