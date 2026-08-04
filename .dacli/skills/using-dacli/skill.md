---
name: using-dacli
version: v2
description: How to work inside a dacli workspace: the task contract, exit codes, and how to record work so it counts. Triggers whenever you are spawned onto a dacli task.
min_delivery: inline
created_by: a-root
---
# using-dacli

You are working inside a **dacli workspace**: a shared markdown store that
several agents and a human read and write. Your work only counts if it is
recorded there. Code that exists but was never recorded is, to everyone else,
work that did not happen.

This skill is runtime-neutral. Every instruction below is a `dacli` command, so
it works identically whether you are Claude Code, Codex, Gemini CLI, OpenCode,
or anything else that can run a binary.

## The one rule that matters most

**Never claim something you did not verify.** Not in a commit message, not in a
task log, not in a completion. A false "done" is worse than an honest "stuck",
because a false done stops anyone from looking. If you could not finish, say so
and leave the task open.

## Exit codes are a contract, not decoration

| Code | Meaning | What you do |
|---|---|---|
| 0 | worked | continue |
| 2 | usage error — your invocation is malformed | fix the command line |
| **3** | **refused by policy** | **this is an ANSWER. Never retry it.** |
| 4 | not found | the ref names nothing; check it |
| 1 | something else failed | read the message |

Retrying a **3** is the single most expensive mistake an agent makes here. It
will never succeed, and the loop will burn its whole budget discovering that.

## Your loop

```
dacli context <ref>          # your brief: task, acceptance, decisions, findings
                             # READ THIS FIRST. It carries what the team already
                             # decided, so you do not re-propose a rejected option.
# ... do the work ...
dacli commit "<what and why>" --task <ref>
dacli pr --task <ref>        # if you are landing through a PR
dacli accept <ref> --verify "<the command that proves it>"
```

`dacli commit` sets the git author to **you**, with provenance trailers. Use it
instead of `git commit` — a plain git commit loses the attribution that lets
reviewers target findings at the right agent.

**Always `git add` your own paths, then `dacli commit --no-add`.** Without
`--no-add` it stages everything in the tree, and in a wave that means your
siblings' half-finished edits land inside your commit, under your name. If you
are in your own worktree it is still the right habit; if you are not, it is the
difference between a clean diff and a PR nobody can review.

## Landing your branch

Your branch name is a lookup key, not a label. `dacli integrate` finds it as
`dacli/<seq>-<slug>` using the task's own slug, and **silently skips** anything
else — no error, just "0 PRs opened". If `spawn --worktree` made the branch, it
is already correct; if you make one yourself, copy the slug from the task file
exactly.

```
git add <the files you touched>
dacli commit "<what and why>" --task <ref> --no-add
dacli task check <ref> --all && dacli task done <ref>
git push -u origin dacli/<seq>-<slug>
```

Then stop. Opening and merging the PR is `integrate`'s job, run from trunk —
and `integrate` refuses from any branch but the trunk you are merging into, so
you cannot do it from your worktree anyway. Do not reach for `gh` to work
around that: the merge has to be recorded as an event against the task, and a
merge that leaves no event did not happen as far as the workspace is concerned.

## Scope discipline

The task's acceptance criteria are the contract. Work outside them — however
tempting the adjacent mess — belongs in a **new task**, not your diff:

```
dacli task add "<the thing you noticed>" --project <p> --accept "<how to tell it is fixed>"
```

If your spawn recorded a `--claim`, `dacli commit` will refuse files outside it.
That refusal is correct; do not `--force` past it without a reason you would say
out loud.

## When you are stuck, say so

```
dacli ask "<question>" --about <ref>       # blocks the task until answered
dacli note add finding "<what you found>" --project <p> --origin <file>:<line>
dacli escalate "<why nobody in the tree can decide this>"
```

Escalating early is a **correct outcome**, not a failure. An agent burning a
long budget on work that needed a different decision, a bigger model, or a human
is the waste all of this exists to prevent.

## Recording what you learned

A finding needs a file:line and the concrete input or state that goes wrong.
"Handles errors poorly" is not a finding. If you cannot say how it fails, you
have a suspicion — mark it as one.

Decisions are the highest-value thing you can write, because they stop the next
agent re-proposing what you already rejected:

```
dacli note add decision "<what you chose>" --project <p> \
  --rejected "<the option you turned down>" --because "<why>"
```

## Before you say you are done

- The acceptance criteria are literally satisfied — reread them.
- You ran the real checks (build, tests, formatter) and they passed.
- Anything you could not do is written down, not omitted.
- `dacli accept --verify "<cmd>"` records WHAT proved it. A close with no
  verification is recorded as unverified, and everyone downstream will see that.
