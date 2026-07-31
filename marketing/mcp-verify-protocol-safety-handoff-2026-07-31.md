# MCP Verify Protocol and Safety Handoff — 2026-07-31

Automation: `business-marketer-not-human-search`

Scope: private product-trust handoff only. No probe was sent to a third-party
endpoint, and this run made no product change, deploy, public post, outreach,
account creation, monitor registration, score-fix action, or broad crawl.

## Why this blocks promotion

NHS markets `verify_mcp` as an active check for a "live, spec-compliant MCP
server" and the registry description calls it part of a "probe-before-use"
workflow. The current implementation does not support that strength of claim:

- it sends `tools/list` before `initialize`, so a compliant server that requires
  the MCP initialization handshake can be returned as `verified: false`;
- it treats JSON-RPC errors `-32600` (invalid request) and `-32601` (method not
  found) as `verified: true`, although those errors do not prove the endpoint is
  an MCP server or that its tools are callable;
- it reduces all other outcomes to one boolean, conflating unreachable,
  authentication-required, handshake-required, invalid response, non-MCP, and
  protocol failure;
- the tool accepts any URL, follows redirects, and calls it from the server
  without the private-host, resolved-address, userinfo, or redirect-destination
  protections used by the monitor boundary;
- the MCP tool has no annotations, while the live description explicitly says
  it performs an outbound probe;
- the shared MCP logging path records the full URL argument in both request and
  intent analytics, so credentials, signed query parameters, fragments, and
  private endpoint paths can be retained.

The same `crawler.ProbeMCPJSONRPC` helper is used by the public MCP tool, the
REST peer, and crawler discovery. A repair therefore needs explicit caller
modes: public user-supplied probing requires strict SSRF and telemetry controls,
while crawler-origin probing still needs correct protocol negotiation and
bounded evidence states.

## Evidence inspected

- Live `GET /api/v1/stats`: `4,343` indexed sites, average score `38`, top
  category `developer`.
- Live MCP `tools/list`: `verify_mcp` says it probes any URL and verifies a
  "live, spec-compliant" server; `annotations` is absent.
- Source `internal/crawler/crawler.go`: one unauthenticated `tools/list` request,
  no `initialize`, six-second client timeout, up to 64 KiB response, and
  `-32600`/`-32601` accepted as success.
- Source `internal/handlers/mcp.go` and `internal/handlers/api.go`: both expose
  the boolean result; failure copy admits the missing initialization handshake,
  while success copy still asserts MCP compliance.
- Source URL boundary: monitor registration rejects private-address shapes, but
  `verify_mcp` does not use that validation and the probe client follows
  redirects without revalidating destinations.
- Aggregate-only MCP analytics, seven days: `tools/list=50,513`,
  `initialize=14,706`, `tools/call=237`. No nonzero first-party `verify_mcp`
  call was present; ten one-call `__verifymcp_auth_probe_*` names are scanner
  behavior, not verified tool use or demand.
- Live surfaces: `/llms.txt`, `/openapi.yaml`, `/.well-known/mcp.json`,
  `/score`, `/monitor`, and `/report` return 200;
  `/.well-known/agent-card.json` returns 404. Anonymous
  `/api/v1/verify-mcp` remains behind the current API subscription boundary.
- `outreach/distribution_log.csv` already records older public promotion of the
  verification claim. The portfolio social ledger exists at
  `/Users/owlassist/foundry-businesses/8bitconcepts/marketing/social-post-ledger.json`;
  no post was queued or fingerprint created in this run.

## Product contract

1. Parse and normalize the target URL before any network call. Allow only HTTP
   and HTTPS, reject userinfo, strip fragments, place a documented limit on path
   and query length, and never store raw query values.
2. Resolve every target and redirect destination. Reject loopback, private,
   link-local, multicast, unspecified, metadata, and other non-public address
   ranges for IPv4 and IPv6. Revalidate after every redirect and protect against
   DNS rebinding by binding validation to the address actually dialed.
3. Run the MCP handshake in order: `initialize` with an explicit supported
   protocol version, validate the response and negotiated version, send
   `notifications/initialized` when appropriate, preserve any negotiated
   session header, then call `tools/list`.
4. Return a stable evidence state instead of a bare compliance assertion. At
   minimum distinguish `verified_tools`, `auth_required`,
   `handshake_required_or_incompatible`, `reachable_non_mcp`,
   `invalid_mcp_response`, `unreachable`, `timeout`, and `blocked_target`.
5. Reserve `verified: true` for a valid initialized MCP session followed by a
   valid `tools/list` result. A generic JSON-RPC error is evidence only for its
   bounded error class, not MCP compliance.
6. Preserve backward-compatible `verified`, `endpoint`, and `note` fields while
   adding `status`, `protocol_version`, `http_status_class`, `tool_count`, and a
   bounded `evidence` array. Never echo credentials, response bodies, session
   tokens, cookies, authorization challenges, or internal network details.
7. Add MCP annotations that disclose the outbound network action and its
   idempotent, non-destructive intent. Keep the REST/OpenAPI, MCP schema,
   `llms.txt`, registry description, and discovery manifests semantically
   coherent.
8. At the shared telemetry boundary, retain only normalized scheme, hostname,
   safe port class, coarse status, duration bucket, and tool-count bucket.
   Remove userinfo, path unless explicitly allowlisted, query, fragment,
   response body, auth headers, session ids, and arbitrary arguments from both
   MCP request and intent logs.
9. Add tests for initialization-required servers, session headers, SSE and JSON
   responses, auth-required responses, protocol mismatch, malformed JSON-RPC,
   `-32600`/`-32601`, redirects, redirect-to-private, DNS rebinding, every
   private/reserved IPv4 and IPv6 range, userinfo, signed queries, fragments,
   timeouts, response limits, analytics redaction, and REST/MCP parity.
10. After a later product-worker deploy, verify the contract only against
    Foundry-owned fixtures representing each state. Do not probe arbitrary
    third parties or run a broad crawl for acceptance.

## Marketing gate

Do not promote `verify_mcp`, `find_mcp_servers -> verify_mcp`, or the
"probe-before-use" registry line until the protocol, SSRF, annotation, and
telemetry acceptance checks pass. Existing directory metadata can remain
unchanged until the next substantive registry maintenance cycle, but new copy
must not claim protocol compliance, security certification, endpoint safety,
uptime, authentication compatibility, customer demand, real-user tool success,
adoption, endorsement, completed payments, revenue, A2A support while Agent
Card remains 404, paid ranking, preferred inclusion, or score-methodology
bypass.

## Acceptance proof

- Unit and integration fixtures show a compliant initialization-required server
  reaches `verified_tools`, while a generic JSON-RPC endpoint returning
  `-32600` or `-32601` does not become verified.
- Public-target validation blocks every private/reserved address and any public
  redirect that lands on one; tests cover IPv4, IPv6, and DNS rebinding.
- Captured aggregate analytics contain no path secrets, query values,
  fragments, userinfo, authorization data, session tokens, raw arguments, or
  response bodies.
- Live `tools/list`, the REST/OpenAPI peer, `llms.txt`, MCP discovery metadata,
  and the registry description state the same bounded verification semantics.
- A Foundry-owned fixture passes the full initialize/session/tools-list flow;
  no third-party probe or broad recrawl is part of acceptance.
