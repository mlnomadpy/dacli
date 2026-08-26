---
id: t-01M0F8DMCN93FCDE59FSEDTJB3
kind: task
created: 2026-08-20T09:35:47Z
created_by: a-root
owner: a-root
github:
  issue: 762
  repo: mlnomadpy/dacli
estimate: "{optimistic: 3, probable: 5, pessimistic: 8}"
---
# [agent-report] project show landing policy flags return success but do not persist
## Context
Adopted from GitHub issue #762.

On a linked project, running 'dacli project show bashnota --landing-mode pr --landing-base master' exits 0 and prints the project, but .dacli/projects/bashnota/project.md remains unchanged with no landing.mode or landing.base frontmatter. Repeating the command has the same result. Shipped docs state this form persists PR landing policy. Expected: frontmatter gains landing.mode: pr and landing.base: master, or command fails. Actual: silent no-op can leave later integration on legacy local landing. Please add persistence and regression coverage, including reading the file/effective policy after the command.

---
_Reported via `dacli report`._
- dacli: dev
- platform: darwin/arm64
- workspace and run transcript withheld (public upstream) — re-run with --disclose to include them

## Acceptance
- [x] `dacli project show <slug> --landing-mode pr --landing-base <branch>` persists both values in the project record before displaying the effective policy.
- [x] A later `project show`, `ship`, `integrate`, and loop-policy resolution observe the persisted landing mode and base without requiring another override.
- [x] Updating only one landing flag preserves the other configured value and unrelated project frontmatter/body content.
- [x] Invalid modes, blank/unsafe bases, duplicate conflicting values, and failed writes return the documented non-zero exit without partially changing the project file.
- [x] Human and JSON output distinguish configured and effective policy after the write, and shared CLI/MCP help retains the exact mutating signature.
- [x] Tests read the project file and reload effective policy after the public command; mutation proof shows removing the persistence step makes the focused regression fail.
- [x] `gofmt -l .`, `go vet ./...`, pinned `golangci-lint run`, and `go test ./...` pass.
## Log
- 2026-08-26T14:22:22Z claimed by a-fixer-eqe3tq
- 2026-08-26T14:32:43Z claimed by a-fixer-1x0gq5
- 2026-08-26T14:43:04Z claimed by a-adversarial-reviewer-a1h3ab
- 2026-08-26T14:46:50Z a-root: PR opened: https://github.com/mlnomadpy/dacli/pull/795 (event 01M0Z8C0NPWVQ8AJFVF6DQDKE9)
- 2026-08-26T15:13:51Z claimed by a-adversarial-reviewer-pdq7xr
- 2026-08-26T15:18:32Z claimed by a-adversarial-reviewer-dftw7y
- 2026-08-26T15:25:58Z claimed by a-adversarial-reviewer-5zhnqk
- 2026-08-26T22:31:43Z accepted by a-root
- 2026-08-26T22:31:44Z verified by `go test ./...` (exit 0) in branch main at c0f01cb — proves that tree builds, not that the work is in trunk
- 2026-08-26T22:31:44Z deliverable: dacli/490-agent-report-project-show-landing-policy-flags-return-success-but-do-not-persist is merged into main
- 2026-08-26T22:31:44Z completed by a-root
## Verification Evidence
{"command":"go test ./...","exit_code":0,"duration_ms":73609,"artifact_hash":"sha256:aa7b36b34f13abe1ae023649eda9938bdd1c0304f19c0824ffa57568a1acfd01","verifier":"a-root","branch":"main","commit_sha":"c0f01cb81978cd3eb29a43ef8c5101324a179570"}
