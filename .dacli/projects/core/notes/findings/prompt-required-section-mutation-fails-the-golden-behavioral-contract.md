---
id: f-prompt-required-section-mutation-fails-the-golden-behavioral-contract
kind: note
note_kind: finding
created: 2026-08-19T14:35:16Z
created_by: a-maintainer-anf4d3
about: "[[t-01M0CX03CC0N95X4M5ESKRP2E6]]"
severity: major
---
# Prompt required-section mutation fails the golden behavioral contract
Mutation evidence: removing the contract:honest-evidence marker from internal/prompts/tpl/autonomous_delivery.md made GOCACHE=/tmp/dacli-go-cache-476 go test ./internal/prompts -run '^TestAutonomousContractGoldenAndRequiredBehavior$' fail at prompts_test.go:116 with missing required section honest-evidence; restoring it returns green.
