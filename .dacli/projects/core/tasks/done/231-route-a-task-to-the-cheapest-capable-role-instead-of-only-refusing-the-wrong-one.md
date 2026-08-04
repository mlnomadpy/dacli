---
id: t-01KZ4WXGJ4ZBB2YAQ36E1W98NA
kind: task
created: 2026-08-03T22:46:38Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 3, probable: 6, pessimistic: 12}"
---
# Route a task to the cheapest capable role instead of only refusing the wrong one
## So that
easy work runs on cheap models without an operator choosing the role by hand
## Acceptance
- [x] given a sized task the tool selects the lowest-cost role whose cap covers it and whose kind fits the phase
- [x] the choice and its cost are reported
## Log
- 2026-08-04T00:02:37Z completed by a-root
