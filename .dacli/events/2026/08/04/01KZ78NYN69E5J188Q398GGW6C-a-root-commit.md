---
id: 01KZ78NYN69E5J188Q398GGW6C
kind: event
event_kind: commit
created: 2026-08-04T20:50:42Z
created_by: a-root
origin: agent
applied: false
---
8c0609b skill: teach wave organization, the claim gate, and the unattended loop

The skill told you how to spawn agents and never how to organise a
wave, so the reader does what I did all day: group tasks by package in
their head, spawn six, and resolve the merge conflicts afterwards.

New 'Running a wave' section makes every coordination step a command —
next --parallel picks the set (critical-path ordered, with slack),
team assign picks the role, spawn --claim reserves the paths, wait
closes it. The claim gate gets the space it deserves, with the real
refusal text: it names the holder, the path, and both recoveries. Two
rules follow — group by claim rather than intuition, and treat a refused
claim as scheduling information rather than something to narrow your way
around.

Three traps that cost real runs today are now stated where they bite:
`wait` is what FINALIZES a run, so polling `agents` means a silent
agent looks like a working one; a read-only agent PROPOSES and its
findings are not in the record until `dacli sync`; and a wip:1 role
held by a long-finished agent is permanently unspawnable until you
retire it.

'Running sprints unattended' replaces the thin sprint section: --yolo is
what removes the between-cycle pause, the three bounds are tabled, the
governor idles on an empty backlog rather than inventing work (so 'until
done' terminates), and `touch .dacli/STOP` is the latching kill switch.

New preview section: a --dry-run on a GitHub command is not politeness,
it is the only chance you get, because an issue published to a public
repo can be closed but never unpublished. It carries today's concrete
number — a windowed push still mirrors decisions project-wide, so
--tasks 268 means one task issue and 157 decision issues.

Anti-patterns gain spawning a wave with no claim, polling instead of
waiting, and believing a read-only agent produced nothing before running
sync.

Also files 299 (upgrade the loop to use claims, finalization, sync and
preview — it is currently the one caller that skips them), 300 (audit
the loop end to end), and 301 (docs).
role: root
