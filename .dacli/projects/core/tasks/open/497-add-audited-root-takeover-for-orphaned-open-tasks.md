---
id: t-01M0Z1C74YZY1WKPYNWPCEPZE1
kind: task
created: 2026-08-26T12:40:31Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
github:
  issue: 778
  repo: mlnomadpy/dacli
---
# Add audited root takeover for orphaned open tasks
## Acceptance
- [ ] Root can take an open task only when its non-root owner has no live process and no transcript-active run, using an explicit force/reason recovery form
- [ ] The same takeover refuses while the owner is live or any run for that owner/task is transcript-active
- [ ] Successful takeover immediately changes owner, records previous owner, new owner, reason, and recovery provenance durably, and preserves task history and pending proposals
- [ ] Doctor stops reporting the recovered task as orphaned and recommends the executable takeover command for unfinished orphaned tasks instead of accept --force
- [ ] A public-command regression reproduces root task claim 496 producing an owner-applied proposal for a dead owner, then proves audited takeover succeeds without delete/recreate
- [ ] Mutation evidence and the full repository verification gates pass
## Log
- 2026-08-26T12:46:30Z claimed by a-fixer-ertmrt
