---
id: 01KZ70RKRWAGJ9WZDXPE6Q3WPE
kind: event
event_kind: commit
created: 2026-08-04T18:32:21Z
created_by: a-root
origin: agent
applied: true
---
f2db34e 297: gh repo view takes the repo positionally, so github push works again

`dacli github push` has been dead since 221 landed this morning. 221
routed every gh call through ghRepo, which appends --repo — correct for
issue create/edit/close/comment/list/view, label create and release
view, all of which inherit the flag. `gh repo view` does NOT: the
repository is its positional argument, and the flag makes gh exit 1 with
'unknown flag: --repo'.

repoView is the disclosure probe, the FIRST gh call push makes, so the
whole outbound mirror failed before writing anything. Nobody noticed
because I spent the day filing issues by hand instead of using the
command — which is exactly the habit the owner told me to stop.

Verified against the installed gh rather than reasoned about: issue
list/create/edit/close/comment/view, label create and release view all
accept --repo; repo view alone rejects it. Uniformity was the bug.

The tests are the point. 221 was green because captureArgs accepts any
argv — a stub like that can prove dacli CALLS gh, never that it calls it
with a shape gh understands. strictGH refuses --repo on repo view the
way real gh does, so a wrong argument shape now fails in CI instead of
on a live repository. Two more tests pin the remaining halves: the repo
is still targeted (positionally, since the flag is unavailable), and an
empty repo still falls back to cwd resolution for doctor and link.

Also files 296 — an agent token minted for a --worktree spawn is not
resolvable from inside that worktree, reported by the 275 agent through
`dacli report` after it could not commit, note, check or close. It
committed via raw git and told us. The escape hatch worked; the identity
resolution did not.
role: root
