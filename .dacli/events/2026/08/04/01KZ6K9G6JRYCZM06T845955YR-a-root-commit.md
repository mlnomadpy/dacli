---
id: 01KZ6K9G6JRYCZM06T845955YR
kind: event
event_kind: commit
created: 2026-08-04T14:36:54Z
created_by: a-root
origin: agent
applied: false
---
cc31348 record 264's audit, and the regression that nearly ate it

The auditor on 264 found a real defect in code that shipped WITHOUT CI
(PRs 322 and 327, task 263): milestoneExists reads
`gh api repos/<repo>/milestones?state=all` with no --paginate, GitHub
caps that at 30 per page, and GitHub does not enforce milestone-title
uniqueness. Past 30 milestones the check goes blind, ensureMilestone
POSTs a create that SUCCEEDS as a duplicate, the re-list still cannot
see it, and every push adds another duplicate while issues are never
grouped. It is the lone uncapped list read in a file where
fetchAllIssues uses --limit 1000 and project.go has
projectItemListLimit — the inconsistency with its own neighbours is the
tell. Hand-verified before filing. Task 266.

It could not file any of that itself, because of me. protocolPreamble
tells a child which dacli to run using os.Executable(), the runtime
allowlist names an absolute path, and nothing checks the two agree. Task
165 moved the allowlist to ~/go/bin/dacli while I kept spawning with
./dacli, so every child since was told to run a path its sandbox no
longer permitted. The auditor reported it plainly, read the backlog by
opening task files directly to check it was not re-filing, did the audit
anyway, and handed me the command to file with.

'Prior siblings hit the same wall' is the line that matters. The agent
on 200 completed and wrote nothing and I read that as the agent
failing — I redid 200 by hand instead of investigating. Same
regression, second wasted run, already reported. Task 267 makes spawn
refuse the mismatch.
role: root
