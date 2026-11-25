# work_machine_1001b62cb8bcdb39

Scope: close the QLimit discovery-quality WorkItem from sanitized aggregate artifacts only.

No full recrawl, broad crawl, deploy, browser automation, desktop automation, public action, production data deletion, process-environment inspection, row-level admin fetch, or secret read was performed.

## Inputs

- Prior completed full-recrawl boundary: 2026-05-19.
- Existing committed aggregate refresh proof: `harness/work_machine_0c0afd045287fa5c.md`.
- Aggregate artifacts checked in this run:
  - `harness/discovery-quality-latest.json`
  - `harness/discovery-quarantine-latest.json`
  - `harness/discovery-quarantine-history.jsonl`

The aggregate refresh state remains sanitized. The committed artifacts contain counts, score buckets, signal-set buckets, and policy flags only. They do not include raw domains, URLs, row IDs, descriptions, emails, tokens, private notes, or candidate-row review text.

## Aggregate State

- Sample rows: 9827
- Hard-signal rows: 4114
- Low-signal rows: 5713
- `category=other` low-signal rows: 2466
- `category=other` hard-signal rows: 752
- `llms_only`: 1004
- `schema_only`: 538
- `zero_score`: 2187

## Decision

Post-recrawl `category=other` state is a no-op fixed point for this lane.

- Taxonomy-rule change: no
- Threshold adjustment: no
- No-op fixed point: yes

Reason: low-signal `category=other` rows lack proven hard agent signals in this aggregate-only path. Aggregate counts alone do not identify a narrow reusable taxonomy rule, and lowering thresholds would weaken the hard-signal boundary.

The `llms_only`, `schema_only`, `zero_score`, and low-signal `category=other` cohorts remain audit-only with `public_search=false` and `score_fix_targeting=false` unless a future bounded private review proves a hard agent signal and summarizes it without raw row data.

The hard-signal `category=other` cohort remains eligible only for bounded private sampling. Any future taxonomy-rule change needs aggregate pattern evidence plus a crawler unit test before it can affect public search or score-fix targeting.

## Follow-up Queue

`harness/generated-work-items.json` was left unchanged. This discovery-quality lane is at a true fixed point. The remaining generated items are separate API-key commerce, monitor quarantine, and score-fix credential-gated lanes.

## Verification

- `python3 tools/quality-gate-discovery.py --quarantine harness/discovery-quarantine-latest.json --repo-root .`
- `python3 -m json.tool harness/discovery-quarantine-latest.json`
- `python3 -m json.tool harness/discovery-quality-latest.json`
- `python3 -m json.tool harness/generated-work-items.json`
- `PYTHONDONTWRITEBYTECODE=1 python3 -m unittest tools/test-discovery-quality-report.py tools/test-discovery-quarantine-report.py tools/test-taxonomy-other-redacted-sample.py tools/quality-gate-discovery-test.py`
- `GOCACHE=/private/tmp/nhs-go-build go test ./...`
