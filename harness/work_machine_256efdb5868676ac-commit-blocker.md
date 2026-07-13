# Commit Blocker

WorkItem: `work_machine_256efdb5868676ac`

The local work is complete, but this runner cannot update the Git index.

Blocked command:

```bash
git update-index --no-assume-unchanged tools/discover.py harness/generated-work-items.json
```

Observed error:

```text
fatal: Unable to create '/Users/owlassist/foundry-businesses/nothumansearch/.git/index.lock': Operation not permitted
```

Why this matters:

- `tools/discover.py` and `harness/generated-work-items.json` are marked assume-unchanged.
- `git status` hides those modified tracked files until the index bit is cleared.
- `git add` and `git commit` require the same blocked index write path.

Files changed but not commit-able in this runner:

- `tools/discover.py`
- `harness/generated-work-items.json`
- `harness/work_machine_256efdb5868676ac.md`
- `harness/work_machine_256efdb5868676ac-commit-blocker.md`

Verification completed before the blocker:

```bash
PYTHONPYCACHEPREFIX=/tmp/nhs-pycache python3 -m py_compile tools/discover.py
PYTHONPYCACHEPREFIX=/tmp/nhs-pycache python3 tools/quality-gate-discovery-test.py
python3 -m json.tool harness/generated-work-items.json
```

Next action on a writable runner:

```bash
git update-index --no-assume-unchanged tools/discover.py harness/generated-work-items.json
git add tools/discover.py harness/generated-work-items.json harness/work_machine_256efdb5868676ac.md harness/work_machine_256efdb5868676ac-commit-blocker.md
git commit -m "Close out July discovery wrapper"
```
