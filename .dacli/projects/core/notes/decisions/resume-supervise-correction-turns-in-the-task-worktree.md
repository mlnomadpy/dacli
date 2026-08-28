---
id: d-resume-supervise-correction-turns-in-the-task-worktree
kind: note
note_kind: decision
created: 2026-08-28T09:31:30Z
created_by: a-fixer-5xjt19
about: "[[t-01M13V42QDZE7CKYDFWYVB5YG5]]"
---
# Resume supervise correction turns in the task worktree
## Chose
Resume supervise correction turns in the task worktree
## Rejected
Launch every correction from the shared workspace root
## Because
A root-reclaimed worktree needs each new correction run recorded in and launched from that checkout so its child has a current task-scoped ownership record for governed commits.
