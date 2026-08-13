---
id: f-release-readiness-suite-now-catches-one-mutation-per-named-scenario
kind: note
note_kind: finding
created: 2026-08-13T20:00:14Z
created_by: a-codex-maintainer-grz3zz
about: "[[434]]"
severity: major
---
# Release-readiness suite now catches one mutation per named scenario
internal/scenarios/scenarios_test.go runs five real-binary offline fixtures; TestScenarioAssertionMutations observed failures for omitted integration, incorrect regression repair, missing dependency edge, non-conflicting edit, and trusted malicious origin. Command: GOCACHE=/private/tmp/dacli-434-gocache go test -v ./internal/scenarios.
