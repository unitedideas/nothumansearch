# NHS provider mechanism proof matrix

**Scope:** Not Human Search only

**Production revision checked:** `7ad7cbfde04b239b6e12a563e05f765de9701df8`

**Production OCI digest checked:** `sha256:df56436486523a17575f629430cece0c3efffb34bc99bc0efae9b491baffd6b1`

**Production mode checked:** `disabled`

**Public smoke:** 54/54 pass on 2026-08-12 UTC

**Commercial proof:** not created

This matrix separates implemented rails from observed business evidence. A
passing test or deployed endpoint proves only the named technical invariant. It
does not prove agent demand, a provider outcome, willingness to pay, revenue,
or a winning mechanism.

| Objective requirement | Required authoritative evidence | Current evidence | Status |
| --- | --- | --- | --- |
| Organic REST/MCP discovery remains free | Live search response says `access=free`; provider sidecar cannot affect organic rank | Exact-revision disabled smoke passes free REST and MCP checks | Proved current |
| Neutrality and privacy | Explicit false assertions for sold rank, queries, prompts, agent identity, and principal identity; no prohibited fields in the proof projection | Production projection and selector fail closed on missing or true assertions | Implemented, no commercial cohort |
| Explicit principal action interest | Genuine non-synthetic agent-surface receipt backed by a returned organic result | The post-text-action report at `2026-08-12T00:41:26Z` contains 158 nominally meaningful searches and 15 developer-tools searches, including two known operator audits that used non-canonical synthetic markers; selections and genuine action-interest receipts remain 0 | Discovery demand observed before correction; clean post-correction intent observation starts at this checkpoint |
| Consent before handoff | Immutable ticket authority plus separate handoff consent | Rail and rejection tests exist; no real pilot ticket | Implemented, unobserved |
| Attribution | One randomly assigned offer arm, immutable terms, exact ticket-to-outcome binding | Database and proof-verifier gates exist; no real pilot outcome | Implemented, unobserved |
| Accepted-handoff mechanism | Authenticated accepted outcome and exact available processor-net settlement for its own arm | Disposable PostgreSQL regression now drives the arm from consent-bound handoff through authentic outcome and available processor net; no real provider evidence | Rail proved, business proof unmet |
| Activated-CPA mechanism | Authenticated activation and exact available processor-net settlement for its own arm | Disposable PostgreSQL regression now drives the arm from consent-bound handoff through authentic outcomes and available processor net; no real provider evidence | Rail proved, business proof unmet |
| Converted-CPA mechanism | Authenticated conversion and exact available processor-net settlement for its own arm | Disposable PostgreSQL regression now drives the arm from consent-bound handoff through authentic outcomes and available processor net; no real provider evidence | Rail proved, business proof unmet |
| Billing and settlement | Signed paid webhook bound to the exact charged outcome, matching Stripe balance transaction, and a later availability receipt | All three arms traverse exact settlement-order and available processor-net aggregation in isolated PostgreSQL; production remains disabled and no provider has been charged | Rail proved, unobserved commercially |
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

## Cryptographic full-funnel delta

The comparator at revision
`37390802a59d259f0e80e79c484b848a6cd8748b` now accepts the sealed v2
post-selection receipt, verifies its report SHA-256, and emits one privacy-safe
comparison across the complete measurable funnel: free REST/MCP discovery,
result selection, active action-interest net state, provider activation,
returned offers, tickets, consented handoffs, authenticated outcomes, paid and
available settlements, and commercial-state events. Durable counters fail
closed if they regress. Active-interest fields are labeled as net changes
because receipts can expire; they are not misrepresented as event creation
counts.

The first production-database comparison covered
`2026-08-12T00:41:26.157992Z` through
`2026-08-12T00:54:51.695779Z`. It revalidated checkpoint SHA-256
`8bd891aa6bfac6f6fa12cb1832eb22a720b4017a1a82484c6fd1db98548bb03e`
and emitted current report SHA-256
`3da07d033095c28e11b1ab0b4e5321ba4444b562db38a7fbd6153665c5958169`.
Every funnel delta was zero. The receipt contains no identifiers, queries,
prompts, or contacts; explicitly says searches are not leads; and leaves the
strongest mechanism unselected. The disposable operator exited zero and was
destroyed. This is a zero-delta control, not demand or commercial proof.

