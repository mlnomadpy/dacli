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
- [x] internal/cloudsync persists an outbound queue and inbound cursor across restart using contracts/controlplane/v1 envelopes
- [x] sync retries are idempotent and automated tests cover delayed, duplicate, reordered, replayed, incompatible, and tampered fixtures
- [x] the client rejects invalid signatures and unsupported schema versions before mutating local state
- [x] go test ./internal/cloudsync ./contracts/controlplane/v1 passes
## Log
- 2026-08-13T16:07:34Z claimed by a-codex-maintainer-sg0bxk
- 2026-08-13T19:01:05Z accepted by a-root
- 2026-08-13T19:01:05Z verified by `GOCACHE=/private/tmp/dacli-425-main-gocache go test ./internal/cloudsync ./contracts/controlplane/v1` (exit 0) in branch main at 5fa7522 — proves that tree builds, not that the work is in trunk
- 2026-08-13T19:01:05Z deliverable: dacli/425-implement-metadata-only-offline-control-plane-sync-client is merged into main
- 2026-08-13T19:01:05Z completed by a-root
- 2026-08-13T19:11:12Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/594 (event 01KZY622D3K5F7H992A0YF7HJ0)
- 2026-08-13T23:59:12Z accepted by a-root
- 2026-08-13T23:59:12Z closed WITHOUT verification — no --verify command was given
- 2026-08-13T23:59:12Z deliverable: dacli/425-implement-metadata-only-offline-control-plane-sync-client is merged into main
- 2026-08-13T23:59:12Z completed by a-root
