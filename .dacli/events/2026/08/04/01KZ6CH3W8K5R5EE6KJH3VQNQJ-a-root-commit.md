---
id: 01KZ6CH3W8K5R5EE6KJH3VQNQJ
kind: event
event_kind: commit
created: 2026-08-04T12:38:43Z
created_by: a-root
origin: agent
applied: true
---
a756ffa 165: point the child allowlist at the installed dacli, not the repo build output

All three runtimes allowlisted Bash(<repo>/dacli:*) — the build output
that sits inside the tree cc-rw and cc-fe hand the child Edit and Write
over. A child could overwrite the very binary the allowlist then let it
execute.

The task says 'remove the dacli binary from the allowlist' and that is
the one fix that does not work: children must run dacli, the entire
using-dacli contract is dacli commands, and a child that cannot record
its work has done work that did not happen. So the target moves out of
the writable tree instead — `go install ./cmd/dacli` puts it at
~/go/bin/dacli, which no worktree contains and no Edit/Write can reach.

Residual, written into the runtime files rather than left implicit:
Bash(go:*) is on the rw lists, so a determined child could
`go build -o ~/go/bin/dacli`. Closing that needs the binary somewhere
the agent's user cannot write at all — a root-owned /usr/local/bin —
which is an install the operator has to perform, not something a spawn
can arrange for itself. The operator also has to re-run `go install`
after pulling or the allowlisted binary drifts behind the source; that
is the cost of this fix and it is stated in the file.

Verified by asserting no runtime still names the repo path.

Also records the decisions behind 200 and 260, both dispatched with
those decisions in their briefs: role prompts are written as method, and
every role gets a declared max_points — only 3 of 18 had one, which is
the mechanism behind team assign answering junior for everything. And
the worktree-to-main workspace resolution is intended, because a
per-branch store forks the backlog, which is the failure 251 and the
branch-local finding both describe.
role: root
