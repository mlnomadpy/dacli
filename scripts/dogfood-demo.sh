#!/usr/bin/env bash
#
# dogfood-demo.sh — dacli, replayed from its OWN .dacli record.
#
# The differentiator dacli sells is "attributed log -> learn from it." The only
# honest proof is the tool having learned from itself. This session's agents
# (tasks 016-029: resource-monitor, hot-path audit, async spawn, worktree
# shadowing fix, then D1-D3) developed dacli, and that history is real,
# self-hosted data sitting in this repo's .dacli/. This script replays it.
#
# It is strictly READ-ONLY: no spawns, no writes, no box-checking. Re-runnable
# and idempotent. Every readout below is live output over this workspace's log.
#
set -euo pipefail

# Resolve the repo root from this script's location, so the demo works whether
# it is run from the repo root, from scripts/, or from a linked worktree (dacli
# redirects a worktree cwd to the shared .dacli via git-common-dir).
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

# Use the repo-root ./dacli; build it if absent. The main package is at
# ./cmd/dacli (NOT the repo root), so target that explicitly. Override with
# DACLI=/path/to/dacli to point at a prebuilt binary (e.g. in CI).
DACLI="${DACLI:-$ROOT/dacli}"
if [[ ! -x "$DACLI" ]]; then
  echo "building $DACLI ..." >&2
  go build -o "$DACLI" ./cmd/dacli
fi

hr()  { printf '\n%s\n' "────────────────────────────────────────────────────────────"; }
say() { printf '\n\033[1m%s\033[0m\n' "$*"; }
run() { printf '\n$ dacli %s\n\n' "$*"; "$DACLI" "$@"; }

# ─────────────────────────────────────────────────────────────────────────────
say "dacli dogfood demo — the tool's own development, replayed from .dacli/"
echo "Everything below is live output over this repo's self-hosted log."
echo "Read-only. No agent is spawned; nothing is written."

# 1. CONTRIB — who authored the code, provably.
hr
say "1. contrib — per-role / per-agent contribution rollup"
cat <<'EOF'
Every commit in this workspace was authored by a spawned agent, stamped with its
agent id and role by `dacli commit`. This rollup is derived from that attributed
log: the agents wrote the code, and the record proves it — not a claim, a query.
EOF
run contrib

# 2. CALIBRATE — the empirical Cone of Uncertainty, self-measured.
hr
say "2. calibrate — dacli measured its own Cone of Uncertainty"
cat <<'EOF'
dacli ported the human software-estimation apparatus (PERT, the Cone of
Uncertainty, velocity — McConnell 2006). Then it estimated its OWN work and
measured actuals against it. The finding that named this whole D-series: agent
wall-time runs ~0.01-0.03 hours/point — roughly 50-100x under human-scaled
points. The multiplier is not error; it is the unit conversion between
human-calibrated points and agent wall-clock. This is the one Cone no competitor
can print: none has the attributed log to measure its own.
EOF
run calibrate

# 3. TAINT — real blast-radius query over the workspace's own provenance.
hr
say "3. taint — blast-radius over the log's own origins"
cat <<'EOF'
taint answers "if this source were poisoned, which briefs are downstream?" First
the security-relevant query: is anything in this workspace derived from an
EXTERNAL origin? If the query returns clean, that is the honest, meaningful
answer — the blast radius of external input is empty.
EOF
run taint "external:"
cat <<'EOF'

Now the same query over the origin that IS present — every artifact here is
self-reported origin=agent. This traces dacli's own development end-to-end: the
events, the tasks they completed, and the briefs each one exposed. That is the
"attributed log" made queryable. (taint is honest that it is a LOWER BOUND:
only labeled provenance is traced.)
EOF
run taint "agent"

# 4. STATUS + STANDUP — the tree state and per-agent roll-up.
hr
say "4. status + standup — tree state and per-agent roll-up, from the log"
cat <<'EOF'
status is the project tree derived from the task log; standup is the per-agent
activity roll-up. Together they show the shape of the work: what is done, what is
open, and which agent did what — the same log, viewed two ways.
EOF
run status
run standup

hr
say "done — every readout above came from this repo's own .dacli/ record."
echo "See docs/DOGFOOD.md for the narrative and what each readout proves."
