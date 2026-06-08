# Commit blocker

WorkItem: `work_machine_fdd98653107272e5`

The score-fix cleanup closeout was written in the worktree, but this executor could not stage or commit because Git cannot create the index lock:

```sh
git add harness/work_machine_fdd98653107272e5.md ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md harness/generated-work-items.json
```

Result:

```text
fatal: Unable to create '/Users/owlassist/foundry-businesses/nothumansearch/.git/index.lock': Operation not permitted
```

Files to commit from a git-writable worker:

- `harness/work_machine_fdd98653107272e5.md`
- `harness/work_machine_fdd98653107272e5-commit-blocker-2026-06-08.md`
- `ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md`
- `harness/generated-work-items.json`

Suggested commands:

```sh
git update-index --no-assume-unchanged harness/generated-work-items.json ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md
git add harness/work_machine_fdd98653107272e5.md harness/work_machine_fdd98653107272e5-commit-blocker-2026-06-08.md ops/ledgers/score-fix-internal-test-cleanup-2026-05-12.md harness/generated-work-items.json
git commit -m "Record score-fix cleanup blocker"
```

Verification completed before the commit blocker:

- `python3 -m json.tool harness/generated-work-items.json >/dev/null`
- `python3 tools/test-redact-geo-jobs.py`
- `GOCACHE=/private/tmp/nothumansearch-gocache go test ./...`
