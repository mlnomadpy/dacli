---
id: t-01KY59FNDY9ECT40SDBF71VWBH
kind: task
created: 2026-07-22T16:10:34Z
created_by: a-root
owner: a-root
priority: should
---
# AUDIT R1: internal/store — entity hub
## Acceptance
- [x] findings filed with file:line for bad patterns, perf, quality in internal/store/**
## Log
- 2026-07-22T16:11:07Z claimed by a-gkrbmb023t
- 2026-07-22T16:17:27Z finding by a-gkrbmb023t: calibrate walks RunsDir 2-3x per readout: runBands+runUsage each ReadDir+parse every invocation.txt (event 01KY59MR5BQGY3TYC5XCZ2DC5Z)
- 2026-07-22T16:17:27Z finding by a-gkrbmb023t: TaskBand re-scans the entire RunsDir to resolve one task's band — O(tasks x runs) if ever looped (event 01KY59MYBFN0MWMKBFCQZHYEP9)
- 2026-07-22T16:17:27Z finding by a-gkrbmb023t: taint over-reports blast radius: a workspace-scoped METRIC note is marked TreeWide but WorkspaceLessons never surfaces metric notes (event 01KY59N88YK81E8ATQJP3ZKYXX)
- 2026-07-22T16:17:27Z finding by a-gkrbmb023t: Slugify returns empty string for punctuation-only or non-ASCII titles, producing '.md' filenames and 'f-'/'d-' ids (event 01KY59NGMPN6KMZXAWB0YXNEHY)
- 2026-07-22T16:17:27Z finding by a-gkrbmb023t: GradeFinding grades a non-deterministic note when two findings share a title; logSpan inflates the span across re-claim/re-complete cycles (event 01KY59NRTKMXVZJ1QR1JYJB5Z2)
- 2026-07-22T16:17:27Z finding by a-gkrbmb023t: Taint resolves refs via FindTask per hit (O(hits x tasks)) though this package already ships TaskIndex for exactly this loop (event 01KY59P8YHH6MXXK7QPN9RYSY1)
- 2026-07-22T18:52:27Z accepted by a-root
- 2026-07-22T18:52:27Z completed by a-root
