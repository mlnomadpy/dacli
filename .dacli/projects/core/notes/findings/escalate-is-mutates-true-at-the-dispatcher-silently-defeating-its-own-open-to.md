---
id: f-escalate-is-mutates-true-at-the-dispatcher-silently-defeating-its-own-open-to
kind: note
note_kind: finding
created: 2026-08-11T10:07:48Z
created_by: a-fixer-kf182p
about: "[[353]]"
severity: moderate
---
# escalate is Mutates:true at the dispatcher, silently defeating its own 'open to any read-only agent' design intent
internal/features/collab/collab.go:25 declares {Path: "escalate", Mutates: true, ...}. The dispatcher's refuseUngrantedMutation (internal/cli/cli.go:227) runs BEFORE cmdEscalate and refuses any ro caller outright (exit 3) for the whole command, no --dry-run support to exempt it. But cmdEscalate's own comment at collab.go:293 says: 'The local escalation above is open to any agent — that is the point,' and only gates --github's remote gh issue create behind an explicit clikit.RequireRW (collab.go:295-297) — a redundant, now-dead-in-the-ro-case check, since a ro caller never reaches that line at all. So today a read-only agent cannot file the local help-event (dacli escalate "msg", no --github) despite the code's own stated design that it should be able to — DESIGN.md § 6 and the event-kind table (FORMAT.md) both list 'help' as an append-only event a ro agent should be able to write, the same class as claim/finding/comment/block. Likely cause: Mutates was added at the dispatcher level (2026-08-06 audit per cli.go:216-221) as a blanket table-driven gate, and escalate's own pre-existing per-flag RequireRW(--github) was never revisited to split the command in two, or to have Mutates declared as false with the --github branch keeping its own explicit RW check (the pattern 'run' and several ghmirror commands already use for effect-gated sub-behavior). Not fixed as part of task 353 (documentation-only scope) — documented as the CURRENT, verified behavior in docs/TRUST.md § 5 rather than the aspirational comment.
