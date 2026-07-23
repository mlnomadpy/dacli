---
id: 01KY7RTB3ZCD7F8F3W6DNZE8NW
kind: event
event_kind: finding
created: 2026-07-23T15:17:01Z
created_by: a-qy5e8fvxm5
about: [[t-01KY60QM1Y7DK05WXB954YNDHJ]]
origin: agent
applied: false
---
Three git/gh subprocesses still unbounded; gitx's 'every git child' deadline invariant is false in the current tree

gitx.go:15-19 states deadlines 'bound every git child so a hung subprocess (a credential-helper prompt, a wedged network push) can never block the caller ... a correctness property, not a nicety.' That invariant is currently violated at three call sites left out of tasks 018 and 105 (105 was scoped strictly to orchestration.go/driver.git):

NETWORK ops (real hang risk):
- internal/skills/skills.go:170 — exec.Command("git","clone","--depth","1","-q",url,tmp) with no context/deadline. 'dacli skill install owner/repo' (skills.Fetch) hangs indefinitely on a wedged network or a credential-helper prompt. Fix: route through gitx.RunNetwork.
- internal/features/collab/collab.go:236 — exec.Command("gh","issue","create",...).Output() with no context/deadline, on the 'dacli escalate --github' path. A hung gh API call blocks the caller; every other gh call site already uses exec.CommandContext (selfreport.go:108, vcs/lifecycle.go:47, ghmirror.go:58). Not named in the earlier f-g3ya9r93e3 finding — new observation.

LOCAL ops (lower risk, complete the invariant):
- internal/store/version.go:98,131,139,148 — four bare exec.Command("git",...).Output() calls in FileChangelog/VersionIsStale, no deadline. Named as sibling violators in f-g3ya9r93e3 but explicitly left out of task 105's strict scope.

Evidence: grep for 'exec.Command("git"|exec.Command("gh"' across internal (excluding tests) returns exactly these plus the already-fixed gitx/vcs/ghmirror/catalog/selfreport CommandContext sites. Under 'dacli mcp serve' a blocked git/gh freezes the whole stdio loop (gitx.go:17), so this is the same correctness class task 018 set out to close.
