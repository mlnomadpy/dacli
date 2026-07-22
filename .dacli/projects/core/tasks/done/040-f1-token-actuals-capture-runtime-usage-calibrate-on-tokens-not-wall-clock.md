---
id: t-01KY58MBW9DTHSHCHSXZVWRNNJ
kind: task
created: 2026-07-22T15:55:39Z
created_by: a-root
owner: a-root
priority: must
estimate: {optimistic: 5, probable: 8, pessimistic: 13}
---
# F1: token actuals — capture runtime usage, calibrate on tokens not wall-clock
## Context
This is the keystone the whole D/E series pointed at: calibration's actuals are wall-clock claim→completion, "a time PROXY until runtimes report token usage" (that caveat is printed by calibrate and repeated across the docs). The runtime CAN report usage — `claude --print --output-format stream-json` streams JSON events (tool uses, text) ending in a `result` event carrying `usage` (input/output tokens), `num_turns`, and `total_cost_usd`. F1 captures that and moves calibration onto tokens.

CRITICAL — make it OPT-IN so nothing existing breaks:
- Add a field to `store.Runtime` (runtimefiles.go:16), e.g. `UsageFormat string` (frontmatter `usage_format:`). Empty = today's behavior exactly (plain text transcript, wall-clock actuals). Only when set to `stream-json` does dacli change the invocation and parse usage. Text runtimes (the default `cc`/`cc-rw` unless you opt them in) MUST be byte-for-byte unaffected — spawn, wait, logs -f, agents --tail all keep working.

Implementation:
- `internal/features/execution/execution.go` `execRuntime` (:781) — when `rt.UsageFormat == "stream-json"`, add `--output-format stream-json` (and `--verbose` if the CLI requires it) to the child argv, and instead of piping raw bytes to transcript.log, parse the stream line-by-line: append human-readable text (the assistant text / tool summaries) to transcript.log SO IT STAYS READABLE (logs -f / --tail must still show activity), and capture the final `result` event's `usage.output_tokens`, `num_turns`, `total_cost_usd`. Write them to the run record (e.g. `usage.txt`: `output_tokens: N`, `num_turns: K`, `cost_usd: X`). Detached runs stream to a file too — keep that working (a small stream parser that tees text to the file and remembers the last usage).
- `internal/store/calibration.go` — `CalibSample` gains optional token fields; `CalibrationSamples` reads `usage.txt` from the task's joined run record. Add a token-per-point ratio. `internal/features/insight/insight.go` `cmdCalibrate` — when samples carry tokens, show a token-per-point band per role/model/runtime and PREFER it (label wall-clock as the fallback); update the printed caveat since tokens are now the real unit when present. Keep the wall-clock path for runs without usage.
- Leave the shipped `cc`/`cc-rw` adapters as text (do not force stream-json on them in this task); optionally add a note in RUNTIMES docs on how to opt in. Proving the parser on a mock/stream-json fixture in a test is enough — do NOT require a live claude call in tests.

## Scope (STRICT) — touch ONLY:
- `internal/store/runtimefiles.go`, `internal/store/calibration.go`
- `internal/features/execution/execution.go`
- `internal/features/insight/insight.go`

## Staging discipline
Do NOT `git add -A`. `git add` ONLY the files above plus this task's file (and a test fixture if you add one, under a test dir in-scope). `go build ./...` + `go test ./internal/...` green — and specifically confirm a plain (empty usage_format) runtime still produces a readable transcript and a wall-clock sample. `dacli note add finding` summary, then `dacli commit`. Box-checking is owner-only.

## Acceptance
- [x] a runtime can OPT IN to usage capture (new adapter field, e.g. usage_format: stream-json); text runtimes are unchanged and keep their readable transcript
- [x] when opted in, dacli parses the child's stream-json to (a) write readable text to transcript.log as today and (b) record final usage (output tokens, num_turns, cost) into the run record
- [x] CalibSample carries tokens when available; calibrate reports a token-per-point band (per role/model/runtime) and prefers it over wall-clock, updating the honest caveat; falls back to wall-clock when no usage recorded
- [x] committed on branch by an agent; go build + go test ./internal/... green; existing spawn/wait/logs behavior for text runtimes is unaffected
## Log
- 2026-07-22T15:56:15Z claimed by a-dk00cd6m97
- 2026-07-22T16:08:11Z accepted by a-root
- 2026-07-22T16:08:11Z completed by a-root
