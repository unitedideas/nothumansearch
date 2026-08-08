# Provider mechanism model

This read-only command compares `accepted`, `activated`, and `converted`
provider charge events across the same bounty grid. It does not call NHS,
Stripe, or a provider and cannot change production.

Run the current synthetic scenario:

```sh
go run ./cmd/provider-mechanism-model \
  -scenario docs/experiments/provider-mechanism-synthetic-v1.json
```

The scenario is deliberately marked synthetic. Its funnel counts, latency,
and provider cost ceiling are hypotheses used to test the decision method, not
claims about demand, willingness to pay, traffic, outcomes, or revenue. A
scenario must be a mature cohort and must contain nonzero handoff, activation,
and conversion counts. This prevents incomplete downstream observation from
making an early charge event look artificially better.

The command evaluates every event at every shared bounty point. A package is
viable only when it meets all three declared constraints:

- minimum charged-event sample;
- maximum provider cost per activation; and
- maximum median time to charge.

It selects the viable package with the greatest gross billable amount and uses
time to charge as a tie-breaker. The result always says
`commercial_proof=false`; only the production proof and paid-settlement gates
in `provider-exchange-pilot-runbook.md` can establish commercial proof.

After a separately authorized pilot is closed, the same command can consume the
privacy-safe projection produced by `provider-pilot-status.py`:

```sh
go run ./cmd/provider-mechanism-model \
  -proof-status /path/to/redacted-closed-pilot-proof.json \
  -policy docs/experiments/provider-mechanism-policy-v1.json
```

Proof ingestion fails closed unless the evidence identifies the closed
`developer-tools` pilot, passes the 3/5/2/1 and outcome-integrity gates, contains
no rejected receipt or ledger entry, preserves the free-organic privacy flags,
has one aggregate latency sample for every verified positive ticket, and
contains at least one exact paid terms-settlement receipt. A Checkout, internal
receivable, or unsigned payment claim is insufficient. The
decision policy stays separate because its price grid, acquisition-cost ceiling,
sample threshold, and cash-latency ceiling are provider/business constraints,
not facts NHS should infer from outcome receipts.

The private proof endpoint now reports only the extra aggregates required by the
comparison: verified observed-handoff count, positive-event latency sample
counts, and median seconds from observed handoff to each authenticated outcome.
It also reports paid-settlement count, collected amount by currency, and median
handoff-to-payment time. Settlement proof rejects a payment not bound to a
verified charged receipt, an amount/currency mismatch, a reversed ticket, or a
payment timestamp outside the outcome-to-receipt chronology. It does not expose
ticket IDs, provider IDs, Stripe IDs, queries, intent, or identity data.

The proof projection and model gate explicitly require all of these assertions:
`organic_rank_sold=false`, `raw_queries_sold=false`, `raw_prompts_sold=false`,
`agent_identities_sold=false`, and `principal_identities_sold=false`.

The included sensitivity tests also identify when the provisional activation
choice changes. With the example funnel and price grid, accepted at $25 wins
instead when any of the following is true:

- the provider cost-per-activation ceiling is $65 rather than $100;
- the maximum tolerable time to charge is three rather than 14 days; or
- the minimum charged-event sample is 20 rather than five.

These are decision boundaries, not evidence that either set of constraints is
commercially correct. A real provider must supply the acquisition-cost and
outcome definitions; production observation must supply the event counts and
latencies.

For a real experiment, do not edit synthetic numbers to resemble observations.
Export a new, reviewable scenario from a closed production cohort after the
owner has separately authorized the provider pilot. Preserve the source receipt
references outside this privacy-safe aggregate and never include raw queries,
prompts, agent identifiers, principal identifiers, or provider credentials.
