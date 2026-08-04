---
id: f-live-seq-collision-two-different-tasks-both-hold-core-251-will-block-dacli
kind: note
note_kind: finding
created: 2026-08-04T11:45:22Z
created_by: a-maintainer-88hjw4
about: "[[251]]"
severity: moderate
---
# Live seq collision: two different tasks both hold core/251 (will block 'dacli accept 251')
The workspace ALREADY contains the exact defect task 251 fixes: two distinct tasks share seq 251 -- core/251-testiddoesnotleakthetoken-asserts-on-4-char-windows-and-fails-by-coincidence and core/251-task-seq-is-allocated-against-the-working-tree-so-two-branches-hand-out-the. 'dacli task check 251' and 'dacli <251>' fail 'ref 251 is ambiguous', so the owner must reference this task by its full slug (or renumber one file) to accept/integrate it. This is precisely what the new doctor 'collided-seq' check surfaces (store.CollidedSeqs). Reconciliation = renumber one of the two files to the next free seq so the ref resolves; a seq is a live reference (branch/worktree/PR names) so this is left as an explicit owner action rather than auto-renumbered.
