---
id: t-01KZPFB35Y2PEQM7E1BJV89ESE
kind: task
created: 2026-08-10T18:35:43Z
created_by: a-root
owner: a-root
priority: should
estimate: "{optimistic: 2, probable: 4, pessimistic: 8}"
---
# Stage the deferred linters: gosec with a curated rule list, and errorlint against the exit-code contract
## Acceptance
- [x] gosec runs with G104/G301/G302/G306 excluded and any remaining finding is either fixed or refused with a stated reason
- [x] errorlint's 39 sites are converted to %w where wrapping is correct, verified against ExitCode's errors.As mapping
- [x] a test asserts an exit-3 refusal wrapped through fmt.Errorf still maps to exit 3
- [x] both linters are enabled in .golangci.yml, or their omission is documented with the measurement behind it
## Log
- 2026-08-10T19:12:34Z claimed by a-fixer-k0d8zr
- 2026-08-10T19:35:05Z accepted by a-root
- 2026-08-10T19:35:05Z verified by `go test ./internal/clikit/ ./internal/store/` (exit 0)
- 2026-08-10T19:35:05Z deliverable: dacli/338-stage-the-deferred-linters-gosec-with-a-curated-rule-list-and-errorlint-against exists but is NOT in trunk — closed anyway
- 2026-08-10T19:35:05Z completed by a-root
