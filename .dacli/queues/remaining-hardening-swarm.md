---
id: q-remaining-hardening-swarm
kind: queue
created: 2026-08-12T19:02:25Z
created_by: a-root
owner: a-root
cursor: 4
---
# Finish the GitHub-backed dacli hardening backlog in claim-safe waves
## Steps
1. Wave 1: implement 393, 396, and 389 concurrently with disjoint claims
2. Land and accept Wave 1 through GitHub PRs; rebuild /private/tmp/dacli-loop-current
3. Wave 2: implement 397, 368, and 373 concurrently after overlapping claims are released
4. Land and accept Wave 2; remove .dacli/STOP only after task 373 is verified
5. Wave 3: implement 366 with the corrected explicit Codex role routing
6. Wave 4: implement 367 after tasks 366 and 368 are landed
7. Run full gates, close mirrored issues, publish dacli-record, and verify an empty ready backlog
