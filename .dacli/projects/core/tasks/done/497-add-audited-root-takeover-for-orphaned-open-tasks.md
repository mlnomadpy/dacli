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
- [x] Root can take an open task only when its non-root owner has no live process and no transcript-active run, using an explicit force/reason recovery form
- [x] The same takeover refuses while the owner is live or any run for that owner/task is transcript-active
- [x] Successful takeover immediately changes owner, records previous owner, new owner, reason, and recovery provenance durably, and preserves task history and pending proposals
- [x] Doctor stops reporting the recovered task as orphaned and recommends the executable takeover command for unfinished orphaned tasks instead of accept --force
- [x] A public-command regression reproduces root task claim 496 producing an owner-applied proposal for a dead owner, then proves audited takeover succeeds without delete/recreate
- [x] Mutation evidence and the full repository verification gates pass
## Log
- 2026-08-26T12:46:30Z claimed by a-fixer-ertmrt
- 2026-08-26T13:11:55Z completed by a-root
- 2026-08-26T13:21:21Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/780 (event 01M0Z2SHRXH62Q7EMTAFYRJXW6)
## Verification Evidence
{"command":"GOCACHE=/private/tmp/dacli-go-cache go test ./...","exit_code":0,"duration_ms":1658,"artifact_hash":"sha256:07db5a7f494d86162ce19dd327d134bb8ac313296b8ab5376f50a81125baa384","verifier":"a-root","branch":"dacli/497-add-audited-root-takeover-for-orphaned-open-tasks","commit_sha":"f7f738626d635ce3d461096292bc33356ad20ac7"}
