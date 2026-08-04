---
id: 01KZ694GEAJ8P398JQHAK91E1Z
kind: event
event_kind: commit
created: 2026-08-04T11:39:25Z
created_by: a-root
origin: agent
applied: false
---
2ee32dd 257: refuse to integrate a named task that is not done

Naming a task on the command line says which BRANCH to merge. It was
being read as a claim that the work was accepted.

Without --tasks the done filter is structural — ListTasks only returns
StatusDone — but a named list walked straight past it. That is how
fourteen tasks whose code had been merged for hours stayed in open/
while `dacli next` went on ranking them must, and it is exactly the
shape `dacli next` cannot recover from on its own: the backlog looked
full of urgent work that had already shipped.

integrationTasks now collects every named task that is not done and
refuses the whole run with exit 3, listing all of them, so one run tells
you the full set to close instead of one per attempt. The message names
the command that closes them. --force is the deliberate override for an
operator who knows the record is behind.

ship is unaffected: it integrates the set ListTasks already filtered to
done, so the gate is a no-op there.

Tests: the refusal (exit 3, names the task, names the fix, gh never
reached, nothing merged), the --force override, and a done task still
merging without --force so the normal path gains no friction.
role: root
