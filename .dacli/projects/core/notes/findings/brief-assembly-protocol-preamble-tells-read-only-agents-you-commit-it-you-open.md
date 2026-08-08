---
id: f-brief-assembly-protocol-preamble-tells-read-only-agents-you-commit-it-you-open
kind: note
note_kind: finding
created: 2026-08-04T18:18:12Z
created_by: a-reviewer-664htb
about: "[[t-01KZ6S9FVG533ABEVXZBZZ7SC3]]"
source_event: 01KZ6SSJ68W9Q3E055KHV9Q84Q
---
# brief assembly: protocol preamble tells read-only agents 'you commit it, you open a PR' -- the default spawn grant cannot do either, and the same brief later says 'report and finish'
Stage: BRIEF ASSEMBLY (describing CLAIM/RECORD/LAND). protocol_preamble.md:7 unconditionally renders: 'Your lifecycle is claim -> work -> commit -> pr -> accept/ship: you have claimed task {{.Ref}}, you do the work, you commit it, you open a PR, and the owner accepts and ships it.' This line is NOT guarded by {{if .RW}} -- the only RW/ro branch in the template is around lines 19-26 (task check/done vs 'report and finish'). But read-only is the DEFAULT grant for spawned agents (collab.go:112-113 comment: 'read-only child ... is the DEFAULT grant for spawned agents'), and a ro agent is REFUSED at dacli commit (vcs.go:81-83 'committing writes to the repo; that needs an rw grant'), has no push, and no pr path. So the assembled brief tells a ro agent to walk a lifecycle it will be refused on at step 3, then contradicts itself at line 25 ('Your grant is read-only: dacli turns your reports into events the owner applies. report and finish.'). Self-evidencing: THIS reviewer's own brief (a-reviewer-664htb, grant: ro) contains both sentences verbatim. Fix: guard the lifecycle sentence on .RW, or give ro agents a claim->work->report arc.
