---
id: t-01KY59FNEAAYDZ0PCTKE0HCBVA
kind: task
created: 2026-07-22T16:10:34Z
created_by: a-root
owner: a-root
priority: should
---
# AUDIT R2: execution + procmon — runtime and process code
## Acceptance
- [x] findings filed with file:line in internal/features/execution/** and internal/procmon/**
## Log
- 2026-07-22T16:11:07Z claimed by a-1s5dxes66y
- 2026-07-22T16:17:27Z finding by a-1s5dxes66y: PID/PGID reuse: stale proc.txt makes dead runs resurface as live, mis-samples and kills unrelated process groups (event 01KY59NN40Y4YSG6BZASBKJC7S)
- 2026-07-22T16:17:27Z finding by a-1s5dxes66y: stdin-mode --detach truncates or drops the prompt: parent exits before the stdin-copy goroutine finishes (event 01KY59NYECGWADBYXKX8T9FYZR)
- 2026-07-22T16:17:27Z finding by a-1s5dxes66y: Detached stream-json runs write raw JSON to transcript.log, so logs -f and agents --tail show unreadable events (event 01KY59P8AK5DAC78G0BV0FPFCJ)
- 2026-07-22T16:17:27Z finding by a-1s5dxes66y: teeStreamJSON silently loses token usage on a scanner error or an over-long line; sc.Err() is never checked (event 01KY59PH15Q60XJSYV0G3NFQCM)
- 2026-07-22T16:17:27Z finding by a-1s5dxes66y: 'dacli agents' reports ps lifetime-average %CPU as if it were current utilization (event 01KY59PTD63E3FBW9SE2YJ9S3K)
- 2026-07-22T18:52:27Z accepted by a-root
- 2026-07-22T18:52:27Z completed by a-root
