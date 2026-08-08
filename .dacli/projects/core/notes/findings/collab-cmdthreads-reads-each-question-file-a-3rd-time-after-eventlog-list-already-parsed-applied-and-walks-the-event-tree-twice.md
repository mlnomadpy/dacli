---
id: f-collab-cmdthreads-reads-each-question-file-a-3rd-time-after-eventlog-list-already-parsed-applied-and-walks-the-event-tree-twice
kind: note
note_kind: finding
created: 2026-07-21T23:09:25Z
created_by: a-hp8fwzbck0
about: [[t-01KY3EKR1MSTD09QSJGSW6RSTM]]
---
# collab.cmdThreads reads each question file a 3rd time after eventlog.List already parsed 'applied', and walks the event tree twice
eventlog.List (eventlog.go:96) already WalkDirs + ReadFiles + parses every event's 'applied' field. collab.go:169-188 cmdThreads then calls List TWICE (EventHelp :169, EventAnswer :173 = two full event-tree walks) and, inside the questions loop, os.ReadFile(q.Path) a THIRD time per question just to strings.Contains("applied: true") — the value List already saw. Expose the parsed 'applied' on Event (the Pending query path proves List reads it) and drop the re-read. Related: vcs.go:184+213 cmdContrib does two full List scans (commits, findings); teamops.go:113 cmdAgentTree does a full List(Query{}). Combining kind-filtered passes halves the event-tree I/O.
