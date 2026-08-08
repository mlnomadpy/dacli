---
id: t-01KZ6S9FVG533ABEVXZBZZ7SC3
kind: task
created: 2026-08-04T16:21:45Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 2, probable: 4, pessimistic: 8}"
---
# Audit the agent lifecycle from the spawned agent's point of view
## So that
the experience of being a dacli agent is designed rather than inherited from whatever the operator needed
## Acceptance
- [x] each finding names the exact moment in the lifecycle it hurts: brief assembly, claim, work, record, land, or close
- [x] at least one finding covers something an agent cannot discover without reading dacli source
- [x] anything already in the backlog is reported as already filed, not re-filed
- [x] if a stage is genuinely good, that is stated rather than padded with a finding
## Log
- 2026-08-04T16:24:19Z claimed by a-reviewer-664htb
- 2026-08-04T18:18:12Z finding by a-reviewer-664htb: brief assembly: MillerCap truncates findings/decisions in alphabetical filename order, and trust-floor reflects only survivors (event 01KZ6SS75TY29WCQGH3K884DA8)
- 2026-08-04T18:18:12Z finding by a-reviewer-664htb: brief assembly: protocol preamble tells read-only agents 'you commit it, you open a PR' -- the default spawn grant cannot do either, and the same brief later says 'report and finish' (event 01KZ6SSJ68W9Q3E055KHV9Q84Q)
- 2026-08-04T18:18:12Z finding by a-reviewer-664htb: claim: an empty/lost DACLI_AGENT resolves to root RW (a-root) -- a spawned child that loses its token silently ESCALATES to full grant instead of failing closed (event 01KZ6SSX76157SW2TWA4B16AWT)
- 2026-08-04T18:18:12Z finding by a-reviewer-664htb: land: dacli pr --auto exits 0 with only a stderr note when auto-merge cannot be queued -- a headless agent reading exit code sees success while the PR is stranded open (event 01KZ6ST6V9FM1758TQKMXEC3FA)
- 2026-08-04T18:18:12Z finding by a-reviewer-664htb: close: dacli task done closes a task with an EMPTY acceptance section with zero verification; manual done/accept has no trunk-landing gate (loop-only 115 fix does not cover hand-driven agents) (event 01KZ6STK6AEZA2BAFHH556NGY5)
- 2026-08-04T18:18:12Z finding by a-reviewer-664htb: record stage is genuinely good: commit is honest about no-ops, unresolved role, and worktree crumb routing -- stated per the audit's 'say when a stage is good' rule, not a defect (event 01KZ6SV708N5HBN6MQ4RVGHY6K)
- 2026-08-04T18:26:18Z accepted by a-root
- 2026-08-04T18:26:18Z verified by `ls .dacli/projects/core/notes/findings/ | wc -l | awk '{exit ($1<16)}'` (exit 0)
- 2026-08-04T18:26:18Z completed by a-root
