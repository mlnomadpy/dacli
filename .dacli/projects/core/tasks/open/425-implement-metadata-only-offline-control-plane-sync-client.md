---
id: t-01KZXXZP8GE9TB394TV5G34AKD
kind: task
created: 2026-08-13T16:06:19Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 5, probable: 8, pessimistic: 13}"
parent: "[[t-01KZXS3QKA4MHMQZNK7B9VQ7V5]]"
github:
  issue: 592
  repo: mlnomadpy/dacli
---
# Implement metadata-only offline control-plane sync client
## So that
the local runtime can exchange signed v1 summaries without uploading source, prompts, secrets, or command output
## Acceptance
- [ ] internal/cloudsync persists an outbound queue and inbound cursor across restart using contracts/controlplane/v1 envelopes
- [ ] sync retries are idempotent and automated tests cover delayed, duplicate, reordered, replayed, incompatible, and tampered fixtures
- [ ] the client rejects invalid signatures and unsupported schema versions before mutating local state
- [ ] go test ./internal/cloudsync ./contracts/controlplane/v1 passes
## Log
- 2026-08-13T16:07:34Z claimed by a-codex-maintainer-sg0bxk
