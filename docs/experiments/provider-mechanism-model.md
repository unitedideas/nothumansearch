# Provider mechanism model

This read-only command tests the selection logic on a synthetic bounty grid and
selects among `accepted`, `activated`, and `converted` only when a closed real
pilot supplies separate paid evidence for all three mechanisms. It does not
call NHS, Stripe, or a provider and cannot change production.

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
  -policy docs/experiments/provider-mechanism-policy-v4.json
```

Proof ingestion fails closed unless the evidence identifies the closed
`developer-tools` pilot, passes the 3/5/2/1 and outcome-integrity gates, contains
no rejected receipt or ledger entry, preserves the free-organic privacy flags,
has one aggregate latency sample for every verified positive ticket, and
contains separate exact paid terms-settlement evidence for accepted-handoff,
activated-CPA, and converted-CPA tickets. Each arm must also independently meet
the policy's minimum distinct-provider coverage and charged-event sample. Only
aggregate provider counts enter the projection; provider IDs remain excluded.
The final selector starts with verified paid revenue for each exact
non-synthetic provider offer returned to an agent, then uses the exact fee and
net from the matching Stripe balance transaction only after Stripe reports that
net as available. Published processor rates never enter winner selection. They
belong only to the pre-pilot price-floor hypothesis because an estimate cannot
replace retained-value evidence.

An arm must clear both the declared processing-net margin floor and the minimum
processing-net cents per 1,000 returned offers. The selector then chooses the
highest processing-net revenue per 1,000 returned offers and uses
paid-settlement latency as its tie-breaker. This is actual available retained
value after the observed processor fee; it is not full profit because Fly,
support, fraud, tax, and owner-labor costs are not subtracted in the commercial
proof. Revenue per observed handoff remains a downstream diagnostic but cannot
select the winner because it ignores agents that saw an offer and did not
proceed. Gross revenue per returned offer also remains diagnostic and cannot
select a winner whose transaction pattern retains less value.

Point estimates can be real and still be too close to support a decision. The
verified policy therefore requires the top viable arm to exceed the runner-up
by both an absolute processor-net value per 1,000 returns and a relative lead.
Policy v4 sets those floors to 1,146 cents, the current 31-day NHS
infrastructure baseline, and 20%. Both must pass. The report exposes the
required and observed leads; a near tie leaves selection empty.

Policy v4 also preregisters a 95% confidence level and a 20,000-cent maximum
processor net per returned offer. The private proof exposes only each arm's
maximum offered bounty, exact available-net sum, net sum of squares, and maximum
net settlement. Unsettled returned offers are zero observations. Those
aggregates are sufficient to apply Maurer and Pontil's empirical-Bernstein
bound without exporting ticket, provider, query, prompt, or identity data.

The selector applies a union bound across both sides of all three mechanism
intervals. A point-estimate leader remains unselected unless its lower
processor-net confidence bound exceeds the runner-up's upper bound. The
predeclared maximum must cover every returned offer's immutable bounty, so an
observed maximum cannot quietly become a data-dependent support assumption.
The proof also binds `nhs-provider-mechanism-arm-v1` and labels the observation
unit `returned_offer_opportunity_not_unique_agent`. The interval therefore
describes randomized returned-offer opportunities; it is never reported as a
count of unique agents or people.
The research basis is Maurer and Pontil, “Empirical Bernstein Bounds and Sample
Variance Penalization,” Theorem 4: https://arxiv.org/abs/0907.3740.

The initial 3/5/2/1 milestone proves the revenue rails but cannot by itself name
a strongest mechanism. Aggregate payment from one charge event cannot qualify
another. A Checkout, internal receivable, or unsigned payment claim is
insufficient. The decision policy stays separate because its acquisition-cost
ceiling, minimum paid evidence, reversal-rate ceiling, cash-latency ceiling,
and retained-value floors are business constraints, not facts NHS should infer
from outcome receipts. Processor fee and available net are evidence, not
policy.

The private proof endpoint now reports only the extra aggregates required by the
comparison: verified observed-handoff count, positive-event latency sample
counts, and median seconds from observed handoff to each authenticated outcome.
It also reports each mechanism's own handoff funnel, paid-settlement count,
collected amount, and median handoff-to-payment time, plus reconciled pilot
totals. Settlement proof rejects a payment not bound to a
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
Use only the reviewed proof projection from a closed production pilot after the
owner has separately authorized it. Preserve source receipt references outside
this privacy-safe aggregate and never include raw queries, prompts, agent
identifiers, principal identifiers, or provider credentials.
