# Fly Auth Wrapper Unification — 2026-05-09

Automation: `business-agent-not-human-search`
Dedupe key: `nhs-fly-auth-wrapper-unification-2026-05-09`

## Change

- Added `tools/fly-auth.sh` as the shared launchd-safe Fly SSH helper.
- Updated `tools/recrawl-common.sh` and `tools/monitor-check.sh` to use the shared helper.
- Updated `tools/discover.py` to use the same Fly binary, config dir, and Keychain service without shell command substitution.

## Redacted Credential Evidence

- `fly-api-token=SET`
- Token value was not printed or copied.

## Proof

```text
NHS_MONITOR_CHECK_DRY_RUN=1 NHS_MONITOR_CHECK_REMOTE_COMMAND=true tools/monitor-check.sh
python3 -m py_compile tools/discover.py
bash -n tools/fly-auth.sh tools/monitor-check.sh tools/recrawl-common.sh tools/full-recrawl.sh tools/seed-refresh.sh
NHS_RECRAWL_DRY_RUN=1 NHS_RECRAWL_HEALTH_FIXTURE=tools/fixtures/recrawl-health-ok.json tools/full-recrawl.sh
```

Dry-run logs recorded shared wrapper construction without running a monitor crawl, seed crawl, discovery crawl, or full recrawl.
