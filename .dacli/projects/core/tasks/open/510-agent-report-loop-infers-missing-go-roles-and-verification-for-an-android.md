---
id: t-01M1068MNDZ72H5R35YRZ9MASK
kind: task
created: 2026-08-26T23:25:11Z
created_by: a-root
owner: a-root
github:
  issue: 798
  repo: mlnomadpy/dacli
estimate: "{optimistic: 5, probable: 8, pessimistic: 13}"
---
# [agent-report] loop infers missing Go roles and verification for an Android project
## Context
Adopted from GitHub issue #798.

For an Android Kotlin Compose project, loop dry-run selected fixer and go-auditor roles that do not exist, while start --profile loop proposed gofmt, go vet, golangci-lint, and go test. The project has explicit android-lead and qa-reviewer roles and Gradle verification. Stack detection/profile resolution should prefer the declared Android project roles and commands or refuse with a clear configuration remedy.

---
_Reported via `dacli report`._
- dacli: dev
- platform: darwin/arm64
- workspace and run transcript withheld (public upstream) — re-run with --disclose to include them

## Acceptance
- [ ] An Android/Kotlin/Compose fixture with declared `android-lead` and `qa-reviewer` roles routes implementation and review to those roles.
- [ ] The resolved profile uses the project's declared Gradle verification commands and contains no Go-only defaults.
- [ ] Missing declared roles or verification commands fail closed with exact `role add` or profile-configuration guidance before spawning.
- [ ] Cost-aware model selection still chooses the cheapest capable configured model within the selected project roles.
- [ ] Dry-run and execution share the same resolved roles, commands, landing policy, and bounded-loop budget.
- [ ] Mutation evidence and the full repository verification gates pass.
## Log
- 2026-08-27T23:13:03Z claimed by a-maintainer-mzxznz
