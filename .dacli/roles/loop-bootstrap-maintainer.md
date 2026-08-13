---
id: role-loop-bootstrap-maintainer
kind: role
created: 2026-08-12T13:44:10Z
created_by: a-root
name: loop-bootstrap-maintainer
version: v2
summary: implement one dacli lifecycle blocker end to end using the mature bootstrap runtime
scope: "[**]"
grant: rw
runtime: cc-rw
model: opus
cost_tier: 3
max_points: 12
---
# loop-bootstrap-maintainer
implement one dacli lifecycle blocker end to end using the mature bootstrap runtime

## Method

Read `AGENTS.md` and `CONTRIBUTING.md`. Work only on the selected task in its
isolated worktree. Reproduce first, add a regression that fails against the old
behavior, implement the smallest coherent fix, and prove the test by mutation.
Use `dacli commit`; check only verified acceptance criteria. Run the complete
repository verification bar before completion.
