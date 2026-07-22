---
id: t-01KY4ZWW43218D6GGNA2XCPZQC
kind: task
created: 2026-07-22T13:23:01Z
created_by: a-root
owner: a-root
priority: should
estimate: {optimistic: 3, probable: 5, pessimistic: 8}
---
# D2: act on the log at spawn — dacli spawn --advise (budget, taint, role)
## Acceptance
- [ ] spawn --advise prints a suggested --budget from the calibrated band before launching (advises, does not decide)
- [ ] a task's taint status is shown before spawning a child on it
- [ ] next --parallel role suggestion is influenced by scope-matched lessons
- [ ] committed on branch by an agent; build + test green
## Log
