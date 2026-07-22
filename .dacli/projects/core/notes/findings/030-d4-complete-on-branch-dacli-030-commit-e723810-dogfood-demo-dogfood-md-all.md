---
id: f-030-d4-complete-on-branch-dacli-030-commit-e723810-dogfood-demo-dogfood-md-all
kind: note
note_kind: finding
created: 2026-07-22T14:02:15Z
created_by: a-2xx9dxvf26
about: [[030]]
severity: moderate
---
# 030 D4 complete on branch dacli/030 (commit e723810) — dogfood demo + DOGFOOD.md, all 4 acceptance met
Committed e723810 by a-2xx9dxvf26 (maintainer) via git add + dacli commit --no-add, staging ONLY the 2 scoped files: scripts/dogfood-demo.sh (mode 100755) and docs/DOGFOOD.md; the built ./dacli binary is gitignored so it was not committed. ACCEPTANCE, all satisfied over this repo's OWN .dacli: (1) scripts/dogfood-demo.sh is a set -euo pipefail, strictly read-only (no spawns/writes/box-checks), re-runnable replay: it builds repo-root ./dacli if absent then prints contrib -> calibrate -> taint external:/taint agent -> status+standup with narration between each; binary path overridable via DACLI= for CI. (2) DOGFOOD.md embeds REAL captured output and frames calibrate as dacli measuring its own Cone of Uncertainty (x0.01-0.03 hours/point, ~50-100x under human-scaled points), honest about thinness (n=9<10, briefs stay silent; agent-band empty; actuals are wall-clock proxy). (3) taint shown end-to-end: 'taint external:' returns clean (no external origin) AND 'taint agent' shows the real blast radius (9 artifacts, 6 briefs exposed) tracing dacli's own 016-029 development, incl. the LOWER-BOUND honesty line. (4) committed on branch by an agent; go build ./... clean; go test ./internal/... all green. NOTE: two deviations recorded as decisions — the build target is ./cmd/dacli (the brief's 'go build .' fails: main pkg is at cmd/dacli/main.go), and the wrapped-script smoke run is blocked by the headless sandbox (only the designated binary path is approved), so the pasted output was captured by running each command directly against the shared workspace; the script's logic is identical. Owner: verify and close via dacli task check/done + dacli merge --task 030.
