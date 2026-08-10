---
id: 01KZ70519ERC1PWJT1V25ZSC93
kind: event
event_kind: commit
created: 2026-08-04T18:21:39Z
created_by: a-root
origin: agent
applied: true
---
6d1d1f2 the six-dimension audit: 16 findings, 13 new tasks

Six auditors swept the agent lifecycle, every refusal, the 117-command
surface, core plumbing, the execution lifecycle, and the docs. They
filed through events because their grant is read-only; `dacli sync`
applied them.

The highest-value findings, all now tasks:

286 — the brief truncates findings and decisions at MillerCap=7 in
os.ReadDir order, which is alphabetical by filename. A critical or
refuted finding whose name sorts late is silently dropped while a stale
minor one survives, and the omission is reported as a bare count. The
brief is the whole product; this is it quietly lying.

287 — the protocol preamble tells a READ-ONLY agent 'you commit it, you
open the PR'. It cannot. Every ro auditor today was instructed to do
what its grant forbids and had to work out the propose-and-sync path
for itself.

288 — an empty or lost DACLI_AGENT resolves to root with an rw grant.
Losing an identity fails OPEN, to the most privileged actor in the tree.

289 — `task done` closes a task whose Acceptance section is empty,
because zero boxes checked equals zero boxes required.

290 — `pr --auto` exits 0 with a stderr note when auto-merge could not
be queued, so a caller that checks the exit status believes it landed.

291 — the global --json flag is honored by 4 of 117 commands and
silently ACCEPTED-AND-IGNORED by 40 read commands. Verified live:
`dacli status --json` printed human text and exited 0. Agents parse
output; this is the single biggest agent-experience defect found.

292-295 — one token ceiling under four flag names, ten creatable object
types with no removal inverse, no --dry-run on any GitHub remote-mutating
command while the local ones have it, and a WIP refusal that omits the
remedy its own sibling names.

Two honest negatives worth as much: the 108 Usagef sites and the 63
Refusedf exit classes are clean, and core plumbing's discarded errors
are each justified or surfaced. The auditors said so instead of padding.
role: root
