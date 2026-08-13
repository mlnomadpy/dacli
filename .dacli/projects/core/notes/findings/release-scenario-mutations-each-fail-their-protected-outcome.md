---
id: f-release-scenario-mutations-each-fail-their-protected-outcome
kind: note
note_kind: finding
created: 2026-08-13T20:18:42Z
created_by: a-codex-maintainer-hh2s7h
about: "[[434]]"
severity: major
---
# Release scenario mutations each fail their protected outcome
go test -v ./internal/scenarios passes all five real-binary fixtures and logs a distinct caught mutation for feature work, regression repair, dependency failure, conflicting edits, and malicious instructions; harness and assertions are in internal/scenarios/scenarios_test.go:45-211.
