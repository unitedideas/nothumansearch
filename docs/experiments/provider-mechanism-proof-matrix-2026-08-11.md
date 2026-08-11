# NHS provider mechanism proof matrix

**Scope:** Not Human Search only

**Production revision checked:** `e5dc7f03d07b69c8ef824d4740ad38743fac7ff4`

**Production mode checked:** `disabled`

**Public smoke:** 54/54 pass on 2026-08-11

**Commercial proof:** not created

This matrix separates implemented rails from observed business evidence. A
passing test or deployed endpoint proves only the named technical invariant. It
does not prove agent demand, a provider outcome, willingness to pay, revenue,
or a winning mechanism.

| Objective requirement | Required authoritative evidence | Current evidence | Status |
| --- | --- | --- | --- |
| Organic REST/MCP discovery remains free | Live search response says `access=free`; provider sidecar cannot affect organic rank | Exact-revision disabled smoke passes free REST and MCP checks | Proved current |
| Neutrality and privacy | Explicit false assertions for sold rank, queries, prompts, agent identity, and principal identity; no prohibited fields in the proof projection | Production projection and selector fail closed on missing or true assertions | Implemented, no commercial cohort |
| Explicit principal action interest | Genuine non-synthetic agent-surface receipt backed by a returned organic result | 0 genuine action-interest receipts at `2026-08-11T08:30:56Z` | Unmet |
| Consent before handoff | Immutable ticket authority plus separate handoff consent | Rail and rejection tests exist; no real pilot ticket | Implemented, unobserved |
| Attribution | One randomly assigned offer arm, immutable terms, exact ticket-to-outcome binding | Database and proof-verifier gates exist; no real pilot outcome | Implemented, unobserved |
| Accepted-handoff mechanism | Authenticated accepted outcome and exact available processor-net settlement for its own arm | Synthetic fixtures only | Unmet business proof |
| Activated-CPA mechanism | Authenticated activation and exact available processor-net settlement for its own arm | Synthetic fixtures only | Unmet business proof |
| Converted-CPA mechanism | Authenticated conversion and exact available processor-net settlement for its own arm | Synthetic fixtures only | Unmet business proof |
| Billing and settlement | Signed paid webhook bound to the exact charged outcome, matching Stripe balance transaction, and a later availability receipt | Rails are deployed and fail closed; no provider has been charged | Implemented, unobserved |
| Bounded 3/5/2/1 revenue milestone | 3 verified providers, 5 accepted handoffs, 2 activations, 1 genuine renewal, and available settlement receipts | Provider activation is owner-gated and has not been authorized | Owner-gated, unmet |
| Strongest mechanism selection | Closed mature three-arm cohort; every arm independently meets provider, offer-return, charged-event, paid-settlement, reversal, time-to-cash, margin, retained-value, economic-lead, and simultaneous-confidence gates | Selector contract v5 consumes actual available processor fee/net plus privacy-safe dispersion aggregates and leaves overlapping intervals empty; no real cohort exists | Correctly empty |

## What can be extracted

NHS can charge a provider only for the immutable downstream event assigned to
that offer: accepted handoff, verified activation, or verified conversion. It
cannot charge the principal or sell discovery, rank, query, prompt, or identity
data. The current 31-day NHS-only infrastructure contribution hypotheses are
`$2.70` per accepted handoff at five events, `$6.28` per activation at two
events, or `$12.23` for one conversion. Those figures are floors based on
observed Fly usage and a published processing-fee allowance; they are not
offers, profit, or settlement evidence.

## Selection boundary

The winner remains empty until each arm has real available processor net. The
verified selector does not accept a published processing rate. It compares the
exact Stripe-observed available net per 1,000 returned offers, using paid
settlement latency only as a tie-breaker after all declared constraints pass.
The top arm must also beat the runner-up by at least 1,157 processor-net cents
per 1,000 returns and 20%. This prevents a high-price, low-exposure arm, modeled
fee assumption, or economically immaterial near tie from appearing strongest.
Policy v3 additionally requires the leader's 95% simultaneous empirical-
Bernstein lower bound to exceed the runner-up's upper bound. The proof exports
only each arm's maximum bounty, net sum of squares, and maximum net—not ticket,
provider, query, prompt, or identity data. Its unit is randomized returned-offer
opportunities, explicitly not unique agents.

No provider was contacted, invited, enrolled, activated, or charged while
producing this matrix. Production provider exchange remains disabled.

## Current agent-only readiness

The production aggregate at `2026-08-11T08:30:56Z` contains 538 meaningful
MCP/REST search receipts, 20 selections, 7 search receipts with a selection, 32
`developer-tools` receipts, and zero action-interest receipts. The observation
span is six days. Stage 1 remains false because the 14-day window, 20 distinct
selected-search receipts, and 10 genuine action-interest-backed search receipts
are unmet. Nine diagnostic attempts are explicitly non-demand: one invalid MCP
request and eight disabled-mode REST refusals.
