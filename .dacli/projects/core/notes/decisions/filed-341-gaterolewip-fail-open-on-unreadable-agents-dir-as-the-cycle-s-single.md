---
id: d-filed-341-gaterolewip-fail-open-on-unreadable-agents-dir-as-the-cycle-s-single
kind: note
note_kind: decision
created: 2026-08-10T19:29:58Z
created_by: a-go-auditor-vek0m1
about: "[[303]]"
---
# Filed 341 (gateRoleWIP fail-open on unreadable agents dir) as the cycle's single highest-value item
## Chose
Filed 341 (gateRoleWIP fail-open on unreadable agents dir) as the cycle's single highest-value item
## Rejected
Filing the ActiveInRole runs-dir swallows (liveChildren/hasFinishedRun), or re-filing anything under the CheckAllAcceptance/317/337 headings already fixed or queued
## Because
The runs-dir swallows inside ActiveInRole fail SAFE (over-count -> refuse); only the ListAgents swallow at roles.go:211 fails OPEN, and it does so on the loop's own spawn path through the FIRST launch gate (execution.go:373), waving a spawn past the WIP cap the gate exists to hold. It is the exact 337 class ('a gate must never certify what it could not read') on a structurally separate sibling gate — the inconsistency-between-neighbours tell: gateClaimOverlap now fails closed (execution.go:612-620) while gateRoleWIP still fails open. The 337 fixer explicitly recommended this as a follow-up and it is not queued (only 303/338 open; 282/331 cover the finished-agent-holds-slot bug, 295 covers the refusal message — none touch the swallowed read). Code-cited, reachable, small well-scoped fix.
