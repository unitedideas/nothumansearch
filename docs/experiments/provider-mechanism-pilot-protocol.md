# Provider mechanism pilot protocol

**Status:** internal, no-send, provider exchange disabled
**Scope:** Not Human Search `developer-tools` only

## Purpose

Test which provider-funded event creates defensible NHS revenue while free REST
and MCP discovery, organic inclusion, and organic rank remain unchanged.

The real pilot must contain three separately settled mechanism arms:
`accepted`, `activated`, and `converted`. Every ticket is immutably assigned to
one provider offer and therefore one charge event before handoff. The provider
callback still records the ticket's normal state transitions, but a ticket can
produce at most one charge and one settlement. Aggregate payment from one arm
must never be reused as proof that another mechanism reached the billing rail.

The provisional $75 activation package in the synthetic model is not an offer.
Each provider must supply and accept its exact event definition, bounty, hard
cap, cost constraint, duplicate/invalid rule, callback SLA, principal price,
and Merchant-of-Record acknowledgement before any real inventory is activated.
Before proposing those bounties, run the no-write price-floor model against the
current NHS-only cost baseline. Its published-fee allowance establishes an
infrastructure contribution floor, not profit or commercial proof. Final
selection continues to use actual available processor-net settlement receipts,
never the modeled allowance.

## Cohort controls

- One demand topic: `developer-tools`.
- Three externally deduplicated, DNS-verified provider companies.
- Every one of those three companies must produce at least one authenticated
  charged event in every mechanism arm before the comparison can select a
  winner. The proof exposes only the distinct-company count per arm, never
  company IDs.
- Three exact charge-event arms across the cohort: `accepted`, `activated`,
  and `converted`; every participating provider must accept the immutable terms
  for each arm it receives before that inventory is activated.
- For one provider and action type, the three arms must be otherwise equivalent:
  same provider, destination, action, principal price, disclosure, and response
  and credit rules. A difference suppresses the whole group rather than
  creating a confounded comparison.
- The random search receipt assigns exactly one of the three arms before the
  provider-funded sidecar is returned. The other two are neither disclosed nor
  persisted as returned offers and therefore cannot mint a ticket. Callers and
  providers cannot choose the accounting event for an individual opportunity.
- No paid placement, ranking change, discovery paywall, or identity/query sale.
- Each ticket has explicit authority and separate handoff consent.
- A ticket is assigned to only one provider offer and can produce at most one
  qualifying charge under its immutable terms snapshot.
- Provider-specific hard caps bound both ticket count and dollar exposure.
- The cohort remains open through the declared downstream observation window;
  immature tickets cannot enter the mechanism comparison.
- Prepaid balances remain unavailable. A receivable becomes collected revenue
  only after the exact signed paid-settlement receipt exists.

## Measurements

For every provider and the deduplicated cohort, record only privacy-safe IDs and
aggregates already permitted by the provider-exchange contract:

| Measure | Required interpretation |
| --- | --- |
| Organic result and offer impression | Opportunity, never a lead or revenue claim |
| Authorized ticket | Principal-authorized intent, not provider contact |
| Observed handoff | NHS-observed contact path, still free |
| Accepted outcome | Authenticated provider assertion after a matching handoff |
| Activated outcome | Authenticated provider assertion under its exact accepted definition |
| Converted outcome | Authenticated provider assertion under its exact accepted definition |
| Credit/rejection/duplicate/invalid | Reverses or excludes affected commercial state as the contract specifies |
| Settlement order | Frozen receivable, not collected revenue |
| Signed paid settlement | Collected revenue evidence |
| Event latency | Time from observed handoff to each authenticated outcome and to payment |

Never export raw queries, prompts, controlled intent, agent identity, principal
identity, IP address, user agent, referrer, callback keys, attribution tokens,
or signing material into experiment analysis.

The proof-to-model projection must carry explicit false assertions for sold
organic rank, raw queries, raw prompts, agent identities, and principal
identities. Missing assertions fail closed; absence is not inferred as false.

## Paid mechanism comparison

After all three arms are mature, run the mechanism model with each arm's own
observed handoffs, accepted, activated, and converted outcomes, paid settlement
count and amount, and median handoff-to-payment latency. The selector fails
closed unless every mechanism has at least one exact signed paid settlement.
For each mechanism compare:

Use the reviewed `provider-pilot-status.py --scope proof` projection as the
count and latency source; do not transcribe database rows or reconstruct a
scenario from private ticket-level data.

The comparison may label collected-revenue evidence only when that projection
contains an integrity-valid exact paid terms-settlement receipt and its
privacy-safe currency total. A receivable, settlement order, Checkout session,
redirect, or provider assertion remains unpaid.

- charged-event and paid-settlement sample size, applying the declared minimum
  separately to each mechanism arm rather than to the pooled cohort;
- distinct verified-provider coverage per arm, requiring the full three-company
  cohort so provider mix cannot masquerade as a mechanism effect;
- exact non-synthetic provider-offer returns per arm;
- processing-net value per 1,000 returned provider offers after the declared
  percentage-plus-fixed payment-processing allowance, with gross revenue per
  returned offer and revenue per observed handoff retained only as diagnostics;
- provider cost per activation and conversion;
- invalid, duplicate, credit, and disputed-event rate; and
- median and tail time to authenticated outcome and paid settlement.

Sensitivity grids remain useful for pricing hypotheses, but modeled gross
billables cannot select the winning mechanism. Final selection uses only exact
paid receipts from tickets whose immutable terms actually charge that event,
subtracts the policy's conservative payment-processing allowance, and divides
the retained amount by the exact eligible offers returned to agents in that
arm. The policy must declare the processing basis points, fixed per-settlement
fee, minimum processing-net margin, and minimum processing-net cents per 1,000
returns. These are policy allowances rather than observed balance-transaction
fees, and the result must not be called full profit.
The initial 3-provider, 5-accepted-handoff, 2-activation, 1-renewal milestone
proves that the consent, attribution, outcome, billing, renewal, and settlement
rails can produce real revenue. It does not by itself select a strongest
mechanism. Selection remains empty until every candidate arm independently
meets `min_charged_provider_companies_per_mechanism`, `min_charged_events`, and
`min_offer_returns_per_mechanism`, plus the paid-settlement minimum.

## Decision rule

Keep `activated` for the next cohort only when all of the following hold:

1. The verified pilot gate reaches 3 providers, 5 accepted handoffs,
   2 activations, and 1 genuine terms renewal.
2. Every revenue claim has an exact signed paid-settlement receipt.
3. Provider-specific activation cost remains within each provider's accepted
   ceiling after credits and invalid/duplicate outcomes.
4. The activation sample and observation window are large enough that
   `accepted` does not win merely from earlier reporting and `converted` does
   not lose merely from incomplete observation.
5. Activation produces the greatest processing-net revenue per 1,000 returned
   offers among mechanisms that satisfy provider cost, sample, dispute/credit,
   time-to-cash, processing-net margin, and processing-net revenue-per-return
   constraints.

If accepted or converted wins, use that event only in a new immutable offer
version; do not reinterpret old tickets. If no mechanism satisfies the
constraints, keep discovery free and stop the commercial experiment rather
than weakening privacy, ranking neutrality, consent, or proof gates.

## Authorization boundary

This protocol does not authorize outreach, provider enrollment, terms
acceptance, offer activation, checkout creation, payment requests, or public
claims. Production must remain in disabled mode until Shane separately
authorizes the bounded pilot.
