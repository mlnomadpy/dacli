---
id: f-049-complete-on-branch-dacli-049-teestreamjson-commit-608a642-all-4-acceptance
kind: note
note_kind: finding
created: 2026-07-22T16:30:32Z
created_by: a-1syt6ccpg3
about: [[049]]
severity: major
---
# 049 complete on branch dacli/049-...teestreamjson (commit 608a642) — all 4 acceptance met
Commit 608a642 by a-1syt6ccpg3. Staged ONLY the 5 scoped files: internal/procmon/{procmon.go,procmon_test.go}, internal/features/execution/{execution.go,verify.go,stream_test.go}.

(1) PID/PGID REUSE GUARDED: procmon gains ProcStart(pid) (ps -o lstart= — an absolute start stamp, stable per-process, identical on macOS/Linux), Record.PIDStart (persisted as pid_start:, parsed keeping text after the first colon), AliveIdentity(pid,wantStart) = Alive(pid) AND live start time == recorded (a recycled PID has a different start time -> treated as gone; empty wantStart on legacy records falls back to bare Alive), and AliveRecord(r). onStart in cmdSpawn/cmdSupervise/verify.go captures pidStart(pid). Every liveness probe now uses AliveRecord: liveAgents (execution.go), cmdLogs -f (execution.go), cmdWait (execution.go). All kill paths (kill <ref>, kill --all, agents --reap) funnel through liveAgents, so a stale/recycled proc.txt is filtered out before any SampleGroup/KillTree — a dead run cannot resurface as live nor steer KillTree onto an unrelated group.

(2) stdin+detach prompt: the detach branch's stdin now writes the prompt to an os.CreateTemp *os.File and sets cmd.Stdin to it (fd inherited at exec, unlinked after Start; inode survives via the child fd), replacing strings.NewReader whose parent-side copier goroutine died on Release()+exit and truncated/dropped prompts over the ~64KB pipe buffer. Detached stream-json transcripts: transcript.log stays RAW json (finalizeRun still harvests usage from it, no race), and the READERS render on read — new renderStreamLine + renderTranscriptTo make cmdLogs/logs -f and lastTranscriptLine (agents --tail) show readable assistant text + [tool: X] instead of raw JSON. (Decision note records why render-on-read beats a live renderer subprocess.)

(3) teeStreamJSON: rewritten on bufio.Reader.ReadBytes (no line cap) so an over-long earlier event no longer aborts the stream before the terminating result event that carries usage; streamUsage gains scanErr, set on any non-EOF read error, surfaced to the transcript as '[dacli: usage capture incomplete]' so a missed result event is visible instead of masquerading as a text runtime. Shared renderStreamLine decodes each event for both the tee and the readers.

(4) agents %CPU relabelled 'CPUavg' with a comment noting ps %cpu is a per-process lifetime average, not current load; Usage.CPUPct doc corrected.

go build ./... clean; go test ./internal/... all green incl. new TestAliveIdentityRejectsRecycledPID, TestTeeStreamJSONLongLineDoesNotLoseUsage, TestTeeStreamJSONSurfacesReadError, TestRenderTranscriptToRendersRawStreamJSON, TestLastTranscriptLineRendersRawStreamJSON, and the existing cli/execution suites. Box-checking refused for non-owner (only a-root) — owner: verify and close via dacli task check/done + dacli merge --task 049.
