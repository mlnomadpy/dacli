---
id: f-configured-landing-base-governs-acceptance-and-pr-base-verification
kind: note
note_kind: finding
created: 2026-08-26T23:32:48Z
created_by: a-root
about: "[[504]]"
severity: major
---
# Configured landing base governs acceptance and PR-base verification
Task 504 fixes internal/features/acceptance: accept resolves --into then the task project's configured landing base then repository trunk; merged PR evidence is trusted only when baseRefName matches that target; unknown remote/ancestry evidence refuses with the selected base named. Mutation replacing configured dev with repository master makes TestAcceptUsesConfiguredLandingBaseForConfirmedMerge fail with NOT in master. Restored tree passes gofmt, go vet, golangci-lint (isolated cache), and go test ./....
