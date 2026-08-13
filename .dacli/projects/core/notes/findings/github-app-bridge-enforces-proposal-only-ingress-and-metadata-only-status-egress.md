---
id: f-github-app-bridge-enforces-proposal-only-ingress-and-metadata-only-status-egress
kind: note
note_kind: finding
created: 2026-08-13T15:09:54Z
created_by: a-maintainer-x2gz8j
about: "[[409]]"
severity: major
---
# GitHub App bridge enforces proposal-only ingress and metadata-only status egress
internal/githubapp/bridge.go:168 verifies the raw webhook body before parsing; :191 binds installation+repository to tenant/project; :205 repairs inbox/effect crash gaps; :337 rechecks revocation and checks:write before an idempotent outbox dispatch. Threat tests are in internal/githubapp/bridge_test.go:57-220 and the exact private pilot grant is executable in internal/githubapp/manifest_test.go. Mutation proof: replacing the signature condition with false && made TestWebhookVerifiesRawBodyBeforeInboxOrParsing fail at bridge_test.go:63 with error=<nil>, want ErrInvalidSignature.
