# Commit blocker - work_machine_3dea3c961aa10ebe

The discovery-quality refresh completed locally, but this runtime cannot write Git metadata:

`fatal: Unable to create '/Users/owlassist/foundry-businesses/nothumansearch/.git/index.lock': Operation not permitted`

Files to commit from a git-writable worker:

- `harness/discovery-quality-latest.json`
- `harness/discovery-quarantine-latest.json`
- `harness/generated-work-items.json`
- `ops/ledgers/work_machine_3dea3c961aa10ebe.md` with `git add -f` because `ops/ledgers/` is ignored
- `harness/work_machine_3dea3c961aa10ebe-commit-blocker-2026-06-10.md`

Suggested commit message:

`Refresh discovery quarantine aggregate`

Verification already run:

- `python3 -m json.tool harness/discovery-quality-latest.json`
- `python3 -m json.tool harness/discovery-quarantine-latest.json`
- `python3 -m json.tool harness/generated-work-items.json`
- `python3 tools/test-discovery-quality-report.py`
- `python3 tools/test-discovery-quarantine-report.py`
- `GOCACHE=/tmp/nhs-go-cache go test ./...`
