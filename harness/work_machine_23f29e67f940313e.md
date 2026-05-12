# Discovery Quarantine Execution Gate

WorkItem: `work_machine_23f29e67f940313e`

Result: locally complete; commit blocked by sandbox denial on `.git/index.lock`.

The existing executable gate already satisfies the WorkItem:

- `tools/discovery-quality-report.py` emits aggregate planner counts only.
- `tools/discovery-quarantine-report.py` rejects candidate-domain fields and
  emits no domains, URLs, raw rows, or row IDs.
- `tools/quality-gate-discovery.py` validates the quarantine artifact,
  `models.AgentFirstFilter`, public discovery SQL, and score-fix eligibility.
- `tools/refresh-discovery-quality.sh` runs the bounded aggregate refresh and
  does not run a full recrawl.

Verification completed:

- `python3 -m unittest tools/test-discovery-quarantine-report.py tools/quality-gate-discovery-test.py`
- `./tools/refresh-discovery-quality.sh`
- `GOCACHE="$PWD/.gocache" go test ./...`

Refresh proof:

- `hard_signal_rows=308`
- `low_signal_rows=571`
- `category_other_low_signal=411`
- `quarantine_active=true`
- `planner_priority=quarantine_first`

State updates:

- Removed the completed quarantine-gate item from
  `harness/generated-work-items.json`.
- Added matching private ledger detail at
  `ops/ledgers/work_machine_23f29e67f940313e.md`.

Commit blocker:

`git update-index --no-assume-unchanged harness/generated-work-items.json`
failed with:

`fatal: Unable to create '/Users/owlassist/foundry-businesses/nothumansearch/.git/index.lock': Operation not permitted`
