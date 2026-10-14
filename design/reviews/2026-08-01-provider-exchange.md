# Provider-funded action exchange design review

Date: 2026-08-01; exact handoff/resolver and provider-continuity refresh: 2026-08-02

## Surface and conversion contract

- Provider surface category: service acquisition. Together with the privacy
  reference, the NHS exchange is a service-acquisition/high-volume-information
  hybrid.
- Privacy surface category: high-volume information.
- Audience: verified providers and their operators, principals, agents, and
  reviewers evaluating free discovery, provider ownership, consented handoff,
  bounded routing context, and outcome accounting.
- Primary provider conversion: complete correct provider setup and reach one
  safe, consented handoff through the six-step exchange path.
- Primary privacy conversion: make an informed principal choice among free
  direct provider access, ticket authorization, a separately consented
  handoff, and optional controlled-intent disclosure.
- Conversion-quality metric: successful DNS claims and a valid first handoff
  and controlled-intent resolution without support, privacy, authorization, or
  security errors. Page views, sign-ins, drafts, synthetic tickets, handoff
  receipts, and signed outcome receipts alone are not commercial proof.
- Main provider question: "Can NHS send measurable, authorized demand without
  charging me for traffic or selling rank?"
- Main agent/principal anxiety: hidden ranking influence, unclear compensation,
  identity leakage, stale provider ownership, a caller attestation being
  overstated as verified principal consent, optional routing data becoming a
  hidden lead sale, or a provider action performed without authority or
  without NHS observing the handoff.

## Evidence classification

### Measured in this review

- The exact provider and privacy templates were rendered and inspected at
  1440x900 and true 390x844 CSS-pixel viewports. Approval is tied to provider
  SHA-256 `8ca054ad176d0902b7342d4314c0ee197fd77e0ebd6e84aba1399b92b017b3ea`
  and privacy SHA-256
  `0ac00c147019b4b35410b87e25434556f077072c7abcfa7106f4885d87ad25ff`.
  The combined working-tree review-scope digest is
  `f3fd4c703e1174c69ca902547fdef705c5b1c040f2f57976d45e3cb8f2c8281d`.
- Provider states inspected: signed-in first screen; the complete six-step
  decision path on desktop and its ordered mobile reflow; signed-in
  empty-account claim and terms-only draft-action setup; and the agent-facing
  machine contract. Current full-page heights measure 5208px desktop and
  8442px mobile for the signed-in state, and 4200px desktop and 6555px mobile
  signed out. The empty-account layout received only a local empty claims
  response. Section captures use local visibility isolation for unrelated
  siblings without changing the reviewed section markup or styles.
- Privacy states inspected: first screen; complete 8301px desktop and 15151px
  mobile pages; physical-retention wording; exact handoff consent; and optional
  controlled-intent consent, resolver allowlist, excluded fields, replay
  boundary, retention, and accounting copy.
- Both refreshed renders measured 1440/1440/1440/1440 desktop and
  390/390/390/390 mobile for inner, client, document, and body widths. No
  horizontal overflow was present.
- Section-specific evidence uses the bundled Chromium renderer and locally
  hides unrelated sibling sections. It does not alter the reviewed template
  source, target-section markup, target styles, wrapping, card order, or
  dimensions.
- The returned-offer disclosure is fully readable at both viewports and states
  that NHS retains the paid offer ID, version, name, action type, disclosed
  bounty/currency, charge event, and binding to the organic result during the
  receipt's 30-day eligibility window.
- The action-interest disclosure is fully readable at both viewports and says
  exactly what the receipt stores, that the recording invocation creates no
  persisted IP/user-agent or generic request telemetry, that earlier-search
  abuse telemetry remains separate, and that no row is shared with or used to
  contact a provider.
- The physical-retention disclosure distinguishes logical eligibility from
  deletion: action-interest replay/reporting and controlled-ticket resolver
  availability stop at the 30-day boundary, while physical deletion or
  redaction runs on the next successful boot or hourly cleanup. It states that
  downtime or cleanup failure can delay physical removal without extending
  replay, reporting, or resolver availability. The retention label measures
  15.04:1 contrast.
- The handoff disclosure is fully readable at both viewports and makes ticket
  preparation and URL release separate steps. It exposes the exact
  `nhs-provider-handoff-consent-v1` attestation, explains that no query,
  identity, contact, referrer, user-agent, or network data enters the receipt,
  and distinguishes a receipt from a provider-reported outcome or charge.
