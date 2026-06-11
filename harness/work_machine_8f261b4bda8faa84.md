# Discovery Quarantine Aggregate Refresh

WorkItem: `work_machine_8f261b4bda8faa84`

## Scope

Refreshed the aggregate-only discovery-quality artifacts from the bounded seed-refresh aggregate source:

```bash
NHS_DISCOVERY_INPUT="$PWD/tools/seed-refresh.log" ./tools/refresh-discovery-quality.sh
```

No full recrawl, broad crawl, raw-domain sampler, browser automation, public posting, or credentialed admin workflow was used.

## Refreshed Artifacts

- `harness/discovery-quality-latest.json`
- `harness/discovery-quarantine-latest.json`
- `harness/discovery-quarantine-history.jsonl`

Aggregate proof:

- sample_rows: 12271
- hard_signal_rows: 8866
- low_signal_rows: 3405
- hard_signal_rate: 0.7225
- category_other: 311
- category_other_hard_agent_signal: 77
- category_other_low_signal: 234
- llms_only: 601
- schema_only: 621
- zero_score: 1342

## Decision

Post-recrawl `category=other` is a `no_op_fixed_point`.

Decision matrix:

- taxonomy_rule_change: false
- threshold_adjustment: false
- no_op_fixed_point: true

Reason: the remaining low-signal `category=other` aggregate lacks proven hard agent signals. Aggregate counts alone do not justify a taxonomy-rule change, and broadening thresholds would violate the agent-first search boundary.

## Guard State

The quarantine report keeps these cohorts audit-only:

- `llms_only`: `public_search=false`, `score_fix_targeting=false`
- `schema_only`: `public_search=false`, `score_fix_targeting=false`
- `zero_score`: `public_search=false`, `score_fix_targeting=false`
- `category_other_low_signal`: `public_search=false`, `score_fix_targeting=false`

Public guards remain:

- `public_search`: protected by `models.AgentFirstFilter`
- `score_fix_targeting`: requires `HasHardAgentSignal`

## Sanitization

Committed artifacts contain aggregate counts only. They do not include raw domains, URLs, row IDs, descriptions, emails, tokens, or private notes.

## Follow-Up

`tools/refresh-discovery-quality.sh` still defaults to `tools/discover.err`, which can be older than the latest bounded seed-refresh aggregate. The follow-up queue records a local hardening item to make the wrapper reject stale default inputs or select the newest bounded aggregate source.

## Verification

```bash
python3 -m unittest tools/test-discovery-quality-report.py tools/test-discovery-quarantine-report.py tools/quality-gate-discovery-test.py tools/test-taxonomy-other-redacted-sample.py
go test ./...
```
