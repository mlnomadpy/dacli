---
id: f-pid-pgid-reuse-stale-proc-txt-makes-dead-runs-resurface-as-live-mis-samples-and
kind: note
note_kind: finding
created: 2026-07-22T16:17:27Z
created_by: a-1s5dxes66y
about: [[t-01KY59FNEAAYDZ0PCTKE0HCBVA]]
source_event: 01KY59NN40Y4YSG6BZASBKJC7S
---
# PID/PGID reuse: stale proc.txt makes dead runs resurface as live, mis-samples and kills unrelated process groups
procmon liveness is a bare signal-0 probe with NO identity check (no start-time/comm match). procmon.go:125 Alive() returns true for any live PID (even EPERM/root-owned), :156 SampleGroup sums RSS/CPU of every system process whose pgid==rec.PGID, :233 KillTree SIGKILLs -rec.PGID. execution.go:1378 liveAgents reads EVERY run's proc.txt and lists it if Alive(rec.PID); proc.txt is never removed when a foreground/finished run ends (only runs prune drops whole dirs after keep=20). So within the last ~20 runs, a finished run whose PID got recycled by an unrelated process (Linux default max PID 32768, recycled fast on busy CI) will: (a) show as a live agent in 'dacli agents' with the OTHER process's RAM/CPU (execution.go:1146,1155); (b) be SIGTERM/SIGKILL'd by 'dacli kill <ref>'/'kill --all'/'agents --reap' — killing an unrelated process group (execution.go:1367-1372,1350-1352,1165-1167 -> killOne :1403 -> KillTree). Same root cause in cmdWait (execution.go:1478): a detached child whose PID is reused stays Alive forever, so wait blocks until the overall timeout instead of finalizing. Fix: record and re-verify process identity (start time via ps -o lstart=, or /proc/PID stat starttime) before treating a PID/PGID as the spawned agent, and delete proc.txt on clean completion.
