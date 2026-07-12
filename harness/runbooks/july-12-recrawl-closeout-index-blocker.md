# July 12 recrawl closeout index blocker

WorkItem: `work_machine_267aa8c5d4641d0d`

The July 12 full-recrawl closeout was completed locally with aggregate-only proof, but this runner could not stage or commit because git index writes are blocked:

```text
fatal: Unable to create '/Users/owlassist/foundry-businesses/nothumansearch/.git/index.lock': Operation not permitted
```

Aggregate-only local changes made:

- `ops/ledgers/README.md`: appended `Full recrawl closeout - 2026-07-12`.
- `harness/generated-work-items.json`: replaced the completed recrawl closeout item with metadata, discovery-quality, and commerce follow-ups.
- `harness/discovery-quality-latest.json`, `harness/discovery-quarantine-latest.json`, and `harness/discovery-quarantine-history.jsonl`: refreshed by `tools/refresh-discovery-quality.sh`.

Closeout proof:

- Wrapper completion: `2026-07-12 09:46:33 wrapper=full-recrawl lock=full-recrawl.lock event=completion phase=full_recrawl api_status=200 api_ok=1 workers=10 dry_run=0`.
- Lock state: `tools/full-recrawl.lock` absent at closeout.
- Crawler aggregate: `success=10220`, `failed=338`, `total=10558`.
- Public wrapper health: post-recrawl `api_status=200 api_ok=1`.
- Runner public DNS probes failed with `Could not resolve host: nothumansearch.ai`; no public body was committed from this runner.
- Last planner public aggregate snapshot: `total_sites=4307`, `avg_score=37`; categories `developer=1314`, `ai-tools=922`, `other=803`, `spam=1`.
- Bounded helper aggregate: `hard_signal_rows=18449`, `low_signal_rows=7061`, `category_other_low_signal=472`, `quarantine_active=true`, `planner_priority=quarantine_first`.

Verification already run:

```sh
python3 -m json.tool harness/generated-work-items.json >/dev/null
python3 -m json.tool harness/discovery-quality-latest.json >/dev/null
python3 -m json.tool harness/discovery-quarantine-latest.json >/dev/null
python3 tools/test-refresh-discovery-quality.py
python3 tools/test-discovery-quality-report.py
python3 tools/test-discovery-quarantine-report.py
GOCACHE=/private/tmp/nhs-go-cache go test ./...
```

Recovery on a writable runner:

```sh
git update-index --no-assume-unchanged ops/ledgers/README.md harness/generated-work-items.json harness/discovery-quality-latest.json harness/discovery-quarantine-latest.json harness/discovery-quarantine-history.jsonl
git add ops/ledgers/README.md harness/generated-work-items.json harness/discovery-quality-latest.json harness/discovery-quarantine-latest.json harness/discovery-quarantine-history.jsonl harness/runbooks/july-12-recrawl-closeout-index-blocker.md
git commit -m "Close out July 12 recrawl aggregate proof"
```
