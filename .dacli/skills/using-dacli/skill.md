---
name: using-dacli
version: v3
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

## Prove it by running it, not by reasoning about it

The rule above has a method. Predicting what a command prints is not evidence,
and this workspace has paid for that repeatedly: a three-way spawn deadlock was
found only by RUNNING the loop, and a task filed from an unreproduced premise
sent two agents to build unreachable code.

- **Run the command. Read the real output.** Before you file a defect, reproduce
  it once.
- **`$?` after a pipe is the LAST command's status, not dacli's.** `cmd | tail`
  reports `tail`'s. Capture the output first if you need the code.
- **A test you cannot make fail is not evidence.** When you add one, break the
  code it covers and confirm it goes red. Put that failure line in your commit
  message. Tests that passed while measuring nothing are the most common defect
  found in this repo — a guard whose refusal branch no test could reach, an
  assertion that accepted *any* error, a check that read a file after the work
  had already finished.
- **Read what an auto-fixer changed.** One "corrected" a deliberately misspelled
  flag that WAS a test's fixture, turning an unknown flag into a known one and
  leaving a green test that proved nothing.

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

## If you are in a worktree

Your branch is checked out under `.dacli/worktrees/`, and two things bite:

- **Edit paths relative to YOUR working directory.** Editing the repo-root
  absolute path writes to the main checkout, not your branch — tests then pass
  against code your commit does not contain.
- **`.dacli` resolves back to the main checkout on purpose**, so your notes and
  task moves land where the team can see them. Never create a second `.dacli`
  inside your worktree.

## The defect worth hunting above all others

**A command that reports success from a path where it did no work.** In a tool
whose product is a record, this is the most expensive bug there is, and it hides
well because everything *looks* fine. Real examples from this workspace:

- a filter matching nothing, so a typo'd status printed an empty list and exited 0
- a gate returning "satisfied" when the read behind it had failed
- a merged task counted as landed even though the close that followed it failed
- a step that narrated an action it never took

If you find one, it outranks whatever you were doing — file it.

## Scope discipline

The task's acceptance criteria are the contract. Work outside them — however
tempting the adjacent mess — belongs in a **new task**, not your diff:

```
dacli task add "<the thing you noticed>" --project <p> --accept "<how to tell it is fixed>"
```

Three things decide whether that task is useful to whoever picks it up:

- **Lead with the verb that states the intent.** Routing reads it to choose a
  role kind: "Audit…", "Trace…", "Review…" go to a reviewer; "Fix…", "Cover…",
  "Add…" to an implementer. A title whose verb misdescribes the work routes it
  to a role chartered not to do it.
- **Acceptance criteria must be checkable by someone else without asking you.**
  Name the file, the command, the observable state. "It works" is empty.
- **Check for a duplicate first** (`dacli task list --status open`). A previous
  cycle may have filed the same thing in different words.

Do not invent work to look productive. If there is genuinely no evidence-backed
defect, say so and file nothing — an honest empty cycle is cheaper than sending
an implementer to churn working code.

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
