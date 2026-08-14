---
id: f-audit-filed-distinct-task-454-for-ambiguous-generated-mutation-refs
kind: note
note_kind: finding
created: 2026-08-14T01:56:49Z
created_by: a-codex-loop-auditor-ejgvrk
about: "[[t-01KZXRP56538HNHDP3P4FJHGGP]]"
severity: major
---
# Audit filed distinct task 454 for ambiguous generated mutation refs
Reproduced in a clean temporary workspace: alpha/001 and beta/001 make 'dacli task check 001 --n 1' fail with 'ref 001 is ambiguous' (exit 1). internal/features/execution/execution.go:2116 and :2178 feed fmt.Sprintf("%03d", t.Seq) to protocol_preamble.md and git_workflow.md, which generate mutating commands. Required open and active core backlog checks found no semantic duplicate; linked GitHub issue lookup was attempted but api.github.com was unreachable. Filed t-01KZYZR9B4312NVSTNS8NMJ1CE with three checkable acceptance criteria. No product files were edited; git status remained clean on main.
