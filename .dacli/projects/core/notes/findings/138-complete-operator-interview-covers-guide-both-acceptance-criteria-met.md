---
id: f-138-complete-operator-interview-covers-guide-both-acceptance-criteria-met
kind: note
note_kind: finding
created: 2026-07-24T09:33:37Z
created_by: a-z32782m8xr
about: [[138]]
severity: moderate
---
# 138 complete: operator interview covers guide, both acceptance criteria met
docs/research/interviews/operator.md (commit a3d939c) answers INTERVIEW_GUIDE.md sections 3/7/8 firsthand in the operator's voice. AC1: goals (loop-driver framing), current pains (Q3 thinking-vs-hung, Q5 confidently-wrong, Q7 kill, Q13 silent overspend), a must/nice/no verdict on every hypothesis H1-H8 (incl. H3 split into view=nice / steer=no-as-chat, H6 split into approval=no / queue=nice), and un-hypothesized needs (thinking|acting|waiting|stalled state field; confidently-wrong gauge; escalation-not-chat; git-log narrative workaround; headless-operator segment). AC2: every answer is a past-behavior story citing real surfaces (dacli agents --tail, status, Swarm panel, governor windowSpent, internal/spm/criticalpath.go, killed.txt) and it ranks the top 3 the segment would pay for: 1) H7 burn-rate-that-yells, 2) H1 DAG view, 3) H5 pause. Docs-only; all relative links resolve. Owner (a-root) should check boxes 1-2 and close via task done + merge --task 138.
