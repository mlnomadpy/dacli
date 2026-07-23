---
id: t-01KY8536GN49XG8HWTMTW1PBEP
kind: task
created: 2026-07-23T18:51:34Z
created_by: a-root
owner: a-root
priority: must
---
# Cross-platform release: GoReleaser + macOS/Linux/Windows binaries + Homebrew formula + release workflow
## Acceptance
- [x] A .goreleaser.yaml builds darwin/linux/windows amd64+arm64 archives and a .github/workflows/release.yml runs it on a version tag
- [x] A Homebrew formula (brew tap template) installs the binary; README documents 'brew install' and direct-download install; no real tag/publish is triggered (that stays a manual step)
## Log
- 2026-07-23T18:54:10Z claimed by a-jze25mtf8b
- 2026-07-23T18:57:56Z accepted by a-root
- 2026-07-23T18:57:56Z completed by a-root
