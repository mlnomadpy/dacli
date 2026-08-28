#!/usr/bin/env bash
set -euo pipefail

# Executable clean-fixture proof for the bounded task journey. This deliberately
# uses local landing so it needs no GitHub credentials; the PR/CI substitution
# is documented in WALKTHROUGH.md and keeps the same acceptance boundary.
DACLI_BIN=${DACLI_BIN:-dacli}
FIXTURE=${1:-"$(mktemp -d)"}

mkdir -p "$FIXTURE"
cd "$FIXTURE"
git init -q -b main
git config user.name fixture
git config user.email fixture@example.invalid
printf '# fixture\n' > README.md
git add README.md
git commit -qm 'initial fixture'

"$DACLI_BIN" init --name fixture --roster agents
"$DACLI_BIN" project add 'Fixture product' --slug fixture --landing-mode local
"$DACLI_BIN" task add 'Add verified result' --project fixture \
  --accept 'result.txt exists after landing'
"$DACLI_BIN" task claim 001

git switch -qc dacli/001-add-verified-result
printf 'verified\n' > result.txt
"$DACLI_BIN" commit 'feat: add verified result' --task 001
"$DACLI_BIN" task check 001 --all --verify 'test -f result.txt'
"$DACLI_BIN" task done 001

git switch -q main
"$DACLI_BIN" integrate --tasks 001 --landing-mode local --into main
test -f result.txt
"$DACLI_BIN" task list --project fixture --status done | grep -q '001'
printf 'walkthrough complete: task 001 is verified, done, and merged into main\n'
