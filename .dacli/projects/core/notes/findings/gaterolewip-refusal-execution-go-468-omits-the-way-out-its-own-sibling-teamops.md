---
id: f-gaterolewip-refusal-execution-go-468-omits-the-way-out-its-own-sibling-teamops
kind: note
note_kind: finding
created: 2026-08-04T18:18:12Z
created_by: a-go-auditor-z48ata
about: "[[t-01KZ6S9Z9ZXRW12XZA1GJJSGB6]]"
source_event: 01KZ6SKJN1927EW6QKZZ1W68CK
---
# gateRoleWIP refusal (execution.go:468) omits the way-out its own sibling teamops.go:63 names
Two refusals fire on the IDENTICAL condition `store.ActiveInRole(w, roleName) >= role.WIP`, but only one names the action that would succeed.

SIBLING THAT NAMES THE WAY OUT — internal/features/teamops/teamops.go:63 (the `team assign` path):
  "role %s is at its WIP limit (%d/%d) — `dacli agent retire` one, or raise wip in the role file"

THE ONE THAT DOES NOT — internal/features/execution/execution.go:468, func gateRoleWIP:
  "role %s is at its WIP limit (%d/%d)"

gateRoleWIP is the MORE-traveled path: its own doc comment (execution.go:460-462) says it binds anything that mints an agent — both `dacli spawn` AND a supervised loop run — whereas teamops.go:63 only guards `team assign`. So the agent-facing spawn/supervise refusal is a dead end (exit 3, a contract it cannot argue with) while the human-facing assign refusal hands you the two remedies. An agent (or the loop) hitting the WIP cap on spawn is told the limit but not that it can `dacli agent retire` a finished agent to free a slot, or raise `wip` in the role file.

MISSING INSTRUCTION: append the sibling's suffix — `— `dacli agent retire` one, or raise wip in the role file`. This also connects to the open finding that a finished agent never releases its WIP slot (task 282): if retire is the documented escape, the spawn-path refusal must name it.

FIX: one-line change to execution.go:468 to match teamops.go:63. Verifiable: grep both call sites, assert both message strings carry the '`dacli agent retire` one, or raise wip' remedy.
