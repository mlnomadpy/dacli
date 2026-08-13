---
id: t-01KZX7PY0M2301GVABGAXH49TY
kind: task
created: 2026-08-13T09:37:03Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 5, probable: 8, pessimistic: 13}"
depends_on: "[408:FS]"
github:
  issue: 553
  repo: mlnomadpy/dacli
---
# Build the least-privilege GitHub App event bridge
## So that
GitHub can steer and observe dacli continuously without replacing local state or depending on a human gh session
## Context
Use a webhook Adapter feeding an idempotent Inbox, a transactional Outbox for GitHub writes, and periodic reconciliation. Installation tokens are minted server-side only; local clients receive signed commands or events, never the App private key.
## Acceptance
- [x] A private pilot App requests the minimum documented metadata, issues, pull requests, and checks permissions required by implemented endpoints
- [x] Webhook ingestion verifies X-Hub-Signature-256 on the raw body, then parses the payload; the inbox insert for X-GitHub-Delivery returns new or duplicate, and effects run for new
- [x] Installation suspension, repository removal or rename, permission changes, retries, redelivery, and out-of-order events converge through reconciliation
- [x] Inbound GitHub actions become dacli proposal events and cannot directly mutate or execute the local workspace
- [x] Outbound status uses an idempotent outbox and posts no source, prompt, transcript, environment, secret, or raw command output
- [x] Threat-model tests cover tenant and installation mapping, replay, confused deputy, forged webhook, and revoked installation
## Log
- 2026-08-13T15:00:59Z claimed by a-maintainer-x2gz8j
- 2026-08-13T15:12:59Z accepted by a-root
- 2026-08-13T15:12:59Z closed WITHOUT verification — no --verify command was given
- 2026-08-13T15:12:59Z deliverable: dacli/409-build-the-least-privilege-github-app-event-bridge exists but is NOT in main — closed anyway
- 2026-08-13T15:12:59Z completed by a-root
- 2026-08-13T15:20:56Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/586 (event 01KZXTY6D80E9CHP3JKE8YBWX3)
