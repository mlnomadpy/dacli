---
id: f-driver-git-now-routes-through-gitx-run-runnetwork-with-a-proven-bounded-timeout
kind: note
note_kind: finding
created: 2026-07-23T13:55:05Z
created_by: a-k51f2ddh5e
about: [[105]]
severity: minor
---
# driver.git now routes through gitx.Run/RunNetwork with a proven-bounded timeout test
internal/features/orchestration/orchestration.go: driver.git() (was a bare exec.Command with no timeout) now calls gitx.Run(d.w.Root, args...) — local leash (gitx.LocalTimeout, 30s default). trunkMarker()'s 'git fetch -q origin <b>' now calls gitx.RunNetwork (gitx.NetworkTimeout, 120s default) instead of d.git, so a hung fetch times out and falls through to the existing local rev-list fallback instead of blocking. gitx.go: localTimeout/networkTimeout consts became exported vars LocalTimeout/NetworkTimeout so a test can shrink them; added gitx.RunNetwork (Push now calls it instead of duplicating runWithTimeout). New test internal/features/orchestration/driver_test.go:TestDriverGitAbortsOnHungSubprocess injects a fake 'git' script on PATH that execs sleep 5 (must exec, not fork, or SIGKILL to the shell leaves the sleep holding the output pipe open and the test waits the full 5s anyway — this bit me during development), shrinks gitx.LocalTimeout to 200ms, and asserts d.git returns an error within 2s. go build ./... clean; go test -exec 'env -u DACLI_AGENT' ./internal/... all green.
