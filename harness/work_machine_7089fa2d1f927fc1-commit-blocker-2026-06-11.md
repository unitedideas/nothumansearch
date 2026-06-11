Scope: QLimit WorkItem `work_machine_7089fa2d1f927fc1`.

Implemented locally:

- `tools/refresh-discovery-quality.sh` now chooses the newest non-empty bounded aggregate input from `tools/seed-refresh.log` and `tools/discover.err` by default instead of hard-defaulting to stale `tools/discover.err`.
- Explicit `NHS_DISCOVERY_INPUT=tools/discover.err` is rejected when `tools/seed-refresh.log` is newer, unless `NHS_DISCOVERY_ALLOW_STALE_DISCOVER_ERR=1` is set.
- `NHS_DISCOVERY_LOG_DIR` was added for deterministic fixture tests without touching production logs.
- `tools/test-refresh-discovery-quality.py` verifies both default supersession and explicit stale-input rejection while asserting aggregate-only output does not contain fixture domains.
- `harness/discovery-quality-latest.json` and `harness/discovery-quarantine-latest.json` were regenerated through `./tools/refresh-discovery-quality.sh`.

Aggregate refresh proof:

```text
discovery_quality_refresh hard_signal_rows=8866 low_signal_rows=3405 category_other_low_signal=234 quarantine_active=true planner_priority=quarantine_first history=harness/discovery-quarantine-history.jsonl
```

Latest sanitized aggregate counts:

```text
sample_rows=12271
hard_signal_rows=8866
low_signal_rows=3405
hard_signal_rate=0.7225
category_other_hard_agent_signal=77
category_other_low_signal=234
```

Verification passed:

```text
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest tools/test-refresh-discovery-quality.py tools/test-discovery-quality-report.py tools/test-discovery-quarantine-report.py tools/quality-gate-discovery-test.py
Ran 20 tests in 0.375s
OK

python3 -m json.tool harness/discovery-quality-latest.json
python3 -m json.tool harness/discovery-quarantine-latest.json
python3 tools/quality-gate-discovery.py --quarantine harness/discovery-quarantine-latest.json --repo-root .
discovery_quality_gate quarantine_active=true hard_signal_rows=8866 low_signal_rows=3405 category_other_low_signal=234 planner_priority=quarantine_first public_filters=agent_first

GOCACHE="$PWD/.gocache" go test ./...
ok github.com/unitedideas/nothumansearch/cmd/monitor-check
ok github.com/unitedideas/nothumansearch/internal/crawler
ok github.com/unitedideas/nothumansearch/internal/database
ok github.com/unitedideas/nothumansearch/internal/handlers
ok github.com/unitedideas/nothumansearch/internal/models
```

Commit blocker:

```text
git add -f tools/test-refresh-discovery-quality.py && git add tools/refresh-discovery-quality.sh harness/discovery-quality-latest.json harness/discovery-quarantine-latest.json harness/discovery-quarantine-history.jsonl harness/generated-work-items.json && git commit -m "Guard discovery quality refresh input"
fatal: Unable to create '/Users/owlassist/foundry-businesses/nothumansearch/.git/index.lock': Operation not permitted
```

`harness/generated-work-items.json` was inspected and left unchanged. This discovery-quality default-input lane is at a true fixed point after the guard and tests; the existing generated work items are unrelated deploy-required or credential-required lanes.
