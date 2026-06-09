# 2026-05-21 full recrawl aggregate closeout

WorkItem: `work_machine_1878125c43df2ad7`
Observed: `2026-06-09`

## Scope

This is an aggregate-only closeout for the completed 2026-05-21 full recrawl and the sanitized discovery-quality boundary artifacts already refreshed from that completed boundary. This note intentionally excludes raw domains, URLs, row IDs, descriptions, emails, tokens, private notes, crawler row output, browser automation, public posting, broad crawl, and full recrawl execution.

## Boundary Proof

- `tools/recrawl-health.log` records 2026-05-21 `seed_refresh` completion at `05:42:37` with `api_status=200`, `api_ok=1`, `workers=10`, and `dry_run=0`.
- `tools/recrawl-health.log` records 2026-05-21 `full_recrawl` start at `06:00:12` with preflight `api_status=200`, `api_ok=1`, `health_outcome=full_pressure`, and `workers=10`.
- `tools/recrawl-health.log` records remote start for `/app/crawler -recrawl -workers 10`.
- `tools/recrawl-health.log` records 2026-05-21 `full_recrawl` completion at `10:25:48` with post-recrawl `api_status=200`, `api_ok=1`, `workers=10`, and `dry_run=0`.
- `tools/recrawl.log` records the matching aggregate full-recrawl result: `success=9847`, `failed=389`, `total=10236`.
- Repo-local lock check found no `tools/full-recrawl.lock` or `tools/recrawl.lock`.

## Sanitized Discovery State

Current aggregate artifacts:

- `harness/discovery-quality-latest.json`
- `harness/discovery-quarantine-latest.json`
- `harness/discovery-quarantine-history.jsonl`

The current committed aggregate boundary for this lane is:

- `sample_rows=473`
- `hard_signal_rows=343`
- `low_signal_rows=130`
- `hard_signal_rate=0.7252`
- `category_other=12`
- `category_other_hard_agent_signal=3`
- `category_other_low_signal=9`
- `llms_only=23`
- `schema_only=23`
- `zero_score=52`

Hard-signal `category=other` review remains aggregate-only:

- `rows=3`
- `top_signal_sets`: `API=2`, `API,schema.org=1`
- `score_buckets`: `0_24=3`, `25_39=0`, `40_59=0`, `60_plus=0`

## Decision

Decision: `no_op_fixed_point`.

- Taxonomy-rule change: no.
- Threshold adjustment: no.
- No-op fixed point: yes.

Reason: the current low-signal `category=other` aggregate does not prove a reusable taxonomy rule. The low-signal cohorts lack proven API, OpenAPI, MCP, or ai-plugin evidence, and a threshold adjustment would weaken the hard-agent-signal boundary.

## Guard State

These cohorts remain audit-only unless a hard agent signal is proven through an executor-only path:

- `llms_only`: `public_search=false`, `score_fix_targeting=false`
- `schema_only`: `public_search=false`, `score_fix_targeting=false`
- `zero_score`: `public_search=false`, `score_fix_targeting=false`
- `category_other_low_signal`: `public_search=false`, `score_fix_targeting=false`

Public search remains protected by `models.AgentFirstFilter`. Score-fix targeting remains gated on `HasHardAgentSignal`.

## Queue Decision

No new `harness/generated-work-items.json` row is warranted for this discovery-quality lane. It is at a true fixed point: no replacement recrawl, broad crawl, taxonomy-rule change, threshold adjustment, public-search eligibility change, or score-fix targeting change follows from the aggregate evidence.

Existing generated rows are separate deploy-gated, credential-gated, or private cleanup lanes and were left unchanged.

## Verification

- `python3 -m json.tool harness/discovery-quality-latest.json`
- `python3 -m json.tool harness/discovery-quarantine-latest.json`
- `python3 -m json.tool harness/generated-work-items.json`
- `PYTHONDONTWRITEBYTECODE=1 python3 -m unittest tools/test-discovery-quality-report.py tools/test-discovery-quarantine-report.py tools/quality-gate-discovery-test.py`
