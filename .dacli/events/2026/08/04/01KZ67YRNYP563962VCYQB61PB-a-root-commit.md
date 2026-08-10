---
id: 01KZ67YRNYP563962VCYQB61PB
kind: event
event_kind: commit
created: 2026-08-04T11:18:48Z
created_by: a-root
origin: agent
applied: true
---
e2f801b 255: reuse an already-open PR instead of aborting the whole integrate run

The integrator's sanctioned merge path could not merge any PR the loop
had opened — which is every PR it will ever be pointed at. openPR calls
`gh pr create` unconditionally, and gh hard-fails on 'already exists',
so prIntegrateTask returned before it ever reached the --auto queue or
the check-gated merge below it. Found by the integrator agent spawned to
merge this session's wave: it filed the finding instead of merging,
because it could not merge.

openPR now probes `gh pr view <branch>` first and reuses a PR that is
OPEN. Anything else — no PR, a closed or merged one, an unreachable
GitHub, unparseable output — reports 'none' and falls through to create,
so the probe can only ever remove a spurious failure; it never invents a
PR. Reuse deliberately skips the eventlog append and the review post:
the create path already did both, and re-posting would stack a duplicate
review on the PR every time an integration run touched it.

openPR returns (url, reused, err) so callers can say which happened.
Nothing after that point distinguishes them, which is the point.

Tests cover the three branches: an already-open PR reaches the merge, a
CLOSED PR still gets created, and a failed probe still gets created.

Also reconciles a duplicate seq: two different tasks were both numbered
250 once #284 and #292 merged, exactly as task 251 predicts. The
doctor-anchor one becomes 254.
role: root
