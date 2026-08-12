---
id: f-gaterolewip-fails-open-on-an-unreadable-agents-dir-activeinrole-swallows
kind: note
note_kind: finding
created: 2026-08-12T13:44:41Z
created_by: a-go-auditor-vek0m1
about: "[[t-01KZ93DW7P6BQ1HHDWY7MEH2KJ]]"
source_event: 01KZPJDWMNN09X8KPATTG7WX7X
---
# gateRoleWIP fails OPEN on an unreadable agents dir: ActiveInRole swallows ListAgents error and returns 0
internal/store/roles.go:211 — ActiveInRole does 'agents, _ := ListAgents(w)', discarding the error. ListAgents (roles.go:178-181) returns (nil, err) when os.ReadDir(w.AgentsDir()) fails (ENOTDIR/permission/transient I/O). ActiveInRole then iterates an empty slice and returns 0.

Consumer: gateRoleWIP (internal/features/execution/execution.go:521-533), the FIRST launch gate (execution.go:373 {"role-wip", gateRoleWIP}). At :525 'if active := store.ActiveInRole(p.w, p.RoleName); active >= p.Role.WIP' — with active=0 and any WIP>=1, the condition is false, so the gate returns nil (PASS). Result: an unreadable agents dir reads as 'nobody is working in this role', and the WIP cap is waved through — the exact 'a gate must never certify what it could not read' failure the SIBLING gate gateClaimOverlap (execution.go:608-628) was hardened against by task 337, which now fails CLOSED on an unreadable runs dir (:612-620). The a-fixer-7s538v 337 finding explicitly flagged store.ActiveInRole/liveChildren as having 'the exact same swallowed-error shape' and recommended a follow-up; this is that follow-up.

Failure scenario: role fixer wip:1 already has a live fixer. On a spawn, w.AgentsDir() is replaced by a regular file (or a permission error occurs). ListAgents returns (nil, ENOTDIR); ActiveInRole returns 0; gateRoleWIP passes; a second fixer spawns, breaking the WIP invariant the gate exists to hold.

Note the runs-dir reads inside ActiveInRole (liveChildren roles.go:285-288, hasFinishedRun roles.go:261-264) also swallow errors but fail in the SAFE direction (over-count -> refuse). The dangerous swallow is the ListAgents one on line 211. Fix: propagate the ListAgents error out of ActiveInRole (signature -> (int, error)) and make gateRoleWIP fail closed on it, matching gateClaimOverlap.
