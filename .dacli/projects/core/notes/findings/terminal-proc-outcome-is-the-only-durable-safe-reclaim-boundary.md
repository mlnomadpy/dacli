---
id: f-terminal-proc-outcome-is-the-only-durable-safe-reclaim-boundary
kind: note
note_kind: finding
created: 2026-08-19T13:23:06Z
created_by: a-maintainer-a5y9am
about: "[[t-01M0AGCX29Q047FZHKNG3YV0WC]]"
severity: major
---
# Terminal proc outcome is the only durable safe reclaim boundary
internal/features/vcs/vcs.go now scans every run whose worktree.txt resolves to the checkout and refuses recovery unless proc.txt is readable, names a child/run, has a non-empty terminal Outcome, and AliveRecord is false. The previous agentWorktreeOwner at vcs.go selected newest run without lifecycle checks.
