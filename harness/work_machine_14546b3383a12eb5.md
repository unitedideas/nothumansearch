# work_machine_14546b3383a12eb5

## Scope

Refresh and close out sanitized aggregate discovery-quality and discovery-quarantine state after the completed 2026-05-19 full recrawl. This run used only repo-local aggregate artifacts and helper output. It did not start a full recrawl, broad crawl, public submission, browser automation, or score-fix targeting.

## Aggregate Evidence

The 2026-05-19 recrawl was already completed before this WorkItem:

- `tools/recrawl-health.log` recorded `full_recrawl` start at `2026-05-19 06:00:11`, preflight `api_status=200 api_ok=1`, workers `10`, remote recrawl start, and completion at `2026-05-19 10:26:32` with post-recrawl `api_status=200 api_ok=1`.
- `tools/recrawl.log` recorded `Done. Success=9830 Failed=396 Total=10226` and `2026-05-19 10:26:32 NHS full_recrawl complete`.
- No repo-local full-recrawl or recrawl lock was present when the originating planner snapshot was created.

Existing aggregate closeout `harness/work_machine_0c0afd045287fa5c.md` records the sanitized May 19 refresh:

- Sample rows: 9827
- Hard-signal rows: 4114
- Low-signal rows: 5713
- Hard-signal rate: 0.4186
- `category=other` low-signal rows: 2466
- `category=other` hard-signal rows: 752
- `llms_only`: 1004
- `schema_only`: 538
- `zero_score`: 2187

Current tracked aggregate artifacts have since been refreshed by later bounded seed-refresh work. They remain sanitized and aggregate-only:

- `harness/discovery-quality-latest.json`
- `harness/discovery-quarantine-latest.json`
- `harness/discovery-quarantine-history.jsonl`

## Decision

Post-recrawl `category=other` state is a `no_op_fixed_point`.

- Taxonomy-rule change: no
- Threshold adjustment: no
- No-op fixed point: yes

Reason: the low-signal `category=other` cohort does not prove a narrow taxonomy rule by itself. Aggregate counts alone do not justify changing classification or score thresholds.

## Guard State

These cohorts remain audit-only unless a hard agent signal is proven:

- `llms_only`
- `schema_only`
- `zero_score`
- low-signal `category=other`

Required guard state remains:

- `public_search=false`
- `score_fix_targeting=false`
- Public search remains protected by `models.AgentFirstFilter`.
- Score-fix targeting continues to require a hard agent signal.

The hard-signal `category=other` cohort may be reviewed only through bounded private sampling or aggregate helper output. Raw domains, URLs, row IDs, descriptions, emails, tokens, and private notes must not be committed.

## Follow-Up Queue

No new `harness/generated-work-items.json` row was added for this lane. The aggregate discovery-quality lane is at an explicit fixed point; reopening it would require new aggregate evidence from a future bounded helper run, not a standing retry item.

## Verification

- `python3 -m json.tool harness/discovery-quality-latest.json`
- `python3 -m json.tool harness/discovery-quarantine-latest.json`
- `python3 -m json.tool harness/generated-work-items.json`
- `python3 tools/quality-gate-discovery.py --quarantine harness/discovery-quarantine-latest.json --repo-root .`
