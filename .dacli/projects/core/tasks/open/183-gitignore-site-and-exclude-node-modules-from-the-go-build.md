---
id: t-01KYHWFE5CVYF9FDVT9MP9SMZC
kind: task
created: 2026-07-27T13:33:22Z
created_by: a-root
owner: a-root
priority: could
estimate: "{optimistic: 2, probable: 4, pessimistic: 8}"
---
# Gitignore site and exclude node_modules from the Go build
## So that
generated output is not committable and go tooling does not compile vendored JS
## Acceptance
- [ ] site is gitignored
- [ ] go build test vet do not traverse node_modules
## Log
