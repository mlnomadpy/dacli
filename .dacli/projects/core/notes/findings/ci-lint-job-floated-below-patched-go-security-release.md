---
id: f-ci-lint-job-floated-below-patched-go-security-release
kind: note
note_kind: finding
created: 2026-08-13T22:44:50Z
created_by: a-fixer-8skqtd
about: "[[438]]"
severity: major
---
# CI lint job floated below patched Go security release
.github/workflows/ci.yml:169 selected the floating Go 1.25 line for both golangci-lint and govulncheck; .github/workflows/contract_test.go now enforces Go 1.25.13 or newer within that line. Mutation evidence: the new test failed with 'lint job must use Go 1.25.13 or newer within the 1.25 line' before the workflow fix.
