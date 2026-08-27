---
id: 01M12Q5JF4G89GSHAVBJG09AQ8
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-27T22:59:05Z
created_by: a-root
about: "[[t-01M1068MEG379NZ2SE5EH6DYZC]]"
origin: agent
applied: true
checksum: sha256:6f395f20215321a4a313a5c8606301f68db9165457872ba824dd1c4de577a568
---
bb892018 Fix runtime build capability preflight

Mutation proof: removing Gradle capability inference made TestGradleRequiresProviderNeutralCoordinationSocketCapability and TestGradleProfileFailsClosedBeforeWorkerSpend fail.

Verified: gofmt -l .; env GOCACHE=/tmp/dacli-509-go-cache go vet ./...; env GOCACHE=/tmp/dacli-509-go-cache go test ./... . Local golangci-lint was unavailable; CI remains required.
role: root
