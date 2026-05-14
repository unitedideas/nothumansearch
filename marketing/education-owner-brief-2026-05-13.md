# NHS Education Owner Brief

Status: prepared, not published.
Automation: `business-marketer-not-human-search`
Prepared: 2026-05-13T22:14Z

## Boundary

No outreach, public post, browser action, account creation, product-code edit,
deploy, full recrawl, or QLimit/global-queue write was performed. This artifact
is a channel brief for a later gated operator. External use still requires active
account verification, duplicate-fingerprint checks, and a sync-state
public-action lock.

No raw user identifiers, private customer rows, API keys, checkout URLs, private
query logs, learner data, payment identifiers, or enrollment claims were written.

## Live Evidence

Public surfaces checked during preparation:

- `https://nothumansearch.ai/api/v1/stats`: `total_sites=4169`, `avg_score=35`,
  `top_category=developer`.
- `https://nothumansearch.ai/api/v1/categories`: 14 categories. The education
  bucket has `21` sites with average score `49`.
- `https://nothumansearch.ai/api/v1/top?category=education&limit=8`: the top
  education example scores 100/100 and exposes all seven public
  agent-readiness signals.
- `https://nothumansearch.ai/llms.txt`: education is listed as a public
  category; `other` and `spam` are audit-only.
- `https://nothumansearch.ai/.well-known/mcp.json`: education appears in the
  public category parameter copy.

Public top-category examples observed during preparation:

- `sourcelibrary.org` - score 100, translated ancient-texts library with all
  seven public agent-readiness signals.
- `coursera.org` - score 90, course and credential platform with `llms.txt`, AI
  plugin, OpenAPI, structured API, AI-friendly robots, and Schema.org, but no
  MCP server in the current crawl.
- `admit-coach.com` - score 70, college-application platform with `llms.txt`,
  OpenAPI, structured API, AI-friendly robots, and Schema.org, but missing AI
  plugin and MCP in the current crawl.
- `quizapi.io` - score 65, developer-first quiz API with `llms.txt`, OpenAPI,
  and structured API, but missing AI plugin, AI-friendly robots, MCP, and
  Schema.org in the current crawl.

## Brief Copy

Subject/heading:

`21 education sites are in Not Human Search. The strongest ones publish machine-readable learning surfaces.`

Short post:

Not Human Search currently tracks 21 education sites in the public education
bucket.

The useful pattern is not generic edtech copy. It is whether a learning,
library, course, quiz, or admissions product exposes the public surfaces an
agent can verify before recommending or using it.

The top education example exposes the full public agent-readiness surface:
`llms.txt`, an AI plugin manifest, OpenAPI, a structured API, MCP, an
AI-friendly robots policy, and Schema.org.

The next tier is where the owner-side work is visible. Course platforms,
learning apps, quiz APIs, reading libraries, and admissions tools often have
some machine-readable surface already, but one or two pieces are missing:
OpenAPI, a maintained AI plugin manifest, MCP where bounded operations exist,
or an AI-friendly robots policy.

For education product owners, the practical checklist is:

1. Publish `llms.txt` with scope, content boundaries, API links, and support
   paths.
2. Keep OpenAPI current for non-sensitive operational endpoints such as course
   search, catalog lookup, quiz generation, content metadata, or progress-safe
   read APIs.
3. Add MCP only for bounded, auditable operations where an agent should act.
4. Keep `/.well-known/ai-plugin.json` pointed at the maintained API surface.
5. Register monitoring after the public evidence path is fixed so deploys do not
   silently remove the signals.

Search the public education bucket:

`https://nothumansearch.ai/api/v1/top?category=education&limit=25`

Check an education site:

`https://nothumansearch.ai/score`

## Owner/Buyer Angle

This is for learning platforms, libraries, course marketplaces, quiz APIs,
admissions products, and education SaaS teams. The sell is not ranking
placement, enrollment growth, certification, or endorsement. The sell is making
the public product surface legible to agents and catching drift before
integrators or students run into it.

## Publication Guard

Before external use:

- Refresh `/api/v1/stats`, `/api/v1/categories`,
  `/api/v1/top?category=education&limit=8`, `/llms.txt`, and
  `/.well-known/mcp.json`.
- Check the shared social ledger for duplicate fingerprints.
- Verify the active account identity for the selected channel.
- Take a sync-state public-action lock.
- Do not claim private demand, revenue, conversion, enrollment growth,
  educational endorsement, paid ranking placement, preferred inclusion, or
  score-methodology bypass.
