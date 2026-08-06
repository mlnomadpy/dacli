---
id: d-290-make-pr-auto-return-a-non-zero-error-on-auto-merge-queue-failure-mirror
kind: note
note_kind: decision
created: 2026-08-04T20:13:18Z
created_by: a-maintainer-6qp5vh
about: "[[290]]"
---
# 290: make pr --auto return a non-zero error on auto-merge queue failure (mirror integrate sibling) and teach the loop's prLandStatus a 'stranded' state (OPEN PR with no auto-merge queued)
## Chose
290: make pr --auto return a non-zero error on auto-merge queue failure (mirror integrate sibling) and teach the loop's prLandStatus a 'stranded' state (OPEN PR with no auto-merge queued)
## Rejected
keep exit 0 and only print the stranded state to stdout
## Because
the finding shows the two --auto surfaces disagree: integrate --pr --auto already treats an unqueueable auto-merge as fatal (lifecycle.go:1239-1249) while cmdPR swallowed it to stderr+exit0 (lifecycle.go:288-295), teaching a headless agent the PR landed. Failing loudly matches the sibling and the exit-code contract (op failure=1). Stdout-only would still exit 0 and a caller keying off exit code stays fooled. The loop can't see the fixer's exit code (it parks by branch-commit presence), so it needs its own signal: prLandStatus must distinguish OPEN+auto-merge-queued (landing) from OPEN+not-queued (stranded).
