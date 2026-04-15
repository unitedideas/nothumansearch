---
name: NHS monetization verdict
description: Ranked monetization options for nothumansearch.ai evaluated 2026-04-14
type: project
---

Monetization priority order decided 2026-04-14:

1. **Paid /check API for CI/CD gates** — GREEN. Build first. API key + rate limit + Stripe subscription around the existing score endpoint. $29/mo per org, $99/mo with SLA. Schema requires no changes. Key constraint: /check must return last-known score + requeue, not synchronous crawl, to stay <2s for CI pipelines.

2. **Verified Agent-Ready badge** — YELLOW (with modification). Build second. `is_verified` + `is_featured` columns already exist in migrations. Paid verification must mean functionally tested (MCP endpoint actually responds), not just presence-detected. Otherwise indistinguishable from free score. $199/yr per site.

3. **Enterprise analytics tier** — ORANGE (year-2). "What agents queried for your competitors" is genuinely valuable but unsellable until 6+ months of real query logs exist. Revisit Q4 when volume data is present.

**Why:** Agents are the primary users — monetization must fall on site owners and commercial buyers, not on agents searching. Sponsored/pinned results RED — destroys the merit-based ranking that is the product's core value.

**How to apply:** When building NHS payment features, evaluate against this order. Do not start analytics tier before query log volume is meaningful.
