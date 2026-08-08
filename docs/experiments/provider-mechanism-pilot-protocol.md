# Provider mechanism pilot protocol

**Status:** internal, no-send, provider exchange disabled
**Scope:** Not Human Search `developer-tools` only

## Purpose

Test which provider-funded event creates defensible NHS revenue while free REST
and MCP discovery, organic inclusion, and organic rank remain unchanged.

The first real cohort should charge one event only: `activated`. The provider
callback may still record the ticket's normal `accepted`, `activated`, and
`converted` state transitions. Those later authenticated observations permit a
counterfactual comparison of all three mechanisms without charging a ticket
more than once or changing accepted terms after activation.

The provisional $75 activation package in the synthetic model is not an offer.
Each provider must supply and accept its exact event definition, bounty, hard
cap, cost constraint, duplicate/invalid rule, callback SLA, principal price,
and Merchant-of-Record acknowledgement before any real inventory is activated.

## Cohort controls

- One demand topic: `developer-tools`.
- Three externally deduplicated, DNS-verified provider companies.
- One charge event across the cohort: `activated`.
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

## Counterfactual comparison

After the cohort is mature, run the mechanism model with the observed handoff,
accepted, activated, and converted counts and median event latencies. Evaluate
all three events across the same bounty grid. For each package compare:

Use the reviewed `provider-pilot-status.py --scope proof` projection as the
count and latency source; do not transcribe database rows or reconstruct a
scenario from private ticket-level data.

The comparison may label collected-revenue evidence only when that projection
contains an integrity-valid exact paid terms-settlement receipt and its
privacy-safe currency total. A receivable, settlement order, Checkout session,
redirect, or provider assertion remains unpaid.

- charged-event sample size;
- gross billable value and actually paid value;
- revenue per observed handoff;
- provider cost per activation and conversion;
- invalid, duplicate, credit, and disputed-event rate; and
- median and tail time to authenticated outcome and paid settlement.

Accepted-event revenue is a counterfactual unless the provider's immutable
terms actually charge `accepted`; converted-event revenue is likewise a
counterfactual in an activation cohort. Label both as modeled and never append
ledger entries or settlement orders for them.

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
5. Activation produces the greatest paid value among packages that satisfy
   provider cost, sample, dispute/credit, and time-to-cash constraints.

If accepted wins, draft a new accepted-event cohort with new immutable terms;
do not reinterpret old tickets. If converted wins, wait for a complete
conversion window before drafting new terms. If no mechanism satisfies the
constraints, keep discovery free and stop the commercial experiment rather
than weakening privacy, ranking neutrality, consent, or proof gates.

## Authorization boundary

This protocol does not authorize outreach, provider enrollment, terms
acceptance, offer activation, checkout creation, payment requests, or public
claims. Production must remain in disabled mode until Shane separately
authorizes the bounded pilot.
