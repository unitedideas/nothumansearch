# Commit blocker for work_machine_f7d95896f3c3629a

The WorkItem was completed locally, but this executor cannot write Git index locks:

```text
fatal: Unable to create '/Users/owlassist/foundry-businesses/nothumansearch/.git/index.lock': Operation not permitted
```

Files changed or added:

- `tools/discovery-quality-report.py`
- `tools/test-discovery-quality-report.py`
- `harness/discovery-quality-latest.json`
- `harness/discovery-quarantine-latest.json`
- `harness/discovery-quarantine-history.jsonl`
- `harness/generated-work-items.json`
- `ops/ledgers/work_machine_f7d95896f3c3629a.md`
- `harness/work_machine_f7d95896f3c3629a-commit-blocker-2026-06-12.md`

Aggregate-safe decision:

- The bounded category=other hard-signal review is an audit-only fixed point.
- Current aggregate proof: `category_other_hard_agent_signal=80`.
- Score buckets: `exact_0=0`, `1_24=80`, `0_24=80`, `25_39=0`, `40_59=0`, `60_plus=0`.
- Top signal sets: `API=54`, `API,schema.org=26`.
- This is not a scoring bug: API-only and API+schema rows are compatible with normal scoring weights.
- A taxonomy gap is not proven from aggregate evidence.
- Stale stored signal remains possible for individual rows, but is not proven by aggregate-only evidence.
- `public_search` and `score_fix_targeting` were not changed.

Verification passed:

```bash
./tools/refresh-discovery-quality.sh
python3 -m unittest tools/test-discovery-quality-report.py tools/test-discovery-quarantine-report.py tools/test-taxonomy-other-redacted-sample.py
python3 -m json.tool harness/generated-work-items.json >/dev/null
```

Commit commands for a runner with Git index write access:

```bash
git update-index --no-assume-unchanged tools/discovery-quality-report.py tools/test-discovery-quality-report.py harness/discovery-quality-latest.json harness/discovery-quarantine-latest.json harness/discovery-quarantine-history.jsonl harness/generated-work-items.json
git add tools/discovery-quality-report.py tools/test-discovery-quality-report.py harness/discovery-quality-latest.json harness/discovery-quarantine-latest.json harness/discovery-quarantine-history.jsonl harness/generated-work-items.json
git add -f ops/ledgers/work_machine_f7d95896f3c3629a.md harness/work_machine_f7d95896f3c3629a-commit-blocker-2026-06-12.md
git commit -m "Close category other exact-zero audit"
```
