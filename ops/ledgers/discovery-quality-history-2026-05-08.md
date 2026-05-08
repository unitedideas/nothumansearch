# Discovery Quality History - 2026-05-08

Scope: completed the aggregate-only discovery quarantine trend-history and refresh-wrapper follow-ups.

## Changes

- `tools/discovery-quarantine-report.py` can append a weekly aggregate JSONL history entry with:
  - `hard_signal_rows`
  - `low_signal_rows`
  - `category_other_low_signal`
  - `quarantine.active`
  - `planner_priority`
- `tools/refresh-discovery-quality.sh` now refreshes:
  - `harness/discovery-quality-latest.json`
  - `harness/discovery-quarantine-latest.json`
  - `harness/discovery-quarantine-history.jsonl`
- The wrapper prints only aggregate counts and the history path.
- `harness/README.md` documents the no-recrawl, aggregate-only guardrail.
- `harness/generated-work-items.json` is empty because both local follow-ups are complete.

## Guardrails

- No full recrawl was run.
- No production rows were mutated.
- No deploy was run.
- No browser, Computer Use, public post, external submission, account action, or spend was used.
- No candidate domains, URLs, raw discovery log rows, or row identifiers were written to the history entry.
- Trend history is for business-local planner priority only; it does not change public ranking or score-fix targeting.

## Proof

- `tools/refresh-discovery-quality.sh`
- `python3 tools/test-discovery-quarantine-report.py`
- `python3 tools/test-discovery-quality-report.py`
- `python3 -m json.tool harness/discovery-quality-latest.json >/dev/null`
- `python3 -m json.tool harness/discovery-quarantine-latest.json >/dev/null`
- `python3 -m json.tool harness/generated-work-items.json >/dev/null`
- JSONL/domain guard:
  - parsed `harness/discovery-quarantine-history.jsonl`
  - rejected `candidate_domains`, `http://`, `https://`, and common domain patterns

Latest history entry:

```json
{"category_other_low_signal": 411, "hard_signal_rows": 308, "history_key": "discovery-quarantine:2026-05-04", "low_signal_rows": 571, "observed_at": "2026-05-08T17:52:34.650593Z", "planner_priority": "quarantine_first", "planner_scope": "business-local only; never public ranking or score-fix targeting", "quarantine": {"active": true}, "sample_rows": 879, "source": "harness/discovery-quality-latest.json", "week_start": "2026-05-04"}
```
