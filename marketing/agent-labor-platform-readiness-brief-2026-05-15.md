# Agent Labor Platform Readiness Brief

Status: prepared, not published.
Automation: `business-marketer-not-human-search`
Prepared: 2026-05-15T17:08-0700

## Boundary

No outreach, public post, browser action, account creation, product-code edit, deploy, full recrawl, checkout completion, or QLimit/global-queue write was performed. This is a sanitized scout artifact for a later gated channel or product operator.

No raw users, emails, API keys, checkout URLs, payment identifiers, private monitor rows, private score-fix rows, or private query logs were written.

## Fresh Evidence

Public surfaces checked during preparation:

- `https://nothumansearch.ai/api/v1/stats`: 4,176 indexed sites, average score 35, top category `developer`.
- `https://nothumansearch.ai/api/v1/categories`: 14 categories; `developer=1229`, `ai-tools=901`, `data=403`, `finance=200`, `jobs=26`.
- `https://nothumansearch.ai/.well-known/mcp.json`: advertises 11 tools.
- `https://nothumansearch.ai/api/v1/top?category=jobs&limit=8`: returns job and hiring platforms, including one Foundry-owned example and several third-party job platforms scoring 50-75.
- `https://nothumansearch.ai/api/v1/top?category=developer&limit=8`: returns developer/agent infrastructure examples with full agent-readiness surfaces.
- `https://nothumansearch.ai/api/v1/top?category=finance&limit=8`: returns finance/payment/data examples relevant to agent work and payout/payment infrastructure.
- `https://nothumansearch.ai/api/v1/catalog`: lists the score-fix product plus Starter, Pro, and Scale API products.

Aggregate admin evidence, last 7 days:

- MCP `tools/list`: 128,661 calls.
- MCP `initialize`: 17,311 calls.
- MCP `tools/call`: 341 calls.
- Top called tools: `search_agents=212`, `get_site_details=42`, `find_mcp_servers=24`, `get_stats=16`, `verify_mcp=15`, `check_url=13`, `get_top_sites=9`, `recent_additions=7`, `list_categories=2`, `submit_site=1`.
- Top-query themes included agent labor and work-market phrasing: `agent earn money freelance no upfront cost`, `agent jobs tasks gig economy platform`, and `K-Dense-AI scientific-agent-skills agent skills finance analysis`.

Aggregate admin traffic, last 336 hours:

- `/.well-known/commerce.json`: 1,439 requests.
- `/api/v1/catalog`: 321 requests.
- `/api/v1/checkout`: 301 requests.
- `/api/v1/quote`: 301 requests.
- `/top`: 138 requests.
- `/newest`: 111 requests.
- `/.well-known/mcp.json`: 91 requests.
- `/api/v1`: 88 requests.
- `/score`: 68 requests.
- Google referrers: 286 combined requests from `google.com` and `www.google.com`.
- Existing agent-builder placement `github.com/0xNyk/awesome-hermes-agent`: 28 requests.

Private workflow aggregates checked:

- Monitor status: `active=1`, `quarantined=1`, quarantine reason `bounded rerun still zero score`.
- Monitor actions in the last 30 days: `request_score_rerun=1`, `keep_quarantined=1`.
- Score-fix aggregate: 11 rows; real-candidate pending rows = 2; real paid or lead rows = 0.

## Read

Agents are not only searching for APIs and MCP servers. Some are searching for agent work markets: ways for agents to perform tasks, earn money, find freelance work, or use finance-analysis skills.

The safe owner-side angle is readiness, not labor-market validation. NHS should not claim that agent gig work is proven, profitable, legal, safe, or available. The useful marketing angle is that any platform trying to route tasks to agents needs public machine-readable contracts:

- Clear task or job listing APIs.
- OpenAPI or structured API docs.
- MCP surfaces when agent workflows are first-class.
- Machine-readable pricing, payout, eligibility, and unsupported-rail metadata.
- Stable categories and source profiles that agents can monitor.
- Free monitor paths for high-score owners and score-fix remediation for low-score owners.

The public `jobs` top list is small and includes a Foundry-owned example, so it should be framed as readiness-pattern evidence, not market coverage proof.

## Channel Brief

Short:

Agents are starting to search for work-market surfaces: freelance tasks, gig platforms, agent jobs, and finance-analysis skills. The owner-side takeaway is not that autonomous gig work is mature. It is that task platforms need probeable public contracts before agents can evaluate, route, or monitor them.

Long:

Agent labor platforms sit at the intersection of jobs, developer tools, payments, and source verification. Human-readable landing pages are not enough for autonomous workflows. Agents need structured task feeds, clear API boundaries, MCP or OpenAPI where relevant, pricing and payout metadata, and monitorable readiness.

NHS can help identify whether those public surfaces exist. It should stay away from claims about earnings, labor-market demand, task quality, legal compliance, or platform certification.

## Suggested Follow-Up

Prepare a gated channel packet for agent-work, AI freelancer, task marketplace, and developer-tool audiences:

- Use aggregate MCP query themes as the signal.
- Use public `jobs`, `developer`, and `finance` top lists as readiness-pattern examples.
- Label Foundry-owned examples if used.
- Keep the conversion path split: high-score site owners to free monitoring, low-score owners to score-fix remediation, API-heavy users to paid API plans.

## Publication Guard

Before any external use:

- Refresh `/api/v1/stats`, `/api/v1/categories`, `/api/v1/top?category=jobs`, `/api/v1/top?category=developer`, `/api/v1/top?category=finance`, `/llms.txt`, `/.well-known/mcp.json`, `/.well-known/commerce.json`, `/.well-known/agent.json`, `/api/v1/catalog`, and `/api/v1/admin/mcp?days=7`.
- Verify active channel account identity.
- Check `marketing/social-post-ledger.json` if a social/channel post is involved.
- Check sync-state public-action locks and `outreach/distribution_log.csv`.
- Avoid `modelcontextprotocol/*` and `punkpeye/*` surfaces from `unitedideas`.
- Do not claim private demand, completed payments, revenue, customer endorsement, earnings potential, legal compliance, task availability, platform safety, labor-market validation, paid ranking placement, preferred inclusion, ACP/x402 support for NHS, or score-methodology bypass.
