---
id: t-01KZP8HAMD22FJRA35NZBRK614
kind: task
created: 2026-08-10T16:36:47Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 2, probable: 4, pessimistic: 8}"
---
# Falsify the safety-property suites: name a surviving mutation or report none exists
## Acceptance
- [x] one named test file audited assertion by assertion, with what each ACTUALLY asserts recorded
- [x] for each weak test: the one-line code mutation that keeps it green, cited at file:line
- [x] the unprotected behaviour and the concrete user-visible failure it would cause are stated
- [x] if no mutation survives, that is filed as the finding rather than inventing work
## Log
- 2026-08-10T16:37:29Z claimed by a-mutation-auditor-dgweq2
- 2026-08-10T16:42:06Z finding by a-mutation-auditor-dgweq2: accept's unlanded-branch refusal is wired but wholly untested: a one-line mutation closes unlanded work green under --require-verify (event 01KZP8QVGG1ZZ8WQVXM667NFCT)
- 2026-08-10T16:47:55Z accepted by a-root (applied 1 proposal(s))
- 2026-08-10T16:47:55Z closed WITHOUT verification — no --verify command was given
- 2026-08-10T16:47:55Z deliverable: no dacli/324-falsify-the-safety-property-suites-name-a-surviving-mutation-or-report-none branch — nothing to check against trunk
- 2026-08-10T16:47:55Z completed by a-root
