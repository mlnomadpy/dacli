---
id: d-232-make-advise-preview-only-on-spawn-supervise-unify-with-loop-rather-than
kind: note
note_kind: decision
created: 2026-08-04T11:46:56Z
created_by: a-maintainer-vrmdqz
about: "[[232]]"
---
# 232: make --advise preview-only on spawn/supervise (unify with loop) rather than renaming the spawn flag
## Chose
232: make --advise preview-only on spawn/supervise (unify with loop) rather than renaming the spawn flag
## Rejected
Rename spawn's --advise to a distinct name (e.g. --preflight/--sizing) so its inform-then-launch behavior survives under a non-'advise' word
## Because
Acceptance lists 'advise previews without acting on both commands' FIRST and it yields ONE honest meaning for the word across the whole CLI (loop/spawn/supervise: look, don't act), directly satisfying the So-that 'a flag named advise cannot cost money'. resolveLaunch is shared by cmdSpawn+cmdSupervise, so a single early-return sentinel (errAdviseOnly, before any gate/mint/exec) fixes both call sites at once. The old axiom-3 note (d-084/028: advise additive, spawn proceeds) is a launch DECISION concern, not violated by an operator explicitly choosing to preview. Rename was rejected because it leaves an ad-hoc extra flag name and the operator loses the single mental model; the exit-0 preview is not a silent no-op because printAdvisory prints a loud 'no agent spawned' line and no run-id.
