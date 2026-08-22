---
id: t-01M0N6P13JVFB9XCN6EYM0XF68
kind: task
created: 2026-08-22T17:00:51Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
github:
  issue: 774
  repo: mlnomadpy/dacli
---
# Make loop routing agree with team assignment and repository stack
## Acceptance
- [ ] For each selected implementation task, loop uses the same cost, capacity, kind, scope, and consequence-aware routing decision as team assign unless an explicit loop role override is set
- [ ] The default review phase selects a reviewer whose scope matches the project stack instead of hard-coding go-auditor for non-Go repositories
- [ ] Dry-run prints the selected role, model/runtime, decision source, and any consequence uplift for every implementation and review seat
- [ ] A Vue/Supabase fixture proves bounded UI work routes to a frontend fixer, transactional SQL work routes to a Supabase implementer, and review never selects go-auditor
- [ ] Focused orchestration tests, full Go tests, vet, format, and diff-check pass
## Log
- 2026-08-22T17:12:43Z claimed by a-maintainer-apg3xj
