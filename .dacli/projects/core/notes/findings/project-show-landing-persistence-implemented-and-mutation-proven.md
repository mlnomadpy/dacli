---
id: f-project-show-landing-persistence-implemented-and-mutation-proven
kind: note
note_kind: finding
created: 2026-08-26T14:38:39Z
created_by: a-fixer-1x0gq5
about: "[[t-01M0F8DMCN93FCDE59FSEDTJB3]]"
severity: minor
---
# Project show landing persistence implemented and mutation-proven
internal/features/planning/planning.go persists and reloads landing flags before output; internal/cli/project_landing_test.go runs the real binary, reads project.md, reloads store policy, and verifies ship/integrate see PR policy. Mutation removing store.SaveProject made go test ./internal/cli -run '^TestProjectShowLandingFlagsPersistBeforeRendering$' fail at project_landing_test.go:26. gofmt -l ., GOCACHE=/private/tmp/dacli-go-cache go vet ./..., and GOCACHE=/private/tmp/dacli-go-cache go test ./... passed; golangci-lint unavailable is filed separately.
