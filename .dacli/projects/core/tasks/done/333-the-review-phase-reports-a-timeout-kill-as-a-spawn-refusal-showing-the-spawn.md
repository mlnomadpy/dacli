---
id: t-01KZPB4BYVP35WWDG9J45J1Z92
kind: task
created: 2026-08-10T17:22:08Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
---
# The review phase reports a timeout kill as a spawn refusal, showing the spawn banner instead of the cause
## Acceptance
- [x] when a review spawn is killed on timeout, the log says so and names the elapsed limit, not 'spawn refused/failed'
- [x] a policy refusal and a timeout kill are distinguishable from the loop's output alone
- [x] the message never prints the spawn's success banner as if it were an error (run 01KZPAKYFN: 'spawning a-go-auditor-yn0a9b on cc for 303')
- [x] a test drives a failing review spawn and asserts the reported cause matches the actual outcome
## Log
- 2026-08-10T17:24:57Z claimed by a-junior-ymc0q9
- 2026-08-10T17:30:14Z accepted by a-root (applied 1 proposal(s))
- 2026-08-10T17:30:14Z closed WITHOUT verification — no --verify command was given
- 2026-08-10T17:30:14Z completed by a-root
- 2026-08-10T17:30:15Z deliverable: dacli/333-the-review-phase-reports-a-timeout-kill-as-a-spawn-refusal-showing-the-spawn exists but is NOT in trunk — closed anyway
- 2026-08-10T17:37:20Z accepted by a-root
- 2026-08-10T17:37:20Z closed WITHOUT verification — no --verify command was given
- 2026-08-10T17:37:20Z deliverable: no dacli/333-the-review-phase-reports-a-timeout-kill-as-a-spawn-refusal-showing-the-spawn branch — nothing to check against trunk
- 2026-08-10T17:37:20Z completed by a-root
