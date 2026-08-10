---
id: t-01KYFYTFFZ3R3SXR1F09PJMA6H
kind: task
created: 2026-07-26T19:35:53Z
created_by: a-root
owner: a-root
priority: should
github:
  issue: 251
  repo: mlnomadpy/dacli
---
# Release v0.1.0: create mlnomadpy/homebrew-tap repo + set HOMEBREW_TAP_GITHUB_TOKEN secret, then push the v0.1.0 tag
## So that
the verified goreleaser pipeline actually publishes binaries + the brew formula (currently blocked only on these operator steps)
## Acceptance
- [ ] mlnomadpy/homebrew-tap repo exists and HOMEBREW_TAP_GITHUB_TOKEN is set on the dacli repo
- [ ] git tag v0.1.0 && git push origin v0.1.0 triggers a green release; brew install mlnomadpy/tap/dacli works
## Log
- 2026-07-26T21:04:48Z blocked: 
- 2026-08-10 DEFERRED BY OWNER DECISION: releasing is out of scope until Taha says a version is solid enough to publish. Not blocked on the PAT — the token was never the real constraint. Do not re-file, re-open, or propose follow-ups to unblock this; see the project's Out of scope section and the decision note.
