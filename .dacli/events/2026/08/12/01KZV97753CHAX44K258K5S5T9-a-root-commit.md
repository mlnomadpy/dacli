---
id: 01KZV97753CHAX44K258K5S5T9
kind: event
event_kind: commit
created: 2026-08-12T15:24:56Z
created_by: a-root
about: "[[t-01KZV4SAVNZ1JXMJ61RRQCJYPY]]"
origin: agent
applied: true
---
886b6a8 375: keep an authenticated guardian alive for runtime descendants

Mutation red: TestRunStillLivePreservesTask177AfterLeaderExit failed with reconciliation lost a genuine helper after its recorded leader exited, while task-369 liveness and no-signal controls stayed green.

Verified: focused guardian/runtime/CLI tests; gofmt -l .; go vet ./...; golangci-lint run; go test -race ./...
role: root
