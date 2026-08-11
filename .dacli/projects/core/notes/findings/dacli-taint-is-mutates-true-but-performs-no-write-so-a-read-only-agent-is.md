---
id: f-dacli-taint-is-mutates-true-but-performs-no-write-so-a-read-only-agent-is
kind: note
note_kind: finding
created: 2026-08-11T10:10:13Z
created_by: a-fixer-kf182p
about: "[[353]]"
severity: moderate
---
# dacli taint is Mutates:true but performs no write, so a read-only agent is refused from running the one command built for auditing suspected prompt injection
internal/features/insight/insight.go:33 declares {Path: "taint", Mutates: true, ...}. cmdTaint (insight.go:922-969) opens the workspace, calls store.Taint (internal/store/taint.go), and only writes to ctx.Stdout. store.Taint itself is documented as a pure read in its own doc comment ('it does NOT prevent injection... converts attribution into a command') and its call path (filepath.WalkDir + mdstore.ReadFile over events, ListNotes over notes) contains no os.WriteFile, SaveTask, or eventlog.Append anywhere. Confirmed independently by two separate research passes over the handler and store code during task 353's documentation work, tracing every call. Effect: the dispatcher's refuseUngrantedMutation (internal/cli/cli.go:227) refuses a ro agent's dacli taint <origin> outright (exit 3) before cmdTaint ever runs, and taint has no --dry-run flag to route around it either. This is backwards for the tool's own threat model: a ro child is exactly the agent most likely to be investigating whether IT is the one that got poisoned, and it is the one grant tier taint refuses outright. Not fixed as part of task 353 (documentation-only scope, and flipping Mutates on a live command is a behavior change, not a doc change) — documented as the current, verified behavior in docs/TRUST.md § 5 rather than silently assumed to write something.
