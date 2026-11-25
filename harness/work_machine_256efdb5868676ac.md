# July 13 Discovery Closeout

WorkItem: `work_machine_256efdb5868676ac`

## Decision

The July 13 discovery wrapper did not exit early. The planner observation raced the final write to `tools/discover.log`.

Local evidence:

- Run header: `2026-07-13 05:00:01`
- Completion totals are present in the same run block.
- Final marker is present: `=== done: nothing new ===`
- No full recrawl was run for this closeout.

## Aggregate Closeout

Source counts from the completed July 13 wrapper block:

| source | candidate_domains |
|---|---:|
| mcp-registry | 27 |
| smithery | 144 |
| glama | 125 |
| awesome-mcp | 240 |
| apis.guru | 536 |
| llmstxt | 1897 |
| mcpservers.org | 188 |
| mcpmarket.com | 0 |
| mcp.so | 32 |
| pulsemcp | 150 |
| mcp.directory | 284 |
| mcpservers.com | 137 |
| github | 645 |

Final aggregate totals:

- total_unique_candidates: 4128
- new_domains_to_crawl: 0
- already_indexed: 4128
- completion: `nothing new`

Aggregate error classes visible in the July 13 error stream:

- GitHub source: HTTP 403 rate-limit class
- PulseMCP source: HTTP 410 gone class
- MCPMarket source: HTTP 429 rate-limit class

## Bounded Rerun

Command:

```bash
python3 tools/discover.py --closeout-only --aggregate-only
```

Result:

- rc: 0
- No crawler or submit path was entered.
- The runner had no working DNS and all source fetch failures collapsed to aggregate `gaierror` classes.
- Because the original July 13 wrapper block already contained the complete aggregate closeout, the DNS-limited rerun was not used as the source of truth for counts.

## Repair

`tools/discover.py` now has a closeout-only mode that computes source and index-status aggregates without submitting or crawling. Source fetch errors are logged as source/context/error-class only, including the 403, 410, and 429 classes, without raw candidate domains or raw candidate URLs.

Verification:

```bash
PYTHONPYCACHEPREFIX=/tmp/nhs-pycache python3 -m py_compile tools/discover.py
PYTHONPYCACHEPREFIX=/tmp/nhs-pycache python3 tools/quality-gate-discovery-test.py
```

Both passed locally.

## Privacy Boundary

This closeout artifact is aggregate-only. It does not include raw candidate domains, raw candidate URLs, tokens, process environments, or candidate rows.
