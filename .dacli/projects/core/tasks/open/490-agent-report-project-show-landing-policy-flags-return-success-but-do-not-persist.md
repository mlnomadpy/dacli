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
- [ ] `dacli project show <slug> --landing-mode pr --landing-base <branch>` persists both values in the project record before displaying the effective policy.
- [ ] A later `project show`, `ship`, `integrate`, and loop-policy resolution observe the persisted landing mode and base without requiring another override.
- [ ] Updating only one landing flag preserves the other configured value and unrelated project frontmatter/body content.
- [ ] Invalid modes, blank/unsafe bases, duplicate conflicting values, and failed writes return the documented non-zero exit without partially changing the project file.
- [ ] Human and JSON output distinguish configured and effective policy after the write, and shared CLI/MCP help retains the exact mutating signature.
- [ ] Tests read the project file and reload effective policy after the public command; mutation proof shows removing the persistence step makes the focused regression fail.
- [ ] `gofmt -l .`, `go vet ./...`, pinned `golangci-lint run`, and `go test ./...` pass.
## Log
