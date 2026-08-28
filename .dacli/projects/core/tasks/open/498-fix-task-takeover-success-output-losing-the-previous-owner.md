---
id: t-01M0Z37RY5ZARHCM8K3PKCYS55
kind: task
created: 2026-08-26T13:13:03Z
created_by: a-root
owner: a-root
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
github:
  issue: 782
  repo: mlnomadpy/dacli
---
# Fix task takeover success output losing the previous owner
## Acceptance
- [ ] After a successful task takeover, stdout names the non-root previous owner rather than a-root
- [ ] The stdout previous owner matches the previous owner stored in the durable takeover log
- [ ] A public-command regression fails if WithTask refresh overwrites the value used for success reporting
- [ ] Mutation evidence and the full repository verification gates pass
## Log