- The optional controlled-intent disclosure is fully readable at both
  viewports. It names the exact five-field routing bundle, provider-claim and
  bearer gates, shorter ticket/30-day availability limit, resolver-only REST
  surface, excluded identity/query/contact/payment fields, and the fact that
  declining disclosure blocks neither handoff nor free direct provider access.
- The mobile provider claim and DNS-responsibility cards remain readable
  without clipping; ongoing TXT, freshness, and failure-state obligations are
  visible before action activation. The claim card now explains that the
  returned-once key reads claim-scoped continuity and privacy-thresholded
  demand, and that NHS never stores the raw key.
- The first screen exposes the free-search/neutral-rank promise, exact capped
  CPA terms-only pilot, provider outcome pricing, bounded-pilot label,
  Merchant-of-Record contract, and one primary setup action. The six-step path remains balanced as two
  rows of three cards on desktop and one ordered column on mobile: verify,
  declare, prepare, hand off, resolve, record.
- Signed-in claim and draft forms retain labels, explanatory copy, correct
  reading order, and 44px-or-taller visible controls at desktop and mobile.
- The machine-contract section now exposes the claim-key-authenticated
  `pilot-status` and privacy-thresholded `demand` reads. Demand derives its
  domain from the key rather than a caller parameter; neither read changes
  organic ranking, charges either party, or discloses raw queries, credentials,
  attribution material, identities, contacts, or action URLs.
- Scoped contrast ratios are 6.07:1 to 16.02:1, including 10.60:1 for the exact
  controlled-intent consent text and 8.16:1 for its version label. These exceed
  the WCAG 2.2 AA normal-text target.
- Each surface has one H1, one main landmark, `lang="en"`, no empty accessible
  link names, no unlabeled visible fields, and no broken `aria-labelledby`
  references. Both pages reflow at 320 CSS pixels without horizontal overflow;
  primary actions, provider fields, and the privacy contact route show a 3px
  visible focus outline. The browser error log was empty for the reviewed
  flows.

These are source and visual measurements. They do not prove provider willingness
to pay, NHS-observed handoffs, provider-accepted outcomes, activations,
collection, or renewal.

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
- Ticket preparation never releases the provider action URL. The exact bearer
  token and a separate v1 handoff attestation must create an immutable NHS
  receipt before a ticket can enter any positive state.
- Controlled-intent disclosure is a third, optional consent. The same active
  DNS-verified claim can resolve only the ticket's bounded topic, region,
  budget band, urgency, and allowlisted requirements during the shorter valid
  window. Resolution is free, read-only, and cannot create a charge, outcome,
  proof, or durable identity telemetry; declining it does not block handoff.

### Observed category patterns

- Service-acquisition onboarding is more credible when value, fee trigger,
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
- Exact hard-capped CPA activation will attract fewer but more credible
  providers than self-serve unlimited terms. Prepaid collection is deliberately
  absent from the launch pilot.
- Agents will use a separate paid-offer object when it contains a relevant
  action, disclosed principal price, NHS compensation, and a canonical free
  alternative.
- Requiring a second, explicit handoff call will reduce accidental provider
  visits enough to justify the extra agent step while preserving useful
  downstream conversion.
- Principals will opt into a five-field routing bundle often enough to improve
  provider handling without requiring NHS to sell raw queries, identities, or
  organic placement. This remains a conversion hypothesis until live pilot
  behavior is measured.

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
   event. The same key can recover bounded acceptance/handoff continuity and
   privacy-thresholded aggregate demand without selecting another domain.
7. NHS joins the provider-authenticated exact capped terms to owner-held
   evidence before activating inventory; prepaid collection is not launched.
8. Agent receives the offer separately after organic search, attests exact v1
   ticket authority, and prepares a controlled ticket without receiving the
   provider action URL.
9. The exact ticket bearer separately attests
   `nhs-provider-handoff-consent-v1`; NHS records one immutable, query-free
   handoff receipt and only then releases the action URL.
10. If the principal separately opted in at handoff, the active key for that
    same fresh DNS claim can resolve only the bounded controlled-intent bundle.
    Declining disclosure leaves the handoff and free direct access intact.
11. Provider reports an idempotent outcome; NHS issues a signed receipt whose
   signature, freshness, and current commercial state remain distinct.

## Release review gate

The provider and privacy surfaces are visually approved only for the exact
template hashes and evidence hashes recorded in the JSON contract. Any covered
template change invalidates the review. This design approval does not establish
database migration, security, deployment, or commercial proof. Production
release still requires the separate exact-candidate gates, a dedicated
signing-key reference, owner-authorized deployment smoke, and owner-authorized
pilot activation. Commercial success remains unproven until the goal's real
provider evidence thresholds are met.
