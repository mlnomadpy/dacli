---
id: t-01KY7VMYD97RTRRZFVZKZJ4ZV2
kind: task
created: 2026-07-23T16:06:30Z
created_by: a-xrcxmhwz96
owner: a-root
priority: should
---
# Fix runtimefiles inline-list encoding: quote list elements containing commas so runtime args round-trip losslessly
## So that
the flagship claude-code runtime's read-only sandbox argv (SandboxRO --allowedTools Read,Grep,Glob,LS,Bash(dacli:*), execution.go:62) stops being silently re-split into 6 argv tokens on load; cc.md:10 had to be hand-corrected after run 01KY2K8N4C and runtime add will re-mangle it
## Acceptance
- [x] setInline in internal/store/runtimefiles.go quotes any list element that contains a comma (or other splitTop-significant char) so a value like Read,Grep,Glob,LS,Bash(dacli:*) is written as a single quoted token
- [x] a new round-trip test in internal/store proves CreateRuntime then LoadRuntime preserves SandboxRO/Args/Env elements containing literal commas (e.g. the claude-code preset's --allowedTools value) as the SAME element count and values
- [x] go build ./... and go test ./internal/... are green
## Log
- 2026-07-23T18:38:59Z claimed by a-v05q6gkkqh
- 2026-07-23T18:41:33Z adopted by a-root (owner a-xrcxmhwz96 orphaned)
- 2026-07-23T18:41:33Z accepted by a-root
- 2026-07-23T18:41:33Z completed by a-root
