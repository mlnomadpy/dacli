---
id: f-loop-audit-six-shipped-capabilities-the-unattended-path-never-invokes
kind: note
note_kind: finding
created: 2026-08-09T23:10:35Z
created_by: a-root
about: "[[300]]"
severity: major
origin: internal/features/orchestration/orchestration.go
---
# Loop audit: six shipped capabilities the unattended path never invokes
Traced all six phases against d.run.run() call sites (spawn, wait, sync, ship, retro, record, accept, review). What the loop DOES do correctly, so it is not re-reported: it accepts only after prLandStatus confirms a merge (orchestration.go:871), so its accept --force overrides ownership only, never verification. What it never calls, and what an unattended run therefore does wrong: (1) team assign — implRole is fixed at config time (:171), so every task in a wave goes to one role regardless of size or cost; a 1-point task burns the expensive model and a task above the role's cap is spawned anyway rather than decomposed. (2) lint — the review phase files tasks that go straight to an implementer without the ambiguity check that exists precisely to catch a vague acceptance criterion before tokens are spent on it. (3) doctor — never run, so duplicate tasks, orphaned records and unparseable files stay invisible on the one path nobody is watching. (4) stage/gates — a project with a tdd template has its gate bypassed entirely; the loop lands work the gate would have refused. (5) estimate — the loop ranks by critical path but never sizes an unestimated task, so criticalPathSlack silently degrades to MoSCoW (:1728) and the wave loses the ordering it appears to be using. (6) catchup — agents are spawned with a brief frozen at spawn and are never told the command exists, so the duplicate-work problem 274 solved is still live for loop-spawned agents specifically.
