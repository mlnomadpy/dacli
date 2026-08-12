---
id: d-handle-the-restricted-macos-warning-inside-the-self-hosting-fixture
kind: note
note_kind: decision
created: 2026-08-12T18:30:02Z
created_by: a-codex-maintainer-f85g9w
about: "[[391]]"
github:
  issue: 513
  repo: mlnomadpy/dacli
---
# Handle the restricted macOS warning inside the self-hosting fixture
## Chose
Handle the restricted macOS warning inside the self-hosting fixture
## Rejected
Override the claim gate to change the production VCS slice
## Because
dacli commit policy-refused internal/features/vcs outside this task's claim. The fixture-owned shim removes only the reproducible confstr diagnostic, preserves git status and all other stderr, and keeps real claim enforcement active.
