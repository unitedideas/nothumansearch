Commit blocker for `work_machine_2247d15593bb4097`.

The discovery-quality fixed-point closeout is written at:

- `harness/work_machine_2247d15593bb4097.md`

The bounded aggregate helper was run and verification passed:

```bash
NHS_DISCOVERY_INPUT=tools/seed-refresh.log ./tools/refresh-discovery-quality.sh
python3 -m json.tool harness/discovery-quality-latest.json >/dev/null
python3 -m json.tool harness/discovery-quarantine-latest.json >/dev/null
python3 -m json.tool harness/generated-work-items.json >/dev/null
python3 tools/quality-gate-discovery.py --quarantine harness/discovery-quarantine-latest.json --repo-root .
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest tools/test-discovery-quality-report.py tools/test-discovery-quarantine-report.py tools/quality-gate-discovery-test.py
```

Commit failed in this executor:

```text
fatal: Unable to create '/Users/owlassist/foundry-businesses/nothumansearch/.git/index.lock': Operation not permitted
```

Run from a git-writable executor:

```bash
git add harness/work_machine_2247d15593bb4097.md harness/work_machine_2247d15593bb4097-commit-blocker-2026-06-09.md
git commit -m "Record discovery quality fixed point"
```
