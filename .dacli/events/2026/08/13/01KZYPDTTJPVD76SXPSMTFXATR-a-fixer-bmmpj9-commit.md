---
id: 01KZYPDTTJPVD76SXPSMTFXATR
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-13T23:13:28Z
created_by: a-fixer-bmmpj9
about: "[[t-01KZYNVFR96911X4D1V3D9T98Y]]"
origin: agent
applied: true
checksum: sha256:1b0bba11bf5317c18d7288f5ec97c125ec9c209515e11a625a0fa61349cb10ec
---
51e8302 439: repair stale worktree removal after prune preview

When an interrupted teardown leaves a registered checkout without its .git link, git worktree remove refuses it even though preview identifies it as reclaimable. Prune that stale registration, verify Git released the path, and remove the orphaned directory.

Mutation evidence: restoring the old RemoveWorktree body made TestWorktreePruneReclaimsFinishedMissingCheckoutAfterDryRun fail at lifecycle_test.go:271 because apply printed 'reclaimed 0 worktree(s)'.
role: fixer
