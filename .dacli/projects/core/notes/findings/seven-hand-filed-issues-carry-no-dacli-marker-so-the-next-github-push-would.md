---
id: f-seven-hand-filed-issues-carry-no-dacli-marker-so-the-next-github-push-would
kind: note
note_kind: finding
created: 2026-08-04T16:19:28Z
created_by: a-root
severity: major
origin: internal/features/ghmirror/ghmirror.go:1434
---
# Seven hand-filed issues carry no dacli marker, so the next github push would duplicate every one
I filed issues 336-342 for tasks 268-274 with `gh issue create` rather than `dacli github push`, because push has no task window and would have mirrored roughly 110 unmirrored tasks onto a public repo in one run.

The consequence is the exact failure the marker system exists to prevent. Idempotency has two gates: `mappedIssue(t)` reads the task's `github:` frontmatter block, and failing that `markerIndex.find` searches issue bodies for `<!-- dacli:<id> ws:<ws> -->`. Those seven issues have NEITHER — no mapping was written to the task files, and no marker was written into the issue bodies. So the next `dacli github push` sees seven unmapped tasks, finds no marker, and creates seven duplicate issues on a public repository.

That makes this more than a cosmetic gap. Tasks 205 and 208 were both about keeping push from duplicating issues under partial information; I reintroduced the same outcome by hand, from the outside.

Two things have to be true for 275 to actually close this: push needs a task window so the scoped mirror is possible at all, AND it needs to adopt an issue that already exists for a task even when that issue's body carries no marker — matching on something the hand-filed issues do have. Until then, do not run `dacli github push` on this workspace.
