# WorkItem closeout: work_machine_2355ca57b79a710b

Completed aggregate-only closeout for the 2026-05-21 full recrawl.

Local proof:

- `ops/ledgers/full-recrawl-closeout-2026-05-21.md`
- `harness/discovery-quality-latest.json`
- `harness/discovery-quarantine-latest.json`
- `harness/discovery-quarantine-history.jsonl`

Boundary result:

- seed refresh complete: `2026-05-21 05:42:37`, `success=469`, `failed=14`, `total=483`
- full recrawl complete: `2026-05-21 10:25:48`, `success=9847`, `failed=389`, `total=10236`
- closeout lock state: `tools/full-recrawl.lock` absent, `tools/recrawl.lock` absent

Discovery-quality decision:

- current `category=other` low-signal state is `no_op_fixed_point`
- taxonomy-rule change: `false`
- threshold adjustment: `false`
- low-signal cohorts remain audit-only with `public_search=false` and `score_fix_targeting=false`

No raw domains, URLs, row ids, descriptions, emails, tokens, private notes, crawler row output, broad crawl, or full recrawl were used for local proof.
