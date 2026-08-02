# Provider-funded action exchange design review

Date: 2026-08-01; privacy-copy refresh: 2026-08-02

## Surface and conversion contract

- Provider surface category: paid product.
- Privacy surface category: high-volume information.
- Audience: verified service providers, humans operating provider accounts,
  agent developers, and principals evaluating free discovery, a noncontacting
  action-interest receipt, or an optional provider-funded action.
- Primary provider conversion: claim one indexed domain, keep DNS ownership
  current, and create one accurate draft action with a bounded outcome bounty.
- Primary privacy conversion: understand the stored/excluded data boundary and
  make an informed choice among the free organic link, a caller-attested but
  noncontacting Stage 1 action-interest receipt, and the separately
  authorization-attested provider action.
- Conversion-quality metric: distinct verified provider claims that reach
  commercially activated offers, followed by provider-reported accepted
  handoffs, activations, and a meaningful post-charge replenishment. Page views,
  sign-ins, drafts, synthetic tickets, and signed receipts alone are not
  commercial proof.
- Main provider question: "Can NHS send measurable, authorized demand without
  charging me for traffic or selling rank?"
- Main agent/principal anxiety: hidden ranking influence, unclear compensation,
  identity leakage, stale provider ownership, an anonymous caller attestation
  being overstated as verified principal consent, or a provider action
  performed without authority.

## Evidence classification

### Measured in this review

- The frozen provider template and refreshed privacy template were rendered and
  inspected at 1440x900 and true 390x844 CSS-pixel viewports. The privacy
  approval is tied to SHA-256
  `a6349757e7819b32fa9d79ae27ea24ffccb13c0da5de3b47f8db0f5aee51acbe`.
- Provider states inspected: signed-out first screen, signed-in setup dashboard,
  verified-claim freshness status, and the claim/draft decision path.
- Privacy states inspected: first screen; complete 5023px desktop and 8442px
  mobile pages; the query-free receipt card; the noncontacting
  action-interest boundary; its exact attestation; the returned-offer evidence
  disclosure; and the operational/accounting grid with DNS-ownership
  disclosure.
- The original provider review measured 1425 effective desktop client pixels
  after the browser scrollbar and 375 effective mobile client pixels. The
  refreshed privacy render used a scrollbar-free Chromium shell and measured
  1440/1440/1440/1440 desktop and 390/390/390/390 mobile for inner, client,
  document, and body widths. No horizontal overflow was present.
- Section-specific privacy evidence scrolls the unchanged rendered DOM to the
  viewport top through the Chrome DevTools Protocol. It does not alter the
  template markup, stylesheet, wrapping, card order, or dimensions.
- The returned-offer disclosure is fully readable at both viewports and states
  that NHS retains the paid offer ID, version, name, action type, disclosed
  bounty/currency, charge event, and binding to the organic result for 30 days.
- The action-interest disclosure is fully readable at both viewports and says
  exactly what the receipt stores, that the recording invocation creates no
  persisted IP/user-agent or generic request telemetry, that earlier-search
  abuse telemetry remains separate, and that no row is shared with or used to
  contact a provider.
- The mobile provider claim card exposes keep-TXT, last-success, next-check, and
  failure-count state without clipping. The mobile DNS privacy card is 491
  pixels high and was fully visible/readable in one viewport position.
- The signed-out first screen exposes the free-search/neutral-rank promise,
  provider outcome pricing, bounded-pilot label, Merchant-of-Record contract,
  and one primary sign-in action.
- Scoped contrast ratios are 6.07:1 to 16.02:1, including 10.60:1 for the exact
  action-interest consent text and 8.16:1 for its version label. These exceed
  the WCAG 2.2 AA normal-text target.
- The browser error log was empty for the reviewed flows.

These are source and visual measurements. They do not prove provider willingness
to pay, accepted handoffs, activations, collection, or renewal.

### Standards and durable principles

