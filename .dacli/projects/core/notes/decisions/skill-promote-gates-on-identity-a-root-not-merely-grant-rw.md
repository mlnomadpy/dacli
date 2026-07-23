---
id: d-skill-promote-gates-on-identity-a-root-not-merely-grant-rw
kind: note
note_kind: decision
created: 2026-07-23T09:49:09Z
created_by: a-sjnmw4x20p
about: [[089]]
---
# skill promote gates on identity == a-root, not merely grant rw
## Chose
skill promote gates on identity == a-root, not merely grant rw
## Rejected
gate on id.CanMutate / any rw-grant identity, matching the task-ownership pattern used by accept/task-check
## Because
SKILLS.md § 6 requires 'a human on the gate' for the lesson-to-skill escalation path (hostile file -> finding -> lesson -> auto-compiled standing instructions); a spawned agent can hold grant rw (e.g. via agent spawn --grant rw) yet is still exactly the actor the gate must block, so the check is agentid.RootID specifically (the only identity dacli resolves when no DACLI_AGENT token is set), not CanMutate
