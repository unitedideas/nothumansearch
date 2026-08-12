# NHS provider mechanism proof matrix

**Scope:** Not Human Search only

**Production revision checked:** `aa690c95e2ccccda4b4d6be7e0c87afb96f9144a`

**Production OCI digest checked:** `sha256:9b84c99d43a064e0d7bddd12b924912c58748a73e664ff765606222877c36434`

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
| Selection-design feasibility | Preregistered cohort can satisfy both the per-arm evidence floors and a sufficient confidence/power design at plausible eligible traffic | Executable audit proves the 3/5/2/1 milestone cannot satisfy the three-arm floors; policy v4's zero-variance necessary confidence floor at its smallest decisive lead is 446,360 returns per arm | Revenue-proof pilot is bounded; winner-selection design is not ready |

## What can be extracted

NHS can charge a provider only for the immutable downstream event assigned to
that offer: accepted handoff, verified activation, or verified conversion. It
cannot charge the principal or sell discovery, rank, query, prompt, or identity
data. The current 31-day NHS-only infrastructure contribution hypotheses are
`$2.68` per accepted handoff at five events, `$6.22` per activation at two
events, or `$12.12` for one conversion. Those figures are floors based on
observed Fly usage and a published processing-fee allowance; they are not
offers, profit, or settlement evidence.

## Selection boundary

The winner remains empty until each arm has real available processor net. The
verified selector does not accept a published processing rate. It compares the
exact Stripe-observed available net per 1,000 returned offers, using paid
settlement latency only as a tie-breaker after all declared constraints pass.
The top arm must also beat the runner-up by at least 1,146 processor-net cents
per 1,000 returns and 20%. This prevents a high-price, low-exposure arm, modeled
fee assumption, or economically immaterial near tie from appearing strongest.
Policy v4 additionally requires the leader's 95% simultaneous empirical-
Bernstein lower bound to exceed the runner-up's upper bound. The proof exports
only each arm's maximum bounty, net sum of squares, and maximum net—not ticket,
provider, query, prompt, or identity data. Its unit is randomized returned-offer
opportunities, explicitly not unique agents.

No provider was contacted, invited, enrolled, activated, or charged while
producing this matrix. Production provider exchange remains disabled.

The current Cost Explorer baseline covers 2026-07-13 through 2026-08-12 and
attributes `$4.36` to `nothumansearch` plus `$7.10` to `nothumansearch-db`, or
`$11.46` total for 31 days. It is a live usage estimate rather than a final
invoice and excludes labor, support, fraud, tax, and profit. The equal-cost
three-arm experiment floors are `$1.11` per accepted handoff at five events,
`$2.28` per activation at two events, and `$4.25` for one conversion. These are
pre-pilot pricing hypotheses, not offers or commercial evidence.

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

The same exact OCI digest was then rotated across both production machines
with `NHS_PROVIDER_EXCHANGE_MODE=disabled`. Both Fly service checks reported
the exact revision, a healthy database, and disabled provider mode. The public
production suite passed 54/54, including free REST/MCP discovery, exact
receipt-bound text actions, an empty paid sidecar, synthetic/invalid interest
rejection, and disabled provider-funded routes. Machine autostart and autostop
were explicitly restored after the direct image updates. No provider or
commercial state was activated.

The first post-cutover comparison covered
`2026-08-12T00:54:51.695779Z` through
`2026-08-12T01:11:02.979061Z` and advanced the sealed report hash to
`fc66efce95fea23d2a493ebc65a107f4d069329c9c3f890a8488ba473e9e92eb`.
It observed four additional non-synthetic MCP discovery receipts, zero
developer-tools receipts, zero result selections, and zero active-interest net
change. Handoffs, outcomes, settlements, and commercial-state deltas also
remained zero. The two unavailable attempt increments were the expected
disabled-mode release-smoke probes and are explicitly not demand. The added
search receipts show free discovery usage only; they are events rather than
unique agents and cannot be sold or relabeled as leads. No mechanism can yet
be selected.

The live no-write agent-evaluation dry-run also passed the strict production
`record_action_interest` schema check. The six-scenario Responses API behavior
run remains unexecuted because the canonical `OPENAI_API_KEY` reference does
not resolve from Keychain, 1Password, injected environment, or the
non-sensitive registry. No alternate credential was substituted. This leaves
model comprehension unproved but does not affect the production funnel
receipt; neither result is commercial evidence.

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

The release-bound read-only Stage 1 operator at
`2026-08-12T01:27:09.111765Z` contains 704 meaningful MCP/REST search receipts,
20 selection events on seven distinct search receipts, and zero explicit
action-interest receipts. Its threshold-qualified candidate topics are
`ai-tools` 118, `search` 54, `developer-tools` 47, and `storage` 21. Counts are
receipts rather than unique agents, and topic buckets may overlap.

Stage 1 report SHA-256
`dc549c4199bfe0665136d2ca158f99c290efb85590619b4a736cbf31fa1a2c2a`
and attempt-funnel SHA-256
`b47330cccf73061c24bfbd9248930b48be18ae22a87937bfb66f57bdc63236a4`
both independently revalidate. The 586,990-second observation span is only six
complete days. Stage 1 remains false because the 14-day window, 20 distinct
selected-search receipts, and 10 genuine action-interest-backed search receipts
are unmet. Developer-tools is a candidate, not an owner selection or pilot
authorization.

The exact archive passed the complete release gate against two temporary local
PostgreSQL 17 instances before its 40 MB OCI was pushed. One disposable 256 MB
reader exited zero and was destroyed. Both production machines were then
rotated to the exact digest with provider mode disabled; Fly health passed on
both and the public production suite passed 54/54. Autostart, autostop, and a
one-machine minimum remain configured. No provider was contacted and no
commercial state changed.

