---
id: d-detached-stream-json-stays-raw-on-disk-render-on-read-instead-of-a-live
kind: note
note_kind: decision
created: 2026-07-22T16:29:32Z
created_by: a-1syt6ccpg3
about: [[049]]
---
# Detached stream-json stays raw on disk; render on READ instead of a live renderer subprocess
## Chose
Detached stream-json stays raw on disk; render on READ instead of a live renderer subprocess
## Rejected
Pipe the detached child through a live dacli renderer subprocess that writes readable text to transcript.log
## Because
A separate renderer would race finalizeRun's usage harvest (which parses raw stream-json from transcript.log after the child exits) and would have to duplicate usage capture to avoid losing tokens. Keeping transcript.log raw for detached runs and rendering in the readers (cmdLogs, lastTranscriptLine via renderStreamLine) makes logs -f/--tail readable with zero risk to usage capture and no new process.
