# Contributing to dacli

dacli turns a repository into a multi-agent workspace. Its product is a
**record** — of what was decided, what was built, and what was verified — so
the most expensive bug it can have is a record that disagrees with reality.
Almost every convention below exists because that happened.

Humans and agents contribute through the same rules. If you are an agent, read
[AGENTS.md](AGENTS.md) as well; it covers the parts specific to working inside
a loop.

---

## Getting set up

```bash
git clone https://github.com/mlnomadpy/dacli && cd dacli
go build ./cmd/dacli
```

The dashboard SPA is embedded with `go:embed all:ui/dist`, so a clean checkout
needs the frontend built once before the Go build reads it:

```bash
cd internal/features/dashboard/ui && npm ci && npm run build
```

Run everything CI runs, before you push:

```bash
gofmt -l .
go vet ./...
golangci-lint run
go test ./...
```

`golangci-lint` is pinned in CI. Install the same version:

```bash
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2
```

It needs Go 1.25+, while this module targets 1.22 — that is why CI runs it in
its own job on its own toolchain. The 1.22 floor is a compatibility guarantee
and is not bumped for a lint upgrade.

---

## The verification bar

**"The tests pass" is not verification.** This codebase has shipped tests that
passed while measuring nothing, more than once:

- an invariant test that asserted *some* error came back — most commands fail on
  a missing positional first, so it never once exercised the thing it was named
  for. Tightened to require the error *name the flag*, it immediately failed on
  28 commands.
- a safety gate whose refusal branch was reachable by zero tests: changing
  `if requireVerify {` to `if false {` closed unlanded work with a green suite.
- a "streaming output" test that read the file *after* the function returned,
  which proves output arrives eventually and nothing about streaming.

So the bar is: **state the mutation.** Break the code your test covers, and show
the test going red.

```bash
# revert the guard, then:
go test ./internal/features/acceptance/ -run Unlanded
--- FAIL: TestAcceptOneRefusesUnlandedUnderRequireVerify
```

If you cannot construct a mutation that your test catches, the test does not
cover the behaviour — whatever its name says.

**Measure the premise before you file.** A task was once filed here claiming the
loop buffered its redirected output. It does not; Go writes `os.Stdout` straight
through. The premise was never reproduced, two agents spent a cycle satisfying
it, and the "fix" was unreachable code. Reproducing it took one command.

---

## Conventions that are load-bearing

### The exit-code contract

| code | meaning |
|---|---|
| `0` | ok |
| `2` | usage — the caller's mistake |
| `3` | refused by policy — **an answer; never retry it** |
| `4` | not found |
| `1` | everything else |

The `1` / `3` distinction is the one that matters: a supervisor that retries a
refusal enters a loop it cannot exit. `clikit.Usagef` and `clikit.Refusedf`
produce 2 and 3. `ExitCode` maps them with `errors.As`, so wrap with `%w`, not
`%v`, when an error passes through `fmt.Errorf`.

### Declare capabilities on the command, not in the handler

`clikit.Command` carries `JSON`, `Mutates` and `Usage`. The dispatcher enforces
all three.

This is not style. Every guard in this codebase that was applied *by convention*
at ~100 call sites has drifted: `Flags.Reject` reached 4 handlers out of 112,
and four verified grant bypasses shipped next to correctly-gated siblings. A
command that describes itself cannot drift.

### Feature slices never import each other

```
shared    clikit, team, ulid, mdstore, spm, prompts
entities  model, workspace, store, eventlog, agentid, brief
features  internal/features/* — one slice per capability, ISOLATED
app       internal/cli, internal/mcp
```

`internal/cli/arch_test.go` enforces the isolation. When two slices need the
same logic, it moves down into `entities` or `shared` — that is how the landing
check ended up in `internal/store`, so `ship` could use it without importing
`acceptance`.

### Status is folder position

A task's status is *where its file lives*, not a frontmatter field. The event
log is append-only. Read-only agents *propose*; the owner applies.

### Comments explain why

The most valuable documentation in this repo is the header comment on each file
and the reason attached to each guard. Name the defect that shaped the code —
"(dacli 329)", "(issue #421)" — so the next reader can find out whether the
constraint still holds. `ST1000` is disabled in the linter precisely so those
headers can stay.

---

## What gets a change rejected

- A test that passes against the unfixed code.
- A guard that reports success from a path where it did no work — a filter that
  matches nothing, a gate that returns `true` when its read failed, a count
  derived from unvalidated input.
- A refusal downgraded to a warning without saying why.
- Dead code left behind as a "fix" — an unreachable branch that reads as
  handled is worse than an absent one, because the next person believes it.
- A `--dry-run` that describes something other than what the real path does.

## What gets it merged quickly

- A failing test first, then the fix.
- The mutation stated in the PR body.
- One defect per PR, with the *why* in the commit message rather than the diff.
- Naming what you were unsure about.

---

## Filing issues

Use the templates. The reports that moved this project most all had the same
shape: the observed symptom, the suspected cause, and the manual step you had
to take. You do not have to be right about the cause — a wrong guess that names
the right area still saves time, and several have.

## Releases

Releases are cut by the maintainer, from a manually pushed `v*` tag. Nothing in
the automation creates one. Please do not open PRs that add a publishing path.

## Code of conduct

By participating you agree to the [Code of Conduct](CODE_OF_CONDUCT.md).
