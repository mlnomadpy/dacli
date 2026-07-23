---
id: d-filed-the-runtimefiles-comma-corruption-bug-task-111-as-the-single-highest
kind: note
note_kind: decision
created: 2026-07-23T16:06:49Z
created_by: a-xrcxmhwz96
about: [[084]]
---
# Filed the runtimefiles comma-corruption bug (task 111) as the single highest-value change
## Chose
Filed the runtimefiles comma-corruption bug (task 111) as the single highest-value change
## Rejected
filing the TestVerdictReviewRendersRecordedVerdicts same-ms ULID flaky-test fix instead
## Because
both are grounded evidence-based defects, but the runtimefiles bug silently corrupts the flagship claude-code runtime's read-only sandbox argv (a security-adjacent enforcement surface), has a documented real corruption incident (cc.md hand-corrected after run 01KY2K8N4C), ships in the default preset, and has zero test coverage; the flaky test is test-only, severity minor, already has a trivial 2ms-sleep workaround noted, and does not affect product behavior
