---
id: f-repeated-claim-loss-occurs-in-shared-launch-resolution
kind: note
note_kind: finding
created: 2026-08-27T09:46:07Z
created_by: a-root
about: "[[501]]"
severity: major
---
# Repeated claim loss occurs in shared launch resolution
Reproduced on the pre-fix code: TestSpawnAdviseAccumulatesRepeatedAndCommaSeparatedClaims omitted claims from preview, and TestResolveLaunchAccumulatesRepeatedAndCommaSeparatedClaims resolved only [internal/c] from --claim 'internal/a, internal/b' --claim internal/c. Fix accumulates Flags.All values in invocation order; focused execution and VCS claim-gate tests pass. Full gates: gofmt -l ., go vet ./..., golangci-lint run, go test ./....
