---
id: f-task-385-implementation-committed-on-isolated-branch
kind: note
note_kind: finding
created: 2026-08-12T16:52:01Z
created_by: a-codex-maintainer-nmzkpw
about: "[[385]]"
severity: major
---
# Task 385 implementation committed on isolated branch
Commit aa2e485 on branch dacli/385-derive-loop-path-claims-from-the-implementation-scope-required-by-acceptance adds behavioral implementation-scope inference plus task-371 store and loop-spawn regressions. Red proof: ClaimHints was [docs/RUNTIMES.md] and missed all three code trees. Focused tests, gofmt, and go vet pass; lint is unavailable and the mandatory CLI race gate is blocked by the pre-existing spawned-worker sandbox failure.
