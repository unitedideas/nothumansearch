# NHS recrawl and score-fix boundary

Date: 2026-05-13T16:47:00Z
Automation: business-agent-not-human-search

## Scope

Confirmed the 2026-05-13 full-recrawl boundary and refreshed score-fix aggregate status without starting another crawl, deploying code, reading process environments, printing secrets, or taking a customer-visible action.

## Recrawl boundary proof

Commands:

```sh
tail -160 tools/recrawl.log
tail -30 tools/recrawl-health.log
find . -name '*recrawl*.lock' -o -name 'full-recrawl.lock' -o -name '*crawler*.lock'
curl -fsS --max-time 20 https://nothumansearch.ai/api/v1/stats
curl -fsS --max-time 20 https://nothumansearch.ai/api/v1/categories
```

Observed:

- `tools/recrawl-health.log` still shows `2026-05-13 06:00:11` full-recrawl `remote_start` and no later `event=completion`.
- No repo-local recrawl/crawler lock file was present.
- `tools/recrawl.log` was still advancing during this run. Sample moved to `mtime=2026-05-13T09:46:50-0700`, size `35209985`, with crawler rows through `2026/05/13 16:46:50`.
- Public aggregate API was healthy: `/api/v1/stats` returned `total_sites=4170`, `avg_score=35`, `top_category=developer`.
- `/api/v1/categories` returned `other=769`, `developer=1228`, `ai-tools=895`, `spam=1`.

Decision: active-run boundary. Do not deploy, run monitor-check, trigger recategorize, or start a manual crawl until the recrawl log stops advancing or the wrapper records completion.

## Score-fix aggregate proof

Commands:

```sh
for s in nhs-admin-api-key nothumansearch-admin-key; do ... SET/MISSING only ...; done
tools/geo-jobs-redacted-read.sh
```

Observed:

- `nhs-admin-api-key`: `SET`
- `nothumansearch-admin-key`: `SET`
- Total score-fix rows: 11.
- Real-candidate pending rows: 2, both `dot_com`, age bucket `7_29d`.
- Test-like rows: 9, including 4 pending, 2 paid, 1 lead, and 2 internal-test rows.
- Real paid or lead rows: 0.

Decision: no customer-visible score-fix follow-up is due in this run. The two real pending rows are the external cohort already followed up in `ops/ledgers/score-fix-pending-followup-2026-05-12.md`; another touch requires a new duplicate-ledger review and fresh public-action lock.

## Verification

```sh
python3 -m json.tool harness/generated-work-items.json >/dev/null
```
