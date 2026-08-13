---
id: f-wait-finalized-two-codex-runs-while-their-transcripts-kept-advancing
kind: note
note_kind: finding
created: 2026-08-12T20:18:25Z
created_by: a-codex-maintainer-w6vv23
about: "[[400]]"
severity: major
---
# Wait finalized two Codex runs while their transcripts kept advancing
Runs 01KZVSF64JQF4X74N6V8AZ99G5 and 01KZVSFBXRH8KY06CCA2VVV6ST were stamped no visible result at elapsed 12s and 7s respectively, but their transcript.log mtimes and terminal turn.completed records show continued work until 20:13:47Z and 20:15:54Z. internal/features/execution/execution.go:2668 finalizes solely when procmon.ReconcileRun rejects the recorded guardian; liveAgents at execution.go:2531 uses the same probe, while runs list only displays outcome.md.
