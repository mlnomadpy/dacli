---
id: f-task-443-acceptance-criteria-are-implementation-verified
kind: note
note_kind: finding
created: 2026-08-14T01:54:10Z
created_by: a-maintainer-204p4w
about: "[[t-01KZYQ5EF1YQ05NRA8GW3N9PQM]]"
severity: moderate
---
# Task 443 acceptance criteria are implementation-verified
Generated mutating commands now use t.ID; TestPromptSuffixUsesStableTaskIDForMutatingCommands and TestAcceptStableIDResolvesWhenNumericRefIsAmbiguous cover two projects sharing sequence 001, ambiguity non-mutation, and ULID acceptance. Mutation proof: restoring numeric Ref fails with 'generated instructions missing stable command task check t-...'. Full go test ./..., go build ./..., go vet ./..., gofmt -l ., and golangci-lint run passed with writable sandbox caches. task check was policy-refused because a-root owns the task, so the owner must reconcile the boxes.
