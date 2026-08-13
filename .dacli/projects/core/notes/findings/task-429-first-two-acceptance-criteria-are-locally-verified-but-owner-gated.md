---
id: f-task-429-first-two-acceptance-criteria-are-locally-verified-but-owner-gated
kind: note
note_kind: finding
created: 2026-08-13T19:46:00Z
created_by: a-codex-maintainer-9gwn2s
about: "[[429]]"
severity: major
---
# Task 429 first two acceptance criteria are locally verified but owner-gated
.github/workflows/release.yml installs Syft 1.50.0 before GoReleaser, and .github/workflows/contract_test.go fails for both omission and reordering mutations. task check 429 --n 1 and --n 2 were refused because only owner a-codex-loop-auditor-hxqjcg may check acceptance.
