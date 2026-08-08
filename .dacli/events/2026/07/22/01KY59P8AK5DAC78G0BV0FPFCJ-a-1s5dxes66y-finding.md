---
id: 01KY59P8AK5DAC78G0BV0FPFCJ
kind: event
event_kind: finding
created: 2026-07-22T16:14:10Z
created_by: a-1s5dxes66y
about: [[t-01KY59FNEAAYDZ0PCTKE0HCBVA]]
origin: agent
applied: true
---
Detached stream-json runs write raw JSON to transcript.log, so logs -f and agents --tail show unreadable events

The stream-json tee that makes the transcript human-readable (assistant text + [tool: X] markers) runs ONLY on the foreground path: execution.go:870-871 pipes stdout and :887-889 calls teeStreamJSON(streamPipe, sink). The detach path (816-841) wires cmd.Stdout/Stderr straight to the sink (824-826), so a detached stream-json child writes raw {"type":...} JSON lines to transcript.log. Consequences: 'dacli logs <run> [-f]' (cmdLogs :1214) prints raw JSON; 'dacli agents --tail' (lastTranscriptLine :1274, shown :1159-1163) surfaces a raw JSON line as the agent's 'current activity', defeating the E5 thinking-vs-hung readability feature precisely for detached agents — the common case that --tail exists to observe. finalizeRun (:1548-1556) later harvests usage from the raw log, so tokens are fine; only readability is lost. Fix: tee stream-json to a readable transcript even on detach (e.g. a small detached rendering pass), or note the raw format in logs/--tail output.
