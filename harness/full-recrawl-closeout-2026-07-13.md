# NHS Full Recrawl Closeout - 2026-07-13

Generated: `2026-07-13T17:18:00Z`

Scope: aggregate-only closeout for `work_machine_b5e4d76b09c244d3`. This artifact intentionally excludes raw domains, crawl rows, candidate URLs, tokens, process environments, private queries, payment identifiers, user agents, and row-level data.

Wrapper evidence:

- `2026-07-13 06:00:06 wrapper=full-recrawl lock=full-recrawl.lock event=start phase=full_recrawl pid=63906`
- `2026-07-13 06:00:06 wrapper=full-recrawl lock=full-recrawl.lock event=health_check phase=preflight api_status=200 api_ok=1`
- `2026-07-13 06:00:06 wrapper=full-recrawl lock=full-recrawl.lock event=health_outcome phase=preflight action=full_pressure workers=10`
- `2026-07-13 06:00:06 wrapper=full-recrawl lock=full-recrawl.lock event=remote_start phase=recrawl command=/app/crawler_-recrawl_-workers_10`
- `2026-07-13 10:06:42 wrapper=full-recrawl lock=full-recrawl.lock event=health_check phase=post_full_recrawl api_status=200 api_ok=1`
- `2026-07-13 10:06:42 wrapper=full-recrawl lock=full-recrawl.lock event=completion phase=full_recrawl api_status=200 api_ok=1 workers=10 dry_run=0`

Lock status:

- `tools/full-recrawl.lock` absent at closeout check.
- No lock clear was performed.

Crawler aggregate:

- `success=10202`
- `failed=359`
- `total=10561`
- Completion marker: `2026-07-13 10:06:42 NHS full_recrawl complete`

Public aggregate probe:

- Runner DNS probe failed with `Could not resolve host: nothumansearch.ai`.
- Same-day WorkItem public aggregate snapshot is attached as fallback proof:
- `total_sites=4307`
- `avg_score=37`
- `top_category=developer`
- Categories: `developer=1312`, `ai-tools=923`, `other=806`, `data=393`, `finance=200`, `productivity=164`, `ecommerce=158`, `communication=121`, `security=113`, `health=62`, `jobs=26`, `education=20`, `news=8`, `spam=1`

Discovery aggregate:

- Weekly discovery completed at `2026-07-13 05:00:01`.
- `total_unique_candidates=4128`
- `new_domains_to_crawl=0`
- `already_indexed=4128`
- Final marker: `done: nothing new`

Decision:

- July 13 full-recrawl boundary is closed on wrapper completion, absent lock, aggregate crawler total, and same-day public aggregate fallback proof.
- No deploy, full recrawl, lock clear, raw crawl inspection, secret read, process-environment inspection, or public action was performed.