- WCAG 2.2 AA is the accessibility target; labeled fields, semantic forms,
  visible focus, status live regions, and 44-pixel-or-larger primary controls
  are required.
- Organic results, score, order, search, canonical provider links, and direct
  provider access remain free and cannot be purchased.
- Provider-funded actions are separate disclosed objects attached only to exact
  already-returned organic sites after a committed query-free search receipt.
- A Stage 1 action-interest receipt is an anonymous caller attestation, not NHS
  verification of the principal, identity, agency, authority, uniqueness, or
  completion. It is bound to one exact fresh non-synthetic organic result,
  cannot contact a provider, and cannot become commercial proof.
- The caller attests authorization; NHS does not claim to verify identity,
  agency, authority, Merchant-of-Record status, or the underlying
  provider-reported business event.
- Provider ownership is renewable proof: the TXT record must stay published;
  NHS stores its challenge-token hash, rechecks every 24 hours, and stops paid
  actions after three failed checks or seven days without success.
- The provider action ticket accepts controlled fields only and redacts
  controlled intent after 30 days.

### Observed category patterns

- Paid-product onboarding converts better when value, fee trigger,
  qualification gate, ongoing responsibilities, and owner controls are legible
  before account creation.
- Trust-sensitive information surfaces work best when stored/excluded fields,
  retention, party controls, and exact attestation wording are inspectable
  rather than buried in generic policy prose.
- Machine-facing offers require deterministic fields, explicit compensation,
  idempotency, renewable ownership proof, and separate current-state
  verification more than decorative marketing claims.

### Hypotheses to test

- Providers will accept an outcome bounty more readily than click or impression
  pricing because organic access remains free and invalid/duplicate charged
  events can be credited.
- Exact prepaid or hard-capped CPA activation will attract fewer but more
  credible providers than self-serve unlimited terms.
- Agents will use a separate paid-offer object when it contains a relevant
  action, disclosed principal price, NHS compensation, and a canonical free
  alternative.

## Decision path

1. Agent uses the canonical free organic link or explicitly records one
   controlled, caller-attested next step without contacting a provider.
2. NHS exposes only owner-only Stage 1 aggregate counts; `stage1_ready` requires
   100 meaningful search receipts, selections on 20 distinct search receipts,
   action interest on 10 distinct search receipts, and a 14-day observation
   span. These are still receipts, not unique agents or commercial proof.
3. Provider sees the free-search, neutral-rank, and outcome-only commercial
   contract.
4. Human signs in through fail-closed email delivery and claims an
   already-indexed domain.
5. Provider publishes the returned-once DNS TXT proof and keeps it published;
   the dashboard shows last success, next check, and consecutive failures.
6. Provider saves a returned-once claim-scoped callback key and drafts a
   same-domain HTTPS action with principal price, provider bounty, and charge
   event.
7. NHS records real prepaid funding or exact capped terms before activating
   inventory.
8. Agent receives the offer separately after organic search, attests exact v1
   authority, and creates a controlled ticket.
9. Provider reports an idempotent outcome; NHS issues a signed receipt whose
   signature, freshness, and current commercial state remain distinct.

## Release review gate

The provider and privacy surfaces are visually approved only for the exact
template hashes recorded in the JSON contract. Any template change invalidates
the review. Real PostgreSQL 18.4 apply plus immediate replay, receipt-deletion
redaction/CASCADE/SET NULL, accounting, URL, signing-proof, and DNS lifecycle
checks now pass for migration 019 SHA-256
`c516b17ff257f4ec994667c7355eddf559542f900bb172e5a87f4d0be20f444f`
and migration 020 SHA-256
`67fd580216b969e062d211e2df706a514d9823155840955df45a5c3e053cc0a2`.
Production release still requires a dedicated signing-key reference, final
security reread, full exact-candidate checks, owner-authorized deployment
smoke, and owner-authorized pilot activation. Commercial success remains
unproven until the goal's real provider evidence thresholds are met.
