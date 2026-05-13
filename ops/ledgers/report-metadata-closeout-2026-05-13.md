# NHS report metadata closeout - 2026-05-13

WorkItem: `Repair stale NHS report metadata before public reuse`

Policy: aggregate-only. No public post, email, directory submission, row-level admin read, public-action lock, or external account action was used.

## Proof

- `GET https://nothumansearch.ai/api/v1/stats` returned `total_sites=4169`, `avg_score=35`, and `top_category=developer`.
- `GET https://nothumansearch.ai/report` rendered metadata for `4169` agent-first indexed domains, OpenGraph average score `35.1/100`, and `217` sites scoring `70+`.
- `GET https://nothumansearch.ai/llms.txt` rendered `We index 4169+ sites` and links to `https://nothumansearch.ai/report`.
- The stale public-copy claims `10205-site`, `23.2-average`, and `219-sites-score-70+` were not present in the live report proof.

## Decision

The stale-report WorkItem is closed. `/report` is safe for reuse as a current aggregate report surface, with the caveat that `/api/v1/stats` rounds average score to an integer while `/report` shows one decimal from the same agent-first corpus.

## Verification Commands

```sh
curl -fsS https://nothumansearch.ai/api/v1/stats
curl -fsS https://nothumansearch.ai/report | grep -E 'agent-first indexed domains|Average score|sites score 70\+'
curl -fsS https://nothumansearch.ai/llms.txt | grep -E '^We index|Report:'
python3 -m json.tool harness/generated-work-items.json
```
