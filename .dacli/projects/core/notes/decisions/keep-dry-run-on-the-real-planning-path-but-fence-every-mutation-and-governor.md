---
id: d-keep-dry-run-on-the-real-planning-path-but-fence-every-mutation-and-governor
kind: note
note_kind: decision
created: 2026-08-12T15:44:24Z
created_by: a-codex-maintainer-3vy9w1
about: "[[370]]"
github:
  issue: 498
  repo: mlnomadpy/dacli
---
# Keep dry-run on the real planning path but fence every mutation and governor charge
## Chose
Keep dry-run on the real planning path but fence every mutation and governor charge
## Rejected
Build a separate preview implementation that duplicates the loop phases
## Because
The existing dryRunner already renders the exact commands; central saveState suppression plus dry-run guards around direct mutators preserves plan fidelity without a second orchestration path that can drift.
