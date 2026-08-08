---
id: d-filed-docs-readme-md-status-honesty-drift-task-129-as-the-single-highest-value
kind: note
note_kind: decision
created: 2026-07-23T19:57:17Z
created_by: a-zq4qdv7py6
about: [[084]]
---
# Filed docs/README.md status-honesty drift (task 129) as the single highest-value change
## Chose
Filed docs/README.md status-honesty drift (task 129) as the single highest-value change
## Rejected
the prior sibling's github-pull closed-issue import bug, and stray-marker / TODO cleanups
## Because
The github-pull bug is refuted — shouldImport already skips closed+unmapped issues (ghmirror.go:382) and is tested (ghmirror_test.go:164); there are no TODO/FIXME markers in product code and zero clikit.Planned call sites remain. The one remaining statically-verifiable, evidence-grounded defect is docs/README.md labeling three shipped features (shortcut promote, skill promote, github inbound) as planned/unimplemented on lines 12/14/18/24 — an honesty defect in the doc whose own line 3 makes status-honesty its purpose, directly serving the project goal that every planned() stub be implemented honestly.
