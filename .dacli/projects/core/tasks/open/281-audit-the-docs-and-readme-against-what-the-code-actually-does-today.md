---
id: t-01KZ6SBRBME7WQMTSBRHS55QCS
kind: task
created: 2026-08-04T16:22:59Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 2, probable: 4, pessimistic: 8}"
---
# Audit the docs and README against what the code actually does today
## So that
an agent that trusts the docs is not misled by a claim that was true three months ago
## Acceptance
- [ ] every command, flag and behavior named in README and docs is checked to exist and behave as described
- [ ] claims about counts, coverage, or status are re-derived rather than trusted
- [ ] documented behavior that is a stub or a plan is labelled as such rather than left reading as shipped
## Log
- 2026-08-04T16:25:15Z claimed by a-go-auditor-ag9yhz
- 2026-08-04T18:18:12Z finding by a-go-auditor-ag9yhz: README calls skill/shortcut promote stubs, but both are shipped, tested, and docs say so (event 01KZ6SKMZPT0VEBV9MVPPM9J4T)
