#!/usr/bin/env bash
# Enforce coverage floors: one global, and one per SAFETY-CRITICAL package.
#
#   ./scripts/coverage-floor.sh coverage.out
#
# A floor is a collapse detector, not a target. It sits just under today's
# number so a package losing its suite fails the build, and it is raised
# deliberately — never lowered to make a build pass.
#
# Per-package floors exist because a single aggregate hides exactly the failure
# that matters. internal/gates could fall from 21.7% to zero and the global
# number would barely move, while the thing that fell is the surface deciding
# whether a stage may advance (dacli 357).
set -euo pipefail
PROFILE="${1:-coverage.out}"
[ -f "$PROFILE" ] || { echo "✗ no coverage profile at $PROFILE"; exit 1; }

# package<TAB>floor. Raise deliberately; never lower to go green.
read -r -d '' FLOORS <<'EOF' || true
TOTAL	60
internal/clikit	66
internal/agentid	88
internal/workspace	50
internal/store	48
internal/gates	90
internal/mcp	52
EOF

fails=0
pct_of() {
  if [ "$1" = "TOTAL" ]; then
    go tool cover -func="$PROFILE" | awk '/^total:/ {gsub("%","",$3); print $3}'
  else
    # Straight from the PROFILE, weighted by statements, which is what
    # `go test -cover` reports. An unweighted mean of per-function percentages
    # is a different number — it read clikit as 55.5% where the real figure is
    # 68.8%, and a floor checker that computes a confidently wrong number
    # fails builds for no reason.
    #
    # Profile lines are: file.go:l.c,l.c numStatements count
    awk -v pkg="/$1/" '
      $0 ~ pkg {
        n = $(NF-1); c = $NF
        total += n
        if (c+0 > 0) covered += n
      }
      END { if (total) printf "%.1f", 100*covered/total; else print "NONE" }
    ' "$PROFILE"
  fi
}

while IFS=$'\t' read -r pkg floor; do
  [ -n "$pkg" ] || continue
  got=$(pct_of "$pkg")
  if [ "$got" = "NONE" ]; then
    echo "✗ $pkg: no statements in the profile — did the package move?"
    fails=$((fails+1)); continue
  fi
  if awk -v g="$got" -v f="$floor" 'BEGIN { exit !(g+0 < f+0) }'; then
    # Name WHICH package fell and by how much. One aggregate number tells a
    # reader that something broke and nothing about where.
    printf '✗ %-24s %5s%% is below its %s%% floor (short by %.1f)\n' \
      "$pkg" "$got" "$floor" "$(awk -v g="$got" -v f="$floor" 'BEGIN{printf "%.1f", f-g}')"
    fails=$((fails+1))
  else
    printf '✓ %-24s %5s%% (floor %s%%)\n' "$pkg" "$got" "$floor"
  fi
done <<< "$FLOORS"

[ "$fails" -eq 0 ] || { echo; echo "$fails floor(s) breached — raise coverage, do not lower the floor"; exit 1; }
