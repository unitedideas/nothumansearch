# Discovery Quality Fixed Point Refresh

Work item: `work_machine_2712bda9bf7c42a2`

Date: 2026-06-09 local / 2026-06-10 UTC

Scope: aggregate-only refresh of sanitized discovery-quality and discovery-quarantine artifacts. No full recrawl, broad crawl, raw domains, URLs, row ids, descriptions, emails, tokens, private notes, or crawler row output were committed.

Inputs used:
- Completed wrapper boundary in `tools/recrawl-health.log`: seed refresh completed 2026-06-09 05:40:17 with `api_status=200 api_ok=1 workers=10 dry_run=0`; full recrawl completed 2026-06-09 10:49:35 with `api_status=200 api_ok=1 workers=10 dry_run=0`.
- Aggregate crawler summaries in `tools/seed-refresh.log` and `tools/recrawl.log`.
- Bounded helper path only: `NHS_DISCOVERY_INPUT=tools/seed-refresh.log ./tools/refresh-discovery-quality.sh`.
- Lock check: no `tools/full-recrawl.lock`, no `tools/recrawl.lock`, no `tools/seed-refresh.lock`.

Refreshed aggregate output:
- `sample_rows`: 11799
- `hard_signal_rows`: 8526
- `low_signal_rows`: 3273
- `hard_signal_rate`: 0.7226
- `llms_only`: 578
- `schema_only`: 596
- `zero_score`: 1292
- `category_other_low_signal`: 225
- `category_other_hard_agent_signal`: 74
- `hard_signal_other_review.top_signal_sets`: API=50, API+schema.org=24

Decision:
- `category=other` low-signal state remains `no_op_fixed_point`.
- Taxonomy rule change: none from the low-signal aggregate cohort.
- Threshold adjustment: none.
- Reason: the low-signal `category=other` cohort lacks hard agent signals. Aggregate counts alone do not prove a narrow taxonomy-rule change, and the hard-signal `category=other` review remains aggregate-only.

Guard state:
- `llms_only`, `schema_only`, `zero_score`, and `category_other_low_signal` remain audit-only.
- `public_search=false` for passive/soft-signal cohorts; public search remains protected by `models.AgentFirstFilter`.
- `score_fix_targeting=false` for passive/soft-signal cohorts; score-fix targeting still requires a hard agent signal.

Follow-up queue decision:
- `harness/generated-work-items.json` was left as-is. This discovery-quality lane is at a true fixed point, while the existing queue already holds unrelated concrete follow-ups for commerce handoff, monitor quarantine review, and score-fix private cleanup.

Verification:
- `python3 -m json.tool harness/discovery-quality-latest.json`
- `python3 -m json.tool harness/discovery-quarantine-latest.json`
- `python3 -m json.tool harness/generated-work-items.json`
- `python3 tools/test-discovery-quality-report.py`
- `python3 tools/test-discovery-quarantine-report.py`
- `python3 tools/quality-gate-discovery-test.py`
- `python3 tools/test-taxonomy-other-redacted-sample.py`
- `GOCACHE=/private/tmp/nhs-go-cache go test ./internal/... ./cmd/server ./cmd/crawler`