## Receipt-bound MCP action correction

Production now returns one exact machine-readable `get_site_details` action for
every eligible result from a scoped MCP discovery call. Each action carries the
result domain and the originating `search_id`, states that following it records
a result selection, and explicitly states that action interest is not inferred,
no provider is contacted, and organic rank is unaffected. Unscoped browsing
returns no receipt-bound actions.

The same exact call is now rendered beside every result in the MCP text content
for clients that do not expose `structuredContent` to their model. The text
states that the call records selection only and does not infer interest or
contact a provider. The public smoke fails unless the text action's domain and
`search_id` match the structured discovery receipt.

The exact archive gate passed against two disposable PostgreSQL instances. The
OCI was built from that archive and pushed at the digest above. A traffic-held
256 MB canary reached exact-revision health in disabled mode and was removed.
The two production machines were then updated by digest. The final topology is
one healthy started machine and one stopped machine, both pinned to the exact
digest; Fly autostop may rotate their IDs. At `2026-08-12T00:22:25Z`,
`e2869126f75686` was started and `e2869124a74686` was stopped. Public smoke
passed 54/54, and a synthetic MCP search returned three results and three exact
receipt-bound detail actions.

The new sealed comparison baseline is
`NHS_POST_TEXT_ACTION_BASELINE_2026-08-12_004126Z_7ad7cbf.json`, checked at
`2026-08-12T00:41:26.157992Z` with report SHA-256
`8bd891aa6bfac6f6fa12cb1832eb22a720b4017a1a82484c6fd1db98548bb03e`.
It contains 158 nominally meaningful eligible searches: 25 MCP, 133 REST, and
15 in the developer-tools topic. It still contains zero result selections, explicit
action-interest receipts, provider activations, handoffs, outcomes, payments,
or settlements. This is the correct zero baseline for measuring whether text-
only and structured clients follow the exact action; it is not a monetization
win.

Two operator MCP audits before this checkpoint used header values other than
the deliberately narrow canonical `NHS-Synthetic-Test: deploy-smoke` marker.
Those two calls are included in the report's nominal meaningful and developer-
tools counters even though they are not demand. The immutable aggregate is not
rewritten or causally relabeled because its privacy boundary provides no row-
level request join. Future delta measurement begins after the checkpoint, so
the contamination is fully on the baseline side of the comparison.

## Sealed post-selection checkpoint

The current comparison baseline is
`NHS_AGENT_INTEREST_CHECKPOINT_2026-08-11_235231Z_a025939.json`, checked at
`2026-08-11T23:52:31.910135Z` with report SHA-256
`a376d2600cbe204e7dcfef287eac2a2937a526e5b7c76fec546387142331d166`.
It contains 148 meaningful eligible searches: 15 MCP, 133 REST, and 13 whose
demand topic was developer tools. It contains zero result selections, explicit
action-interest receipts, provider activations, handoffs, outcomes, payments,
or settlements. The prior 20 boundary-spanning unavailable attempts did not
increase. They remain excluded from demand.

This is evidence that agents use free discovery, not evidence that a principal
wants a provider action or that any provider-funded mechanism will convert.
Counts are search receipts, not unique agents. The current monetizable-intent
rate is unobserved at zero; the system must not relabel discovery as a lead.

## Current agent-only readiness

The last authenticated Stage 1 aggregate at `2026-08-11T08:30:56Z` contains 538 meaningful
MCP/REST search receipts, 20 selections, 7 search receipts with a selection, 32
`developer-tools` receipts, and zero action-interest receipts. The observation
span is six days. Stage 1 remains false because the 14-day window, 20 distinct
selected-search receipts, and 10 genuine action-interest-backed search receipts
are unmet. The newer sealed post-selection checkpoint above is the authoritative
baseline for incremental action-interest measurement. The registered Fly
credential bridge was restored by running the approved injector outside the
sandbox; no token value was exposed.
