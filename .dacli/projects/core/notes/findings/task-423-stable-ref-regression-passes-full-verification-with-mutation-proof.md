---
id: f-task-423-stable-ref-regression-passes-full-verification-with-mutation-proof
kind: note
note_kind: finding
created: 2026-08-13T15:56:41Z
created_by: a-fixer-f6typj
about: "[[423]]"
severity: major
---
# Task 423 stable-ref regression passes full verification with mutation proof
TestDriverUsesStableTaskIDsAcrossProjects creates duplicate 001/002 sequences in projects p and q and verifies implementation and review spawns resolve through stable IDs. Red mutation: orchestration.go build ref reverted to fmt.Sprintf("%03d", t.Seq) failed driver_test.go with build spawn task ref = "001", want stable ID. Green: gofmt -l . empty; go vet ./... passed; golangci-lint v2.12.2 reported 0 issues; go test ./... passed.
