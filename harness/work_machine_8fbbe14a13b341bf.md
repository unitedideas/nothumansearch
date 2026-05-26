# NHS discovery-quality aggregate closeout - work_machine_8fbbe14a13b341bf

Observed at: 2026-05-25T22:10:53Z

Scope: close the QLimit discovery-quality work item from aggregate artifacts only. No full recrawl, broad crawl, row-level sampler, deploy, browser automation, public action, production mutation, secret read, or raw crawler-row output was used.

## Boundary Evidence

Repo-local wrapper evidence records the requested completed boundary:

- `tools/recrawl-health.log` records the 2026-05-23 full-recrawl start at `06:00:08`, post-run health check at `10:44:26`, and completion with `api_status=200`, `api_ok=1`, `workers=10`, `dry_run=0`.
- `tools/recrawl.log` records the matching aggregate completion: `Done. Success: 9867, Failed: 379, Total: 10246` and `2026-05-23 10:44:26 NHS full_recrawl complete`.
- `tools/recrawl-health.log` records the 2026-05-24 seed-refresh completion at `05:39:06` with `api_status=200`, `api_ok=1`, `workers=10`, `dry_run=0`.
- `tools/seed-refresh.log` records the matching seed-refresh aggregate: `Done. Success: 477, Failed: 6, Total: 483`.

The 2026-05-23/2026-05-24 refresh proof already exists in `harness/work_machine_7b0379ce2d6ca72a.md` from `NHS_DISCOVERY_INPUT=tools/seed-refresh.log ./tools/refresh-discovery-quality.sh` before the later 2026-05-25 wrapper runs advanced the live aggregate files. That proof recorded:

- `sample_rows=5653`
- `hard_signal_rows=4096`
- `low_signal_rows=1557`
- `hard_signal_rate=0.7246`
- `category_other=143`
- `category_other_low_signal=108`
- `category_other_hard_agent_signal=35`
- `llms_only=279`
- `schema_only=286`
- `zero_score=620`
- `passive_only_share=0.2754`

Current aggregate artifacts are intentionally not rewound. `harness/discovery-quality-latest.json`, `harness/discovery-quarantine-latest.json`, and `harness/discovery-quarantine-history.jsonl` now reflect the later 2026-05-25 completed seed/full-recrawl boundary:

- `sample_rows=6127`
- `hard_signal_rows=4440`
- `low_signal_rows=1687`
- `hard_signal_rate=0.7247`
- `category_other=155`
- `category_other_low_signal=117`
- `category_other_hard_agent_signal=38`
- `llms_only=303`
- `schema_only=308`
- `zero_score=672`
- `passive_only_share=0.2753`

## Decision

- Taxonomy-rule change: `none`.
- Threshold adjustment: `none`.
- Fixed point: `no_op_fixed_point`.

Reason: `category=other` low-signal rows still lack proven hard agent signals in the aggregate-only review path. The aggregate counts do not identify a narrow taxonomy rule, and lowering thresholds would weaken the hard-signal boundary. The hard-signal `category=other` cohort remains aggregate-only review evidence, not planner row output.

## Guard State

These cohorts remain audit-only unless a hard agent signal is proven:

- `llms_only`: `public_search=false`, `score_fix_targeting=false`
- `schema_only`: `public_search=false`, `score_fix_targeting=false`
- `zero_score`: `public_search=false`, `score_fix_targeting=false`
- `category_other_low_signal`: `public_search=false`, `score_fix_targeting=false`

Public search remains protected by `models.AgentFirstFilter`. Score-fix targeting remains gated by `HasHardAgentSignal`.

## Generated Work Items

`harness/generated-work-items.json` is intentionally unchanged for this lane. The discovery-quality lane is at a true fixed point: no replacement full-recrawl, broad crawl, taxonomy-rule-change, threshold-adjustment, public-search, or score-fix-targeting follow-up is warranted from the aggregate evidence. Existing generated items are unrelated API-key commerce/admin traffic and score-fix private cleanup work.

## Verification

- `python3 -m json.tool harness/discovery-quality-latest.json`
- `python3 -m json.tool harness/discovery-quarantine-latest.json`
- `python3 -m json.tool harness/generated-work-items.json`
- `python3 -m unittest tools/test-discovery-quality-report.py tools/test-discovery-quarantine-report.py tools/quality-gate-discovery-test.py`

`go test ./...` was also attempted with `GOCACHE=/private/tmp/nothumansearch-go-cache`. It remains blocked by pre-existing `cmd/monitor-check` helper drift: `firstCheckFailedQuarantineReason`, `firstCheckZeroScoreQuarantineReason`, and `firstCheckQuarantineReason` are undefined in `cmd/monitor-check/main_test.go`.

## Commit Status

Commit blocked in this runner: `git add harness/work_machine_8fbbe14a13b341bf.md && git commit -m "Close discovery quality fixed point"` failed because Git could not create `.git/index.lock` (`Operation not permitted`). The worktree change is limited to this aggregate closeout note.
