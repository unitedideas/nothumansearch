# Free agent discovery evidence brief

Date: 2026-08-01

## Surface and conversion contract

- Category: high-volume information.
- Audience: AI agents, developers operating agents, and humans evaluating agent-ready APIs and services.
- Primary conversion: complete a relevant free search, inspect a result, and progress to a verified provider action when one is useful.
- Conversion-quality metric: search-to-detail selection rate, paired with zero-result rate and later provider-accepted action rate. Raw searches, page views, MCP initialization, and tool-list calls are not success by themselves.
- Main question: "Which service can my agent actually use for this task?"
- Main anxiety: stale or commercially biased results, hidden access fees, and endpoints that claim compatibility but do not work.
- Proof required before action: neutral organic ordering, visible readiness signals and scores, canonical provider URLs, result freshness, free-access disclosure, and explicit rate-limit behavior.

## Evidence classification

### Measured evidence

- The deployed production index reports 4,344 indexed sites.
- The post-deploy smoke suite passed 39/39 checks, including a committed query-free search receipt and a successful search-to-detail selection receipt.
- Unauthenticated production REST search returns `200`, `access: free`, and `X-RateLimit-Tier: free`; unauthenticated MCP `search_agents` returns a result and search receipt without a tool error.
- Production rejects a loopback live-check target with `400` before crawling it.
- The provider-demand report returned zero real Razorpay receipts after multiple synthetic Razorpay smoke searches and declares `synthetic_excluded: true`, confirming deployment tests do not become commercial demand evidence.
- A true 390 CSS-pixel mobile emulation reported document and body widths of exactly 390 pixels; inspected home, populated-search, and result-card states have no horizontal overflow.

These observations prove the free-discovery and receipt layer is live. They do not prove provider willingness to pay or future conversion.

### Standards and durable principles

- WCAG 2.2 AA is the accessibility target.
- Search and navigation remain keyboard-operable with visible focus, explicit labels, and sufficient contrast.
- Organic readiness score and order are not purchasable. Any future provider-funded inventory must be returned in a separate, machine-readable, disclosed object.
- Canonical provider URLs remain available for free. A future NHS action path must win through verification, consent, and receipts rather than forced intermediation.
- Search telemetry stores short-lived pseudonymous receipts and result positions. Provider reporting may expose only thresholded aggregates, never raw prompts, IP hashes, user-agent strings, or alleged agent identities.

### Observed category patterns

- High-volume directories put the query and core filters in the first screen, show enough evidence to compare results, and keep promotional inventory visually and semantically separate.
- Agent-facing APIs need deterministic JSON contracts, explicit rate-limit headers, stable receipt identifiers, and a direct next action more than decorative interface density.

### Hypotheses to test

- Removing the paywall will produce meaningful search and detail-selection behavior that is currently suppressed.
- A search receipt carried into the detail request will measure real consideration without trapping the user behind a redirect.
- Providers will value thresholded missed-demand and selection reporting only after enough free discovery volume exists.

## Decision path

1. Query or filter the index without login or payment.
2. Compare neutral results using readiness score, signals, description, category, and domain.
3. Open a result detail page or call `get_site_details`, carrying the search receipt.
4. Inspect the full readiness report and canonical provider link.
5. In a later bounded pilot, optionally request a verified provider action with principal consent and explicit commercial disclosure.

## First-screen and responsive requirements

- The search control and value proposition remain visible without scrolling at a realistic desktop viewport.
- Mobile preserves the query, submit action, result name, score, and signal explanation without horizontal scrolling.
- Removing the subscription wall must not leave an empty promotional gap or introduce a competing primary CTA.
- Free access and abuse limits are described in plain language in agent contracts; no login prompt interrupts discovery.

## Release review gate

Approval requires current-source desktop and mobile renders, inspection of the first screen and a populated search state, keyboard/focus and contrast review, live REST/MCP contract checks, and a design contract bound to the exact `templates/home.html` source hash. Any later template edit invalidates that approval.
