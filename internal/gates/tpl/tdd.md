---
name: tdd
summary: test-driven delivery — a stage cannot close until the suite is green and covered
cost: "four gates, two of which run your real build; use when correctness is the product"
---
# tdd

Test-driven development, enforced rather than encouraged. Every other template
gates on documents; this one gates on the software. The `command:` and
`coverage:` predicates run your real build, so a stage cannot be advanced by
writing prose about tests that do not exist.

Set the three commands below to your stack's real ones before you rely on this
template — `dacli template add tdd` vendors an editable copy into the workspace.
The defaults assume Go.

## stage: design
cone: definition
phase: planning
allow: researcher, planner, reviewer
- project_sections: Goal | Success criteria
- tasks: all_have_acceptance

## stage: red
cone: approach
phase: implementation
allow: implementer, reviewer
- artifact: go.mod
- command: go build ./...

## stage: green
cone: design
phase: implementation
allow: implementer, reviewer
- command: go build ./... && go vet ./...
- command: go test ./...
- coverage: 70 go test -coverprofile=/tmp/dacli-cov.out ./... >/dev/null && go tool cover -func=/tmp/dacli-cov.out | tail -1

## stage: ship
cone: design
phase: release
allow: implementer, reviewer
- tasks: musts_done
- command: go test ./...
- retro: required
