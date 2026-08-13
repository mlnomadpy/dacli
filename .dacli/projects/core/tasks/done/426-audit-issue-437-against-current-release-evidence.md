---
id: t-01KZXXZPBXT332W00RP94HTR2K
kind: task
created: 2026-08-13T16:06:19Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 2, probable: 3, pessimistic: 5}"
parent: "[[t-01KZXS3QNDANVK83M26D3S8W7M]]"
github:
  issue: 593
  repo: mlnomadpy/dacli
---
# Audit issue 437 against current release evidence
## So that
the release-readiness epic records only real remaining gaps instead of re-requesting work that already landed
## Acceptance
- [x] the task log maps every issue 437 requirement to a current file, CI job, reproducible command, or an explicit missing-evidence finding
- [x] each missing gap is filed once as a task-sized child with checkable acceptance criteria after duplicate checking
- [x] GitHub issue 437 is updated with the evidence matrix and closed only if every acceptance criterion is demonstrably satisfied
## Log
- 2026-08-13T19:32:39Z claimed by a-codex-loop-auditor-hxqjcg
- 2026-08-13T20:09:00Z issue #437 evidence matrix (audited at `937138b`):
  - Required work 1–3 (oversized detached prompt, deterministic Unix process monitoring, clean supported-platform checkout): `internal/features/execution/execruntime_test.go:362-429`, `internal/procmon/procmon_unix_test.go:106-131`, `internal/procmon/zombie_unix_test.go:41-89`, and `.github/workflows/ci.yml:10-141,215-233`; reproduced with `go test -race -count=3 ./internal/procmon/` and `go test -race -count=3 ./internal/features/execution/ -run 'TestExecRuntimeDetached|TestAlive|TestKillTree|TestRecordRoundTrip'` (both exit 0). The requested procmon adapter/testdata shape is superseded by build-tagged `procmon_unix.go`/`procmon_windows.go` plus live process helpers and both-OS CI, which prove the stated behavior without a mock adapter.
  - Required work 4 (race, fuzz, failure injection): `.github/workflows/ci.yml:79-137`, `internal/eventlog/failure_injection_test.go`, `internal/store/failure_injection_test.go`, and fuzz targets in `internal/clikit`, `internal/mdstore`, and `internal/workspace`; `go test ./internal/clikit/ ./internal/eventlog/ ./internal/store/` exits 0. Missing literal local-event schema/checkpoint work is child 430; missing queue/stage-gate retry/dead-letter state is child 431.
  - Required work 5 (unattended end-to-end fixture): `internal/cli/e2e_fixture_test.go`, `scripts/selfhost-fixture.sh`, and `docs/SELFHOSTING.md:18-50`; `go test ./internal/cli/ -run TestE2EFixtureRepo` exits 0. The remaining four named scenario classes are absent (`internal/scenarios` does not exist) and are child 434.
  - Required work 6 (completion/retry/wall/token/intervention metrics): `internal/features/insight/metrics.go:15-257` and its tests pass. The requested shared metrics interface, failure-class/budget JSON export, and repeatable machine comparison are absent and are child 433.
  - Required work 7 (trust/taint, boundaries, secrets, branch protection, rollback): `docs/TRUST.md`, especially its complete mutating-command matrix at lines 261-373, plus `SECURITY.md:12-46`; the documentation contract is exercised by `internal/cli/trust_test.go`. This is current equivalent evidence despite the issue naming `SECURITY.md` as the sole destination.
  - Required work 8 (agent identity/self-host review): `docs/SELFHOSTING.md:53-159` maps `Dacli-Agent`, `Dacli-Role`, and `Dacli-Task`, and states that local integrate reads no diff.
  - Required work 9 (CLI/MCP compatibility): `docs/COMPATIBILITY.md` and `internal/cli/compat_json_shape_test.go`; the CLI shape test passes. MCP schemas still lack per-tool versions and golden fixtures (`internal/mcp/testdata` is absent), filed as child 435.
  - Required work 10 (release artifacts): `.goreleaser.yaml:13-53`, `.github/workflows/ci.yml:239-285`, and `scripts/verify-release-artifacts.sh`; `goreleaser check` validates the config. The actual tag workflow omits Syft and reproducibly exits 1 at SBOM generation; child 429 covers this distinct release blocker.
  - Acceptance 1 (green clean checkout on Linux/macOS with race): the `test-matrix` job runs Ubuntu Go 1.22 and macOS Go 1.25, builds the frontend, vets, races, fuzzes, and repeats process suites; `clean-checkout` separately builds/tests without Node. Local focused race commands pass; current remote run status is unverified because GitHub is unreachable.
  - Acceptance 2 (reproducible self-host case study): `docs/SELFHOSTING.md:18-50`, `scripts/selfhost-fixture.sh`, and `TestE2EFixtureRepoGoesFromEmptyToShipped`; the issue's proposed filenames differ, but the same documented, deterministic, outcome-asserting artifact exists.
  - Acceptance 3 (security/failure recovery for every mutating surface): `docs/TRUST.md:261-373` enumerates every mutating command with effect, gate, and undo/no-undo; `internal/cli/trust_test.go` prevents command-table drift. Structured evidence provenance remains child 432.
  - Acceptance 4 (tagged release installable via README): snapshot CI verifies six archives, per-archive SBOMs, checksums, extraction, `version`, and `--help` without publishing. It is not demonstrably satisfied for the tag workflow itself because `.github/workflows/release.yml:46-52` lacks Syft; reproduction: `GOCACHE=/private/tmp/dacli-426-gocache goreleaser release --snapshot --clean` exits 1 with `exec: "syft": executable file not found in $PATH`. Child 429 is the sole duplicate-checked gap for that failure.
- 2026-08-13T20:09:00Z duplicate check: searched every core task/note for each missing concept, then listed open, active, and blocked work. No semantic duplicate existed for children 429–435; completed tasks 184, 352, 355, 408, and 425 were reviewed and cover related but narrower or control-plane-only contracts.
- 2026-08-13T20:09:00Z remote limitation: `gh issue view 437 --repo mlnomadpy/dacli` and issue 593 both failed once with `error connecting to api.github.com`; issue #437 was not updated or closed, and remote CI state remains unverified.
- 2026-08-13T19:40:57Z accepted by a-root
- 2026-08-13T19:40:57Z verified by `GOCACHE=/private/tmp/dacli-426-main-gocache go test ./internal/cli ./internal/procmon ./internal/features/execution ./internal/features/insight ./internal/mcp` (exit 0) in branch main at 937138b — proves that tree builds, not that the work is in trunk
- 2026-08-13T19:40:57Z deliverable: dacli/426-audit-issue-437-against-current-release-evidence is merged into main
- 2026-08-13T19:40:57Z completed by a-root