## Rolling Stage 1 checkpoint comparison

Revision `aa690c95e2ccccda4b4d6be7e0c87afb96f9144a` adds a sealed Stage 1
checkpoint comparator. It cryptographically revalidates the prior Stage 1 and
attempt-funnel projections, requires the same Stage 1 epoch and rolling window,
and emits aggregate net changes only. It contains no agent, principal, search,
domain, query, prompt, contact, or request coordinates. Net changes may be
negative when old rows expire and are explicitly not represented as newly
created event counts.

The first comparison covered `2026-08-12T01:27:09.111765Z` through
`2026-08-12T01:44:46.13295Z`. Meaningful search, selection, and explicit-interest
net changes were all zero. The MCP invalid-attempt bucket stayed flat; the REST
unavailable bucket increased by two. Those two attempts are disabled-mode
operational probes, not unique agents, demand, leads, or commercial proof.
Stage 1 remains unready at 704 meaningful search receipts, seven distinct
selected-search receipts, zero explicit-interest search receipts, and six
complete observation days.

The exact archive passed full tests, race tests, vet, build, protected migration
checks against two disposable PostgreSQL 17 instances, disabled recovery smoke,
and secret scan. Its 41 MB OCI was pushed and deployed by exact digest with
`NHS_PROVIDER_EXCHANGE_MODE=disabled`; production then passed 54/54 public
checks. The reader machine exited zero and was destroyed. No provider was
contacted, invited, activated, or charged, and the mechanism winner remains
empty.

## Live processor boundary

At `2026-08-12T02:01:36Z`, a read-only Stripe balance metadata call proved that
the Keychain-backed processor credential is live-mode. Production Fly secret
metadata contains both `STRIPE_SECRET_KEY` and `STRIPE_WEBHOOK_SECRET`, while
the provider signing-key names remain absent because the exchange is disabled.
A live Stripe search for PaymentIntents carrying product metadata
`nhs_provider_outcome_settlement` returned zero objects with no additional
page. No Stripe object identifiers or customer/provider data were retained.

Because the only resolved processor credential is live-mode, the proposed
write-side sandbox drill failed closed before creating an object. This proves
configuration and the absence of tagged live provider-settlement intents; it
does not prove a successful provider payment, webhook, processor fee, available
net, or mechanism winner. A future processor write drill requires a separately
provisioned test-mode credential and test webhook secret, isolated from
production. Provider exchange remains disabled and charges created remain zero.

## Selection-to-interest client contract

The first Stage 1 funnel has seven distinct selected-search receipts but zero
explicit-interest receipts. The rail itself is reachable and privacy-safe, but
the MCP initialize message previously named only search and detail tools. Its
post-selection text told text-first agents to call `record_action_interest`
without carrying the fixed search receipt, attestation, and confirmation-version
fields required by that tool. Structured clients could reconstruct those fields
from the tool inventory; text-first clients had a weaker transition.

The additive correction introduces `action_interest.call_contract` on REST and
MCP discovery/detail responses. It supplies the fixed fields only under the
published invocation condition, enumerates the eligible organic domains and
controlled action types, keeps the exact five-field allowlist, and says it is
not executable without explicit current-principal intent. MCP initialization
and text-only guidance now point to this same contract. It never accepts query,
prompt, contact, or identity data; records neither provider contact nor a
ticket/charge; and cannot affect organic rank. Synthetic production smoke must
see an unavailable empty contract, while disposable disabled-recovery smoke
must see an exact receipt-bound available contract. This fixes measurement
clarity; it does not turn a selection into interest or create demand evidence.

### Production correction and sealed baseline

Revision `9589816820a213e9a2ff77a4ff95f832a52ab088` passed the exact
release gate, including full tests, race tests, vet, two disposable PostgreSQL
17 checks, disabled-mode recovery smoke, and a zero-finding secret scan. The
verified source archive SHA-256 is
`9c9fb953edc6a2c7f7435b15b27e0fe9723e29196ab0abed5de37050381eb397`.
Its 41 MB OCI was pushed and production was rotated to exact digest
`sha256:4538c9c3c4c4bfa2a80989d1e5f0bf23bdc3b5eb1dcca666a025567bc3be7cee`.
Both production machines report that digest and its revision/archive labels,
provider mode remains disabled, and the started machine's health check passes.
Autostart, autostop, and the one-machine minimum remain configured. The public
production suite passed 54/54.

The first sealed post-correction read is as of
`2026-08-12T02:25:39.905203Z`. It reports 706 meaningful search receipts,
20 result selections spanning seven distinct selected-search receipts, zero
action-interest receipts, and zero interest-backed search receipts. Compared
with the prior checkpoint, meaningful searches increased by two, selection and
explicit-interest receipts did not change, and disabled REST attempts increased
by four. Operator release verification can contribute to those search and
unavailable-attempt deltas, so they are not represented as new agents, demand,
leads, or commercial evidence. Stage 1 remains unready: the 14-day observation
window, 20 selected-search receipts, and 10 interest-backed search receipts are
still unmet. The mechanism winner remains empty.

The aggregate receipt is sealed outside the repository as
`NHS_ACTION_INTEREST_CONTRACT_BASELINE_2026-08-12_022540Z_9589816.json`.
Its embedded Stage 1 and attempt-funnel SHA-256 values were independently
recomputed and matched. The disposable 256 MB capture machine was destroyed.
No provider was contacted, invited, activated, or charged; no commercial state
or organic rank changed.
