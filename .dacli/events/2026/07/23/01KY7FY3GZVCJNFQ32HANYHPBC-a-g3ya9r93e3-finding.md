---
id: 01KY7FY3GZVCJNFQ32HANYHPBC
kind: event
event_kind: finding
created: 2026-07-23T12:41:47Z
created_by: a-g3ya9r93e3
about: [[t-01KY60QM1Y7DK05WXB954YNDHJ]]
origin: agent
applied: false
---
Perpetual loop's git subprocesses have no deadline; a hung 'git fetch origin' freezes the whole loop

internal/features/orchestration/orchestration.go:416-421 driver.git() uses plain exec.Command("git", args...) with NO context/deadline. This violates the codebase's explicit convention: gitx.go:15-23 states deadlines 'bound every git child so a hung subprocess (a credential-helper prompt, a wedged network push) can never block the caller ... a correctness property, not a nicety', and every other production git/gh call site uses exec.CommandContext (gitx.go:34, vcs/vcs.go:59, vcs/lifecycle.go:47, ghmirror.go:58, selfreport.go:108, catalog.go:308,327). driver.git is worse than the others: trunkMarker() (orchestration.go:396-414) calls d.git("fetch","-q","origin",b) — a NETWORK op — unconditionally on EVERY cycle of the always-on perpetual loop (called at orchestration.go:222 and 268). On a wedged network or a credential-helper prompt, that fetch blocks indefinitely and freezes the entire loop: no window, no thrash-guard, no stop-file check can fire because the process is stuck inside git. Task 018 gave git/gh subprocesses deadlines, but orchestration was added afterward and reintroduced the unbounded-git antipattern. Fix: route driver.git through internal/gitx (entity plumbing, importable by slices) or wrap in exec.CommandContext with localTimeout for local ops and networkTimeout for the fetch. Sibling lower-risk violators to fix alongside: internal/skills/skills.go:170 (git clone, network) and internal/store/version.go:98,131,139,148 (local ops).
