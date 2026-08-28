---
id: t-01M13X19WKEC3MXWMS475GCSR2
kind: task
created: 2026-08-28T10:00:51Z
created_by: a-root
owner: a-root
priority: must
estimate: "{optimistic: 2, probable: 4, pessimistic: 7}"
github:
  issue: 851
  repo: mlnomadpy/dacli
---
# Optimize CI cost by removing routine full-suite macOS duplication
## So that
autonomous dacli loops preserve meaningful cross-platform evidence without exhausting hosted-runner budgets
## Acceptance
- [ ] The pull-request CI required check runs the full Go/frontend verification on one Linux runner rather than a Linux/macOS full-suite matrix.
- [ ] Merges do not immediately duplicate the already-tested pull-request pipeline on main; manual recovery remains available through workflow_dispatch.
- [ ] Native macOS validation is retained as a narrow platform-sensitive or release gate, while darwin amd64 and arm64 cross-compilation remains on Linux.
- [ ] Workflow contract tests fail if routine PR CI reintroduces macos-latest or push-to-main duplication.
- [ ] The issue documents expected cost reduction, the public-repository billing caveat, and the residual risk of moving native macOS coverage out of every PR.
## Log
- 2026-08-28T10:02:33Z claimed by a-fixer-fv8pny
