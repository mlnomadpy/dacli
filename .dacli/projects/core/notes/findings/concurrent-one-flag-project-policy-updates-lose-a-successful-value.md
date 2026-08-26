---
id: f-concurrent-one-flag-project-policy-updates-lose-a-successful-value
kind: note
note_kind: finding
created: 2026-08-26T15:20:20Z
created_by: a-adversarial-reviewer-dftw7y
about: "[[t-01M0F8DMCN93FCDE59FSEDTJB3]]"
severity: major
against: a-root
---
# Concurrent one-flag project policy updates lose a successful value
internal/features/planning/planning.go:130-151 (authored by root role a-root) performs an unlocked LoadProject -> merge flags -> SaveProject read-modify-write. Trigger: from configured local/main, run 'project show p --landing-mode pr' and 'project show p --landing-base release' concurrently. Reproduced on iteration 1: both processes exited 0, but a later JSON show reported configured/effective local/release, so the successful mode update was lost. Wrong outcome: updating one flag does not preserve the other concurrently, and a success report disagrees with durable state later observed by ship/integrate/loop. Serialize project policy updates or use a compare-and-swap/retry around the project record, with a synchronized public-command regression.
