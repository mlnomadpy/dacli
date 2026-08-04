---
id: d-filed-the-readme-says-promote-is-a-stub-contradiction-as-the-single-highest
kind: note
note_kind: decision
created: 2026-08-04T16:27:26Z
created_by: a-go-auditor-ag9yhz
about: "[[281]]"
---
# Filed the README-says-promote-is-a-stub contradiction as the single highest-value doc-audit finding
## Chose
Filed the README-says-promote-is-a-stub contradiction as the single highest-value doc-audit finding
## Rejected
Filing the stale '96 merged PRs' count on README.md:11 (git log already reaches PR #335), or re-auditing SPM/command-reference claims that spot-check as accurate
## Because
The stub misclaim is a strict correctness defect in the record: README.md:85 and :278 assert two SHIPPED, tested commands (skillforge.go:86-138, shortcuts.go:cmdPromote) are unimplemented stubs, and clikit.Planned() now has zero callers — the README contradicts its OWN linked docs (docs/README.md:30, docs/SKILLS.md:3, docs/SHORTCUTS.md:3) on the same tree, and a test literally comments 'no more planned stubs left'. That is the auditor's #1 class (the record disagreeing with reality) and cheaply verifiable. The '96 merged PRs' count is a known drifting number already tracked by a prior decision note and re-updated once (77->96); it is observability churn, strictly lower value than a doc that tells an agent a working command is broken.
