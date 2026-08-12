---
id: 01KZV42Q7TPVT64PKZH331CBRJ
kind: event
event_kind: commit
created: 2026-08-12T13:55:06Z
created_by: a-root
about: "[[t-01KZV16F64YVHKS00NQ9CZ5Q0C]]"
origin: agent
applied: true
---
6b417aa 369: reject reused groups after recorded leader death

Mutation red: TestRunStillLiveRejectsTask285ResidualDeadLeaderWithReusedGroup failed because an unrelated live group resurrected the dead run.

Verified: gofmt -l .; go vet ./...; golangci-lint run; go test -race ./...
role: root
