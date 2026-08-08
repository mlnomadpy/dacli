---
id: f-e5-dacli-agents-tail-shows-each-live-agent-s-last-transcript-line
kind: note
note_kind: finding
created: 2026-07-22T14:56:17Z
created_by: a-3x4vsezdhm
about: [[035]]
severity: moderate
---
# E5: dacli agents --tail shows each live agent's last transcript line
internal/features/execution/execution.go cmdAgents: added --tail flag. For each live agent, lastTranscriptLine(w.RunDir(rec.RunID)/transcript.log) reads the most recent non-empty line (backward scan, no full split) and prints it under the RAM/CPU line as '↳ <line>' truncated to 100 runes via truncateLine (rune-safe + ellipsis). A detached child streams straight to transcript.log so the last line is its current activity — a reasoning agent's tail keeps moving, a hung one is frozen. Default output unchanged; --tail is additive. Empty/missing transcript prints '(no transcript output yet)'. Command Brief updated. Verified helper logic on 6 cases (trailing newline, no-newline, trailing blanks, empty, missing, truncation). go build + go test ./internal/... green.
