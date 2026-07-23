---
id: t-01KY8GDJ8G31R81D72AMGYSCQ5
kind: task
created: 2026-07-23T22:09:28Z
created_by: a-root
owner: a-root
priority: must
---
# Fix deprecated goreleaser 'brews' config so 'goreleaser check' passes clean (blocks the v0.1.0 release)
## Acceptance
- [x] goreleaser check passes on .goreleaser.yaml with no deprecation warnings (migrate 'brews' to the current schema)
- [x] the Homebrew formula still installs the dacli binary and the release build (goreleaser release --snapshot --clean) succeeds locally; go build stays green
## Log
- 2026-07-23T22:23:10Z claimed by a-53n5n76v8q
- 2026-07-23T22:33:45Z accepted by a-root
- 2026-07-23T22:33:45Z completed by a-root
