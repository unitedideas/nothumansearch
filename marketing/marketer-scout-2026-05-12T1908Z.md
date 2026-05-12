# NHS marketing scout segment - 2026-05-12T19:08Z

Automation: `business-marketer-not-human-search`

## Boundary

No outreach was sent. No posts, browser/Computer Use, account creation, deploys, full recrawls, product-code edits, public submissions, or QLimit/global-queue writes were performed.

## Live surface checks

- `https://nothumansearch.ai/api/v1/stats`: `total_sites=4238`, `avg_score=35`, `top_category=developer`.
- `https://nothumansearch.ai/api/v1/categories`: 14 categories. Largest buckets: `developer=1300`, `ai-tools=892`, `other=775`, `data=403`, `finance=201`.
- `/.well-known/mcp.json`: advertises 11 tools and describes public categories plus audit-only `other` and `spam`.
- `/llms.txt`: advertises 4238+ indexed sites, 11 MCP tools, and the public/audit-only category split.

## Public Vertical Evidence

These are public `/api/v1/top` results, not private admin/customer rows.

Healthcare owner-channel shortlist from `GET /api/v1/top?category=health&limit=8`:

- `zgts.in` - score 100, microneedling ecommerce/health surface with all 7 signals.
- `hipaaagent.ai` - score 80, HIPAA compliance product missing OpenAPI.
- `opdstar.com` - score 70, clinical documentation assistant missing OpenAPI, MCP, and Schema.org.
- `monarchinitiative.org` - score 65, cross-species biomedical data project missing ai-plugin, AI-friendly robots, and MCP.
- `fhirfly.io` - score 65, healthcare data APIs missing ai-plugin, AI-friendly robots, and MCP.
- `lakmesalon.in` - score 60, salon/skincare commerce surface missing ai-plugin and OpenAPI.
- `tau.edu.gy` - score 50, medical school surface missing ai-plugin, OpenAPI, and MCP.
- `skills-hub.ai` - score 50, AI skills catalog tagged healthcare but missing ai-plugin, OpenAPI, and MCP.

Education owner-channel shortlist from `GET /api/v1/top?category=education&limit=8`:

- `sourcelibrary.org` - score 100, ancient-text translation/library surface with all 7 signals.
- `coursera.org` - score 90, learning platform missing MCP among major implementation signals.
- `admit-coach.com` - score 70, college application platform missing ai-plugin and MCP.
- `quizapi.io` - score 65, quiz API missing ai-plugin, AI-friendly robots, and MCP.
- `slidemaster.tw` - score 65, AI course-design tool missing OpenAPI, AI-friendly robots, and MCP.
- `sansfiction.com` - score 55, digital library missing OpenAPI, structured API, and MCP.
- `samiolearning.com` - score 50, adaptive kids learning app missing ai-plugin, OpenAPI, and MCP.
- `nhavan.vn` - score 50, literary reading/education surface missing ai-plugin, OpenAPI, and MCP.

Jobs owner-channel shortlist from `GET /api/v1/top?category=jobs&limit=8`:

- `aidevboard.com` - score 100, Foundry-owned AI jobs board with all 7 signals.
- `jseek.co` - score 75, company-watchlist job alerts missing structured API and MCP.
- `himalayas.app` - score 65, remote job board missing OpenAPI, AI-friendly robots, and MCP.
- `ctojobshq.com` - score 50, executive job board missing ai-plugin, OpenAPI, and MCP.
- `reed.co.uk` - score 50, large UK job board missing ai-plugin, OpenAPI, and MCP.
- `upstaff.com` - score 50, engineering talent platform missing ai-plugin, OpenAPI, and MCP.
- `ziprecruiter.com` - score 50, large job marketplace missing ai-plugin, OpenAPI, and MCP.
- `micro1.ai` - score 45, AI training data workforce platform missing ai-plugin, OpenAPI, AI-friendly robots, and MCP.

## Draft Brief Angles

Healthcare:

`Healthcare is a useful owner-channel because the gap is not abstract AI positioning. NHS has 54 health sites, and the top list includes biomedical data, FHIR/data APIs, HIPAA compliance, clinical documentation, and healthcare education. Several already have llms.txt plus structured APIs; the missing pieces are usually OpenAPI, MCP, ai-plugin, or AI-friendly robots. That makes the owner pitch concrete: publish the public surfaces that let agents discover and verify the healthcare workflow safely.`

Education:

`Education has a small but high-signal NHS category: 21 indexed sites with an average score of 49. The top examples include a score-100 source library, a score-90 learning platform, college admissions, quiz APIs, course-design tools, and digital libraries. This is a good channel for showing that agent-readiness applies to content/library and learning platforms, not just developer tools.`

Jobs:

`The jobs category is compact at 26 indexed sites, but the owner gaps are obvious. AI Dev Board is the score-100 reference point; most other job boards have llms.txt and some API-like surface but miss MCP, OpenAPI, or ai-plugin. The owner-channel angle should be about making job data callable and monitorable by agents, not about general SEO.`

## Duplicate And Channel Checks

- `ops/sweeper/marketer-inbox.jsonl` had no existing healthcare, education, or jobs vertical brief rows before this segment.
- `outreach/distribution_log.csv` already has many broad NHS directory submissions and ADB/NHS job-board placements, so later publication should be channel-specific rather than another broad directory push.
- Shared social ledger grep found no current healthcare, education, or jobs owner-channel brief row for these exact category packets.
- No public action was taken, so no public-action lock was claimed. Any later publication still needs active account verification, duplicate fingerprinting, and a sync-state public-action lock.

## Appended Intake Rows

Rows appended to `ops/sweeper/marketer-inbox.jsonl`:

- `Prepare healthcare owner-channel brief from public top-category evidence.`
- `Prepare education owner-channel brief from public top-category evidence.`
- `Prepare jobs-platform owner-channel brief from public top-category evidence.`
