---
id: f-cross-branch-seq-collision-reproduced-twice-while-filing-this-wave-s-tasks
kind: note
note_kind: finding
created: 2026-08-04T10:44:48Z
created_by: a-root
---
# Cross-branch seq collision reproduced twice while filing this wave's tasks
Filing three tasks from main just now produced seq 250, 251 and 252. Both 250 and 251 were ALREADY allocated on the branch behind PR #284, to entirely different tasks:

- 250 on main = 'A role can declare grant rw and a read-only runtime'; 250 on #284 = 'doctor reports the loop anchor as orphaned on every run'
- 251 on main = 'Task seq is allocated against the working tree'; 251 on #284 = the TestIDDoesNotLeakTheToken fix

When #284 merges, the project has two different tasks under each of those references, and `dacli task show 250` becomes ambiguous.

This is 209/247 one level up. Task 247 fixed mutual exclusion between concurrent PROCESSES against one tree; the allocator still scans only the tree it is standing in, and a branch is a tree it cannot see. Every agent spawned with --worktree files against its own checkout, so the swarm makes this the normal case, not the edge one.

Filed as task 251 — which is itself one of the colliding numbers, so the register carries its own reproduction.

Adjacent finding, same session: role `junior` declares `grant: rw` but `runtime: cc`, and cc's allowlist is Read/Grep/Glob/LS plus the dacli binary — no Edit, no Write. The junior spawned on task 183 burned a full run discovering it could not edit .gitignore, and tried to invoke an update-config skill to widen its own permissions before giving up. Filed as task 250. Note that `dacli team assign` recommends junior for every open task, so as configured the cheapest route is a route that cannot write.
