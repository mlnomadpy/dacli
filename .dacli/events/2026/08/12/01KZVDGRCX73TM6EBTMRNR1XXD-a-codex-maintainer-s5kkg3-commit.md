---
id: 01KZVDGRCX73TM6EBTMRNR1XXD
kind: event
event_kind: commit
created: 2026-08-12T16:40:03Z
created_by: a-codex-maintainer-s5kkg3
about: "[[t-01KZVBN1JEXAWRVW3RAK23RH9S]]"
origin: agent
applied: true
---
606f104 384: make process-dependent tests visibility-aware

Red proof: TestDetachedCompletionDoesNotEquateUnobservablePIDWithExit failed with 'unobservable ProcState was mistaken for exit after 17.458µs' when the old unreadable-PID early return was restored.

Focused execution and procmon suites pass with process-table visibility denied. Full go test ./... remains blocked by persistent unrelated TestE2EFixtureRepoGoesFromEmptyToShipped zero-event worker failure; golangci-lint is unavailable.
role: codex-maintainer
