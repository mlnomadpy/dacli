# Working on dacli as an agent

You are an agent contributing to the tool that spawned you. Read
[CONTRIBUTING.md](CONTRIBUTING.md) first — the verification bar, the exit-code
contract and the slice-isolation rule apply to you unchanged. This file covers
what is different about working inside a loop.

Everything here is written from defects that actually shipped. Where a rule has
a task or issue number, that is the incident.

---

## 1. Your identity and your claim

You were spawned with a token in `DACLI_AGENT`. It decides what you may write.

- **A read-only (`ro`) grant means you propose, you do not apply.**
  `dacli task-check`, `dacli note add finding` and `dacli accept` all record an
  event; the owner materialises it with `dacli sync`. This is not a limitation
  to route around — proposals are how a wave of parallel agents avoids
  overwriting each other.
- **Claim only what you will touch.** The path-claim gate refuses a spawn whose
  files overlap a live agent's. That refusal is doing its job; narrow your
  scope rather than waiting it out.
- **A `3` is final.** Exit 3 is a policy refusal. Retrying it unchanged cannot
  succeed. Read the message — it names the way forward — and if there is none,
  file a finding and stop.

## 2. If you are in a worktree

Your task branch is checked out in a linked worktree under
`.dacli/worktrees/`. Two things bite here:

- **Edit paths relative to your own working directory.** Editing the repo-root
  absolute path writes to the *main* checkout, not your branch. Tests then pass
  against code your commit does not contain.
- **The workspace is shared.** `.dacli` resolves back to the main checkout by
  design (`workspace.Find`), so your notes and task moves land where everyone
  can see them. Do not create a second `.dacli` in your worktree.

Commit with `dacli commit`, not raw `git commit` — it attaches the
`Dacli-Agent` / `Dacli-Role` / `Dacli-Task` trailers that make the work
traceable, and it refuses files outside your claim.

## 3. Prove your work, do not assert it

The failure mode this project cares about most is **a report that disagrees
with what happened**. You are as capable of producing one as the tool is.

- Run the thing. Do not reason about what a command would print — this repo's
  three-way spawn deadlock was found only by *running* the loop, and a task
  filed from an unreproduced premise cost two agent cycles and shipped
  unreachable code.
- `$?` after a pipe is the *last* command's status, not dacli's. Capture the
  output first if you need the code.
- When you add a test, break the code it covers and confirm it goes red. Put
  the failure line in your commit message. A test you cannot make fail is not
  evidence.
- If you could not verify something, say so in the task log. An honest
  "unverified" is worth more than a confident guess, and it is never held
  against you.

## 4. Filing work

`dacli task add "<title>" --project <p> --accept "<criterion>" --accept "..."`

- **Acceptance criteria must be checkable by a different agent** without asking
  you. "it works" is empty. Name the file, the command, the observable state.
- **Lead with the verb that states the intent.** Routing reads the leading verb
  to pick a role kind: "Audit …" and "Trace …" go to a reviewer, "Fix …" and
  "Cover …" to an implementer. A title whose verb misdescribes the work gets it
  routed to a role chartered not to do it.
- **Check for a duplicate first** — `dacli task list --status open`. A previous
  cycle may have filed the same thing in different words.
- **One task, one sitting.** If it cannot be finished in one, decompose it.

## 5. Reporting a problem with dacli itself

Use `dacli report`. It files upstream with version, platform and run context
attached, and withholds your workspace and transcript unless you pass
`--disclose`.

Report the **symptom, your suspected cause, and the manual step you took**. The
agent reports that changed this project most were valuable because of the third
one: the thing you had to do by hand is usually the thing the tool should have
done. Being wrong about the cause is fine — several have been, and were still
the report that found the bug.

## 6. Things that look like progress and are not

- **Inventing work.** If the backlog is genuinely empty and you find no
  evidence-backed defect, file nothing and say so. An honest empty cycle is
  cheaper than sending an implementer to churn working code.
- **Broadening scope mid-task.** Finish what you were spawned for. File the
  rest.
- **Deleting a failing test**, or loosening its assertion to go green. If a test
  is wrong, say why in the commit; if it is right, the code is wrong.
- **Reformatting or renaming beyond your claim.** It buries the real diff and
  collides with every other agent in the wave.
- **Auto-fixers on fixtures.** `golangci-lint --fix` once "corrected" a
  deliberately misspelled flag that *was* a test's fixture, turning an unknown
  flag into a known one and leaving a test that measured nothing. Read what a
  fixer changed before you commit it.

## 7. Before you finish

```bash
gofmt -l . && go vet ./... && golangci-lint run && go test ./...
```

Then mark your acceptance criteria (`dacli task-check`) and leave the task for
the owner to accept. **Do not close your own task** unless you own it — and do
not mark a box you did not actually satisfy. A checked box is a claim, and the
whole point of this workspace is that its claims hold.
