# Not Human Search

The Google for AI agents. Search engine that indexes agent-first tools ranked by agentic readiness.

## Stack
- Go server + crawler, single binary
- Postgres on Fly.io (sjc region, 2 shared-cpu machines, 512MB each)
- Resend verified for nothumansearch.ai (sending + receiving)

## Key Architecture
- `cmd/server/main.go` — HTTP server with middleware chain: logging > domain redirect > CORS
- `cmd/crawler/main.go` — CLI: `-seed` (add new), `-recrawl` (update all), `-url` (single)
- `internal/crawler/crawler.go` — CrawlSite() checks 7 signals, categorize(), generateTags()
- `internal/crawler/seeds.go` — 440+ seed URLs
- `internal/models/queries.go` — FTS search with weighted tsvector, AgentFirstFilter constant
- `internal/handlers/seo.go` — All GEO endpoints: robots, llms.txt, llms-full.txt, mcp.json, ai-plugin.json, openapi.yaml, sitemap.xml

## Agent-First Rule (CRITICAL)
Every site in the index MUST have at least one strong agent signal:
- has_structured_api, has_llms_txt, has_openapi, has_ai_plugin, has_mcp_server
- Schema.org and robots.txt ALONE do not qualify
- The `AgentFirstFilter` in queries.go enforces this in all queries
- API detection requires content verification for /docs and /developer paths (3+ indicators)

## Scoring (max 100)
llms.txt=25, ai-plugin=20, OpenAPI=20, API=15, MCP=10, robots.txt=5, schema.org=5

## Domains
- nothumansearch.ai (canonical)
- nothumansearch.com (301 redirects to .ai)
- nothumansearch.fly.dev (Fly default)

## Automation
- Daily recrawl: 6am via launchd `com.foundry.nothumansearch.recrawl`
- Script: `tools/recrawl.sh` — seeds, recrawl, IndexNow submission

## Common Operations
```bash
# Deploy
fly deploy --remote-only

# Add new seeds + crawl
fly ssh console -a nothumansearch -C "/app/crawler -seed -workers 10"

# Recrawl all (updates scores, categories, tags)
fly ssh console -a nothumansearch -C "/app/crawler -recrawl -workers 10"

# Crawl single site
fly ssh console -a nothumansearch -C "/app/crawler -url https://example.com"
```
