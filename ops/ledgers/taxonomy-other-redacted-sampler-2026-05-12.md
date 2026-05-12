# Taxonomy Other Redacted Sampler

Observed: 2026-05-12T02:47:11Z

Action: added a committed, tested aggregate-only sampler for `category=other`
taxonomy review. The sampler can inspect live row-level search results in
memory, but emits only bucket counts and explicit redaction-policy flags. It is
intended for business-agent executor review before adding narrow category rules,
without leaking candidate domains, URLs, row IDs, names, or descriptions into
planner artifacts.

Proof:

- `python3 -m unittest tools/test-taxonomy-other-redacted-sample.py tools/test-discovery-quarantine-report.py tools/quality-gate-discovery-test.py`
- Anonymous live sample currently returns NHS quota `402`, which confirms the
  sampler does not bypass the paid API boundary. The new Keychain-backed option
  constructs Bearer auth without printing the secret; admin keys are not REST API
  usage keys, so the live taxonomy sample remains gated until a proper NHS API
  key exists.

No public action was taken. No external post, email, directory submission, or
customer-visible message was sent.
