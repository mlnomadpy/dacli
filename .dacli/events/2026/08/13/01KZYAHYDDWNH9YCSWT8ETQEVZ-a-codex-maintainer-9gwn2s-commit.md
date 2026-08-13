---
id: 01KZYAHYDDWNH9YCSWT8ETQEVZ
kind: event
event_kind: commit
created: 2026-08-13T19:46:00Z
created_by: a-codex-maintainer-9gwn2s
about: "[[t-01KZY9YGP5XDD748GEGQG2ACZ2]]"
origin: agent
applied: true
---
32fc85b 429: install pinned Syft before tagged releases

Mutation proof before the fix:
--- FAIL: TestReleaseInstallsPinnedSyftBeforeGoReleaser
    contract_test.go:55: release workflow must install a pinned Syft distribution

Reordering mutation:
--- FAIL: TestReleaseInstallsPinnedSyftBeforeGoReleaser
    contract_test.go:62: release workflow must install Syft before GoReleaser
role: codex-maintainer
