#!/usr/bin/env bash
# Drive dacli from an EMPTY repository to shipped code, unattended.
#
#   ./scripts/selfhost-fixture.sh [workdir]
#
# This is the human-runnable twin of TestE2EFixtureRepoGoesFromEmptyToShipped
# (internal/cli/e2e_fixture_test.go), which runs the same arc in CI. Both exist
# on purpose: the test proves the arc on every push, and this script lets you
# WATCH it, with a real workspace left on disk to poke at afterwards.
#
# The "agent" is a shell script, so the run is offline and deterministic. What
# is being proven is dacli's coordination — a real model would only add
# variance to a question that is not about the model.
set -euo pipefail

DIR="${1:-$(mktemp -d)/fixture}"
mkdir -p "$DIR" && cd "$DIR"
echo "▸ fixture workspace: $DIR"

git init -q && git checkout -q -b main
git config user.email fixture@example.invalid
git config user.name  fixture
echo "seed" > README.md && git add -A && git commit -qm "init"

# --- plan ---------------------------------------------------------------
dacli init --name fixture >/dev/null
dacli project add Adder --slug adder \
  --goal "Provide an add function with a test that proves it" >/dev/null
dacli task add "Implement Add and cover it with a test" --project adder \
  --accept "adder.go defines Add" \
  --accept "the package tests pass" >/dev/null
echo "▸ planned: $(dacli task list --project adder)"

# --- the agent ----------------------------------------------------------
# Writes the code, commits through dacli (so the work is attributed), then
# reports with `task done` — the protocol verb, which files a PROPOSAL. A
# spawned agent does not close its own task; the owner applies it.
cat > agent.sh <<'AGENT'
set -e
cat > adder.go <<'GO'
package adder

// Add returns the sum of a and b.
func Add(a, b int) int { return a + b }
GO
cat > adder_test.go <<'GO'
package adder

import "testing"

func TestAdd(t *testing.T) {
	if Add(2, 2) != 4 {
		t.Fatal("Add is wrong")
	}
}
GO
printf 'module adder\n\ngo 1.22\n' > go.mod
git add -A
dacli commit "implement Add with a test" --task 001 --no-add
dacli task done 001
AGENT

dacli runtime add worker --binary sh --mode stdin --arg "$DIR/agent.sh" --env PATH >/dev/null

# --worktree gives the agent its own task branch (dacli commit refuses to
# commit on trunk); --claim is the scope guard that refuses files it did not
# declare.
dacli spawn --task 001 --runtime worker --grant rw --worktree \
  --claim adder.go,adder_test.go,go.mod

# --- land ---------------------------------------------------------------
# The TOOL closes its own loop. Nothing below reconciles by hand.
dacli sync
dacli ship --project adder --into main

# --- assert the OUTCOME -------------------------------------------------
fail() { echo "✗ $1"; exit 1; }

git show main:adder.go | grep -q 'func Add'   || fail "Add never reached trunk"
(cd "$DIR" && go test ./... >/dev/null 2>&1)  || fail "the shipped code does not pass its own tests"
dacli task list --project adder | grep -q done || fail "the task did not close"
dacli task list --project adder | grep -q '\[2/2\]' || fail "closed with unchecked acceptance boxes"
# The one signal that catches every "reported success and did nothing" at once.
git log --oneline main | grep -q 'implement Add' || fail "trunk never advanced"

echo
echo "✓ empty repository → shipped, unattended"
dacli task list --project adder
git log --oneline main | head -3
echo
echo "workspace kept at: $DIR"
