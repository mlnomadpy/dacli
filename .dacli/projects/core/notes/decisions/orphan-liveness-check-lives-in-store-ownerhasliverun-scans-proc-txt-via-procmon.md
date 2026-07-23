---
id: d-orphan-liveness-check-lives-in-store-ownerhasliverun-scans-proc-txt-via-procmon
kind: note
note_kind: decision
created: 2026-07-23T12:06:35Z
created_by: a-vrppnfvawm
about: [[095]]
---
# orphan liveness check lives in store (OwnerHasLiveRun, scans proc.txt via procmon.AliveRecord), not inline in insight.go
## Chose
orphan liveness check lives in store (OwnerHasLiveRun, scans proc.txt via procmon.AliveRecord), not inline in insight.go
## Rejected
scan w.RunsDir() directly inside cmdDoctor, importing procmon from the insight slice (legal per arch_test, vcs.go already does it)
## Because
store already hosts DuplicateTaskFiles/LogHasStamp as doctor-consumed helpers over task/run data, and store already imports workspace; keeping the proc.txt scan there matches that precedent and keeps insight.go free of procmon plumbing. procmon has no internal/ imports so store->procmon adds no import cycle.
