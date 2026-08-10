---
id: 01KZ6S6K8E9RKQN6P5ANTFEVV1
kind: event
event_kind: commit
created: 2026-08-04T16:20:10Z
created_by: a-root
origin: agent
applied: true
---
2ab2537 275: file the gap I worked around, and the duplicate hazard the workaround created

Owner directive: when a dacli command cannot do the job, fix the
command — do not reach past it for the raw tool. Recorded as a decision
so it stops being my judgement call each time.

I filed issues 336-342 with `gh issue create` because
`dacli github push` has no task window and would have mirrored ~110
unmirrored tasks onto a public repo in one run. Safe for that moment,
wrong as a habit: dacli still cannot mirror a wave, the next operator
hits the same wall, and the product gets no better.

It also left real damage. Idempotency has two gates — the task's
`github:` mapping, and a marker in the issue body — and those seven
issues have NEITHER. The next `dacli github push` would find seven
unmapped tasks, search for a marker, find none, and create seven
duplicates on a public repository. Tasks 205 and 208 were both about
keeping push from duplicating under partial information; I reproduced
the same outcome by hand, from outside the tool.

So 275 is not just 'add a --tasks flag'. It has to do both: scope the
mirror to a window, AND adopt an issue that already exists for a task
when that issue carries no marker. Until it lands, do not run
`dacli github push` on this workspace — recorded as a finding, not
left as something I happen to remember.
role: root
