---
id: t-01KY59FNFAEE0KT7PWV8HAAY4A
kind: task
created: 2026-07-22T16:10:34Z
created_by: a-root
owner: a-root
priority: should
---
# AUDIT R5: mdstore + eventlog + prompts + workspace + clikit — core plumbing
## Acceptance
- [x] findings filed with file:line in internal/{mdstore,eventlog,prompts,workspace,clikit,agentid,ulid,model}
## Log
- 2026-07-22T16:11:07Z claimed by a-0htd7wjqyt
- 2026-07-22T16:17:27Z finding by a-0htd7wjqyt: mdstore.WriteFile leaks its temp file when os.Rename fails (event 01KY59SR9XT0BFNAG6M2YK2S0Z)
- 2026-07-22T16:17:27Z finding by a-0htd7wjqyt: eventlog.List silently drops malformed or unreadable events (event 01KY59T2D9YDAH289R2WT5HBBK)
- 2026-07-22T16:17:27Z finding by a-0htd7wjqyt: sync.go logOnce comment claims a fresh-per-pass reload that no longer happens (event 01KY59TAFKB2JEPGKDV2N701H7)
- 2026-07-22T16:17:27Z finding by a-0htd7wjqyt: R5 scope: six prior sibling findings verified FIXED in current tree (event 01KY59TVZ2EZ732CYG3RRC8CJS)
- 2026-07-22T16:17:27Z finding by a-0htd7wjqyt: R5 audit coverage: workspace/clikit/agentid/ulid/model reviewed, no new defects (event 01KY59VJAJ0DVFKZHX2YN9P5E1)
- 2026-07-22T18:52:27Z accepted by a-root
- 2026-07-22T18:52:27Z completed by a-root
