# Provider-funded action pilot: owner brief

**Status:** internal, no-send, no-contract brief
**Prepared:** 2026-08-03
**Scope:** Not Human Search only; it does not cover AI Dev Board.

## What is being sold

Not Human Search keeps its public REST and MCP discovery, result membership,
and organic ordering free and neutral. A provider can separately fund one
clearly disclosed, optional action **after** its site was returned
organically. The searching agent and principal never receive a payment
challenge.

The provider pays NHS only when its authenticated callback records the exact
pre-agreed `accepted`, `activated`, or `converted` outcome. A provider
remains Merchant of Record. This is a capped, post-action CPA pilot—not a
prepaid lead bundle, ranking purchase, or sale of candidate/agent data.

## The value exchange

| Provider receives | NHS receives | Never exchanged |
| --- | --- | --- |
| An explicitly consented, disclosed handoff from a neutral organic result; an exact action/outcome receipt; aggregate demand reporting only once privacy thresholds are met | The exact CPA attached to an independently recorded, qualifying provider outcome | Raw queries or prompts; per-agent histories; alleged identity/contact data; user-agent/IP/referrer data; organic-rank influence; payment from the searching agent |

Ticket creation and observed handoff are free. An optional, separately
consented controlled-intent resolver exposes only a fixed, redacted field set
for that exact ticket. Declining it cannot block free direct provider access
or the handoff.

## Who is a legitimate first pilot provider

Choose a provider only when all of the following are true:

1. Its currently indexed domain has a real same-domain action surface (for
   example a trial, demo, quote, signup, booking, or purchase), which can be
   represented without credentials or provider query parameters.
2. It can prove DNS control, retain a claim-scoped callback key, and send an
   idempotent server-to-server outcome callback.
3. Its operator can accept exact, hard-capped CPA terms and acknowledge that
   it remains Merchant of Record.
4. The action has a clear, auditable provider-side qualification event and a
   real credit/duplicate rule.
5. The owner has reviewed a privacy-thresholded demand report. Lack of a
   qualifying cohort is a reason to wait, not a reason to make a demand claim.

The provider is not buying placement or a promise of traffic. It is buying a
measurable, voluntary action path and only pays after the agreed outcome.

## Current evidence and non-claims

On 2026-08-03, the 30-day privacy-safe aggregate for `stripe.com` contained
two search receipts, two organic results returned, and zero selections. The
topic threshold is 20. This is below threshold and therefore **not** evidence
of provider demand, buyer intent, a qualified lead, conversion, or revenue.

Do not use this brief to claim that a named provider has demand, that NHS can
increase conversion, or that a provider has agreed to a pilot. No provider
agreement, provider payment, or production provider exchange is in place when
this brief was prepared.

## Exact pilot proposal, to complete only with the provider

The following fields must be agreed and stored in the exact commercial terms
before an action can become active:

| Field | Provider-supplied value |
| --- | --- |
| Indexed domain and verified claim | _Not yet selected_ |
| Same-domain action URL and action type | _Not yet selected_ |
| Qualifying event (`accepted`, `activated`, or `converted`) | _Not yet selected_ |
| CPA amount, currency, hard cap, and term period | _Not yet selected_ |
| Principal price and what the principal receives | _Not yet selected_ |
| Duplicate/invalid credit rule | _Not yet selected_ |
| Callback SLA and idempotency event identifier | _Not yet selected_ |
| Merchant-of-Record acknowledgement | _Not yet accepted_ |

NHS must not fill these fields with inferred provider data, historical demand,
or a sales estimate. A changed offer or terms hash requires a new acceptance.

## Required evidence before a charge can be collected

1. The release candidate passes disposable PostgreSQL migration and replay
   regressions, then completes the owner-authorized protected cutover.
2. The provider completes DNS verification and authenticates the exact company
   and terms acceptance.
3. The owner verifies the acceptance and activates the exact offer only after
   all implementation and commercial gates are satisfied.
4. A free organic result leads to the principal's explicit ticket and separate
   handoff consents. NHS writes the immutable handoff receipt before returning
   the attributed URL.
5. The provider sends an authenticated, idempotent qualifying outcome callback
   bound to that exact ticket and terms snapshot.
6. Only then may the owner create the post-action Stripe Checkout. Its amount,
   currency, provider claim, offer, ticket, outcome, and terms hash are frozen
   by the settlement order.
7. Only a Stripe-signature-verified paid `checkout.session.completed` event
   appends a paid receipt. A created Checkout, an invoice, an internal charge,
   or a provider assertion is not revenue proof.

## Owner decision points

Before any external outreach, contract acceptance, production migration,
provider activation, checkout creation, payment request, or public statement,
confirm the exact owner authority required by
[`provider-exchange-pilot-runbook.md`](provider-exchange-pilot-runbook.md).
Until then this document is preparation only; it authorizes no external action.
