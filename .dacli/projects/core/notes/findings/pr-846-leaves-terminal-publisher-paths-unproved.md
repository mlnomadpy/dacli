---
id: f-pr-846-leaves-terminal-publisher-paths-unproved
kind: note
note_kind: finding
created: 2026-08-28T08:54:32Z
created_by: a-root
about: "[[511]]"
severity: major
---
# PR 846 leaves terminal publisher paths unproved
Commit 16dda175 streams mutating gh output and covers success plus immediate retry, but task 511 criteria 3 and 5 also require failure, cancellation, timeout, process-tree cleanup, and race evidence. ghExec still uses default exec.CommandContext cancellation, which kills only the direct gh process; no regression makes gh fork a child that retains output or the sequence lease. Do not accept until those paths are deterministically tested and the process tree is drained before return.
