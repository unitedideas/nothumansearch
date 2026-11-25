# MCP server-discovery endpoint handoff

Date: 2026-07-31
Automation: `business-marketer-not-human-search`
Status: private product handoff only; no outreach, public post, account action,
browser action, product-code edit, deploy, crawl, monitor registration, checkout,
or global-queue write performed

## Segment

Make `find_mcp_servers` return the callable MCP endpoint and an honest cached
verification state. The tool currently identifies domains with an MCP signal,
but its list query does not hydrate the stored `mcp_endpoint`, and its text
response prints the site's origin URL rather than an endpoint that can be passed
to `verify_mcp`.

This is narrower than the existing broad MCP client, verification-directory,
search-result, and domain-detail handoffs. It closes the direct machine path
from discovery to a bounded endpoint probe.

## Fresh evidence

- Public stats report 4,349 indexed sites, average score 38, and `developer` as
  the top category.
- Seven-day aggregate MCP analytics recorded 49,722 `tools/list`, 14,319
  `initialize`, and 237 `tools/call` requests. `find_mcp_servers` does not appear
  in the current top tool-call table, so this is a product-contract repair, not
  evidence of demand, customers, or successful discovery usage.
- Live JSON-RPC `tools/list` exposes 11 tools, including `find_mcp_servers` and
  the bounded `verify_mcp` probe.
- The crawler stores an MCP endpoint in `models.Site.MCPEndpoint` when a manifest
  declares one or a common-path JSON-RPC probe succeeds.
- `models.SearchSites` does not select or scan `mcp_endpoint`. Because
  `toolFindMCPServers` uses `SearchSites`, its raw site objects cannot serialize
  the stored endpoint even though the model has that JSON field.
- The current text response includes name, score, domain, category, description,
  origin `URL`, and NHS `Report`; it does not identify the callable MCP endpoint
  or supply an invocation-ready `verify_mcp` next step.
- MCP presence does not always mean a live endpoint. A manifest may declare an
  endpoint whose probe failed, or may expose MCP metadata without a usable
  endpoint. The result contract must distinguish these states.
- `/health`, `/mcp`, `/.well-known/mcp.json`, `/.well-known/agent.json`,
  `/llms.txt`, `/openapi.yaml`, `/api/v1`, `/api/v1/catalog`, `/score`,
  `/monitor`, and `/report` returned HTTP 200.
  `/.well-known/agent-card.json` returned 404, so A2A positioning remains
  blocked.
- A full recrawl that started at 06:00 PT is still active. This scout did not
  deploy, recrawl, or compete with it.

No raw MCP queries, emails, private monitor rows, private score-fix rows,
checkout URLs, payment identifiers, API keys, IP data, or customer identifiers
were written to this artifact.

## Conversion hypothesis

An agent that asks NHS to find MCP servers should be able to take the next safe
step without parsing a homepage or guessing `/mcp`:

1. Return the stored callable endpoint when known.
2. Label whether it was live-verified, declared but not verified, or unavailable.
3. Offer an invocation-ready `verify_mcp` suggestion only when an endpoint is
   known.
4. Keep the list call fast and cached; do not probe every result automatically.
5. Link the NHS report for methodology and provenance without mixing in monitor
   registration or paid remediation before a domain owner has been established.

## Acceptance test

1. Add `mcp_endpoint` to the `SearchSites` select and scan path, preserving all
   existing fields and filters. Audit the relevant domain-detail read path for
   the same hydration gap so list and detail contracts do not diverge.
2. Return stable structured fields for `endpoint`, `endpoint_source`,
   `verification_status`, `last_crawled_at`, and `report_url` on each discovery
   result. Use bounded enums rather than prose-only status.
3. Print the callable endpoint in the text response when known. If none is
   known, say `endpoint unavailable`; do not present the origin URL as though it
   were an MCP endpoint.
4. When an endpoint is known, include an invocation-ready next step equivalent
   to `verify_mcp {"url":"<endpoint>"}`. Do not invoke it automatically.
5. Distinguish at least live-verified, manifest-declared/unverified, and
   endpoint-unavailable states. Do not label every indexed MCP signal as live.
6. Add unit coverage for a common-path live endpoint, a verified manifest
   endpoint, a failed manifest endpoint, a tools-only manifest without an
   endpoint, empty results, and text/structured backward compatibility.
7. After the active recrawl completes and a later product worker deploys the
   change, verify `tools/list`, an authenticated `find_mcp_servers` response,
   and the suggested `verify_mcp` chain against one bounded known fixture. Do
   not run a broad recrawl.
8. Track discovery delivery and explicit verification separately. A result,
   endpoint, or probe is not a customer, owner, endorsement, uptime guarantee,
   security certification, payment, or revenue event.

## Duplicate and claim boundary

The existing MCP client and verification artifacts cover channel positioning,
directory/operator qualification, or the general verification workflow. The
existing search-result and domain-detail handoffs cover NHS action routing. No
current artifact, inbox row, distribution-log entry, or social-ledger item is an
exact contract for hydrating the stored endpoint in `find_mcp_servers` and
handing it safely to `verify_mcp`.

Do not claim indexed MCP presence proves liveness, uptime, security, protocol
conformance, customer demand, adoption, endorsement, completed payments,
revenue, paid ranking, preferred inclusion, A2A support while the Agent Card
route is 404, or a score-methodology bypass.
