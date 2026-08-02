# Provider-funded action exchange design review

Date: 2026-08-01; privacy-copy refresh: 2026-08-02

## Surface and conversion contract

- Provider surface category: paid product.
- Privacy surface category: high-volume information.
- Audience: verified service providers, humans operating provider accounts,
  agent developers, and principals evaluating an optional provider-funded
  action.
- Primary provider conversion: claim one indexed domain, keep DNS ownership
  current, and create one accurate draft action with a bounded outcome bounty.
- Primary privacy conversion: understand the stored/excluded data boundary and
  make an informed choice to use the free organic link or the optional,
  authorization-attested provider action.
- Conversion-quality metric: distinct verified provider claims that reach
  commercially activated offers, followed by provider-reported accepted
  handoffs, activations, and a meaningful post-charge replenishment. Page views,
  sign-ins, drafts, synthetic tickets, and signed receipts alone are not
  commercial proof.
- Main provider question: "Can NHS send measurable, authorized demand without
  charging me for traffic or selling rank?"
- Main agent/principal anxiety: hidden ranking influence, unclear compensation,
  identity leakage, stale provider ownership, or an action performed without
  authority.

## Evidence classification

### Measured in this review

- The frozen provider template and refreshed privacy template were rendered and
  inspected at 1440x900 and true 390x844 CSS-pixel viewports. The privacy
  approval is tied to SHA-256
  `6fe9c9dc8f6d797a88481bf3ab129718880c4d9e74c48ba59f87a36dda6e376c`.
- Provider states inspected: signed-out first screen, signed-in setup dashboard,
  verified-claim freshness status, and the claim/draft decision path.
- Privacy states inspected: first screen, the complete query-free receipt card
  containing the exact returned-offer evidence disclosure, and the
  operational/accounting grid with DNS-ownership disclosure.
- The original provider review measured 1425 effective desktop client pixels
  after the browser scrollbar and 375 effective mobile client pixels. The
  refreshed privacy render used a scrollbar-free Chromium shell and measured
  1440/1440/1440/1440 desktop and 390/390/390/390 mobile for inner, client,
  document, and body widths. No horizontal overflow was present.
- Section-specific privacy evidence positions the unchanged rendered DOM at the
  viewport top in a temporary local preview harness. It does not alter the
  template markup, stylesheet, wrapping, card order, or dimensions.
- The returned-offer disclosure is fully readable at both viewports and states
  that NHS retains the paid offer ID, version, name, action type, disclosed
  bounty/currency, charge event, and binding to the organic result for 30 days.
- The mobile provider claim card exposes keep-TXT, last-success, next-check, and
  failure-count state without clipping. The mobile DNS privacy card is 491
  pixels high and was fully visible/readable in one viewport position.
- The signed-out first screen exposes the free-search/neutral-rank promise,
  provider outcome pricing, bounded-pilot label, Merchant-of-Record contract,
  and one primary sign-in action.
- Existing scoped contrast ratios remain 6.07:1 to 15.88:1; new freshness copy
  measured 5.98:1 to 6.62:1. These exceed the WCAG 2.2 AA normal-text target.
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

1. Provider sees the free-search, neutral-rank, and outcome-only commercial
   contract.
2. Human signs in through fail-closed email delivery and claims an
   already-indexed domain.
3. Provider publishes the returned-once DNS TXT proof and keeps it published;
   the dashboard shows last success, next check, and consecutive failures.
4. Provider saves a returned-once claim-scoped callback key and drafts a
   same-domain HTTPS action with principal price, provider bounty, and charge
   event.
5. NHS records real prepaid funding or exact capped terms before activating
   inventory.
6. Agent receives the offer separately after organic search, attests exact v1
   authority, and creates a controlled ticket.
7. Provider reports an idempotent outcome; NHS issues a signed receipt whose
   signature, freshness, and current commercial state remain distinct.

## Release review gate

The provider and privacy surfaces are visually approved only for the exact
template hashes recorded in the JSON contract. Any template change invalidates
the review. Real PostgreSQL 18.4 apply plus immediate replay, receipt-deletion
redaction/CASCADE/SET NULL, accounting, URL, signing-proof, and DNS lifecycle
checks now pass for migration 019 SHA-256
`c516b17ff257f4ec994667c7355eddf559542f900bb172e5a87f4d0be20f444f`.
Production release still requires a dedicated signing-key reference, final
security reread, full exact-candidate checks, owner-authorized deployment
smoke, and owner-authorized pilot activation. Commercial success remains
unproven until the goal's real provider evidence thresholds are met.
