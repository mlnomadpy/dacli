---
id: f-wave-execution-treated-the-enforced-per-worker-token-cap-as-routing-budget
kind: note
note_kind: finding
created: 2026-08-27T22:32:13Z
created_by: a-fixer-ce80x9
about: "[[t-01M12N9W6E9T3SSG3V2P2Y6Y9R]]"
severity: major
---
# Wave execution treated the enforced per-worker token cap as routing budget
internal/features/orchestration/orchestration.go:953 passed perCycleTok as Strategy.TokenBudget and :1254 as Candidate.RemainingBudget. With a calibrated maintainer cost above 20k, TestWaveProfilePreviewMatchesLoopRouting reproduced start preview selecting maintainer while loop launched no worker; the runtime's max-tokens cap already bounds the worker, while the rolling governor admits aggregate spend.
