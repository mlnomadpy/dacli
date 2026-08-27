---
id: t-01M11KPNTMDJZAHR1YZHH6RF3A
kind: task
created: 2026-08-27T12:39:17Z
created_by: a-root
owner: a-root
github:
  issue: 825
  repo: mlnomadpy/dacli
---
# Align remaining lifecycle-boundary documentation with the agent-first vision
## Context
Adopted from GitHub issue #825.

## Problem

After issues #817–#820 landed, a repository-wide terminology audit found docs/RUNTIMES.md and historical dashboard research still quoting the older `runs agents, not work` boundary without the shipped distinction between deterministic dacli lifecycle orchestration and agent engineering judgment. Readers can interpret those passages as contradicting the shipped loop and the new normative actor model.

## Acceptance
- [ ] docs/RUNTIMES.md states that dacli governs coding-agent lifecycles while agents do engineering work, and retains the rejection of arbitrary job DAGs/cron.
- [ ] Research documents that quote the old phrase either use the refined boundary or clearly label the quote as historical.
- [ ] Historical evidence and interview conclusions are not rewritten to claim observations that were never made.
- [ ] A repository-wide search finds no unqualified normative use of the stale boundary.
- [ ] Documentation tests pass.
## Log
