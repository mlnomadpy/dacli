---
id: f-release-scenario-suite-catches-one-outcome-mutation-per-named-fixture
kind: note
note_kind: finding
created: 2026-08-13T20:03:10Z
created_by: a-codex-maintainer-8c7ncp
about: "[[434]]"
severity: major
---
# Release scenario suite catches one outcome mutation per named fixture
internal/scenarios/scenarios_test.go runs five real-binary offline fixtures. GOCACHE=/private/tmp/dacli-434-gocache go test -count=1 -v ./internal/scenarios passed and logged caught mutations for missing feature integration, incorrect regression repair, missing dependency edge, non-conflicting edits, and trusted malicious origin.
