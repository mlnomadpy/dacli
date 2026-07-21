# Shortcuts

**Status:** the pure engine (`internal/shortcut`: expansion, quoting, effect guards, catalog) is implemented and tested; commands are stubs.

Named, parameterized command templates. `dacli run test` instead of regenerating `go test ./... -count=1` for the four hundredth time.

---

## Why, honestly

The token argument is real but small. Regenerating a command costs ~20 tokens; naming one costs ~3. Multiply by a few hundred invocations and you have saved something, but not enough to justify a subsystem.

**The stronger argument is that a shortcut is a memoized derivation.** The first agent to get a command right paid to discover the correct flags, the right working directory, the environment variable the suite needs, the fact that `-race` deadlocks on this codebase. That knowledge normally evaporates when the session ends, and the next agent rediscovers it — usually getting it subtly wrong first, then debugging its own invocation instead of the problem.

A shortcut turns that derivation into something durable, reviewable, attributable, and diffable. Which is the same argument as the rest of `dacli`, applied to commands.

There is a third benefit that only shows up at scale: shortcuts are **auditable**. When commands are generated fresh each time, nobody can answer "what commands does this agent tree actually run?" When they come from a committed file, that question has an answer, and a dangerous one can be caught in review rather than in production.

### The honest cost

A catalog of shortcuts costs tokens in **every brief that advertises it**, roughly 12 tokens per entry. A shortcut nobody calls is a permanent tax on every child agent forever.

So the catalog is ranked by use count and truncated, with the truncation announced. An unadvertised shortcut still exists and still runs — it just stops being pushed into everyone's context. This is the same budget discipline the context assembler uses, applied to the same problem.

## Definition

`.dacli/shortcuts/<name>.md`:

```markdown
---
id: sc-test
kind: shortcut
name: test
summary: run the Go test suite
command: go test {{pkg}} [[ -run {{pattern}} ]] -count=1
effect: read
dir: .
params:
  - name: pkg
    default: ./...
  - name: pattern
    summary: optional -run regex
roles: [backend, reviewer]
uses: 47                  # derived from run events at sync; never hand-edited
---

Use this rather than calling `go test` directly: `-count=1` defeats the
result cache, which otherwise reports stale passes after a dependency edit.
```

The body is the part worth writing. It records *why this shortcut exists in this form* — which is the knowledge that would otherwise be rediscovered.

### Template syntax

Two constructs, no more:

- `{{name}}` — substitute a declared parameter.
- `[[ ... ]]` — optional group, dropped entirely if any placeholder inside resolves to empty.

The optional group exists because a placeholder that resolves to empty still renders as an empty quoted argument, and passing a flag an empty argument is not the same command as omitting the flag. Without it, `test` with no pattern would run every test named "".

Resisting further template features is deliberate. A shortcut file that needs conditionals and loops is a shell script, and should be one — `command: ./scripts/thing.sh {{arg}}`.

## Safety

### Every value is quoted

Parameter values are wrapped in POSIX single quotes unless the parameter declares `raw: true`. This is not configurable per call, and it is the security boundary of the feature.

Parameter values routinely carry model-generated or file-derived text. A template rendered by string concatenation becomes an arbitrary-command vector the first time a value contains a semicolon — and "the model will only pass sensible values" is not a security argument, especially when the value can originate in a file the model was asked to summarize.

`raw: true` exists for the few legitimate cases (passing a pre-built flag list) and is declared in the committed shortcut file rather than at the call site. Enabling it is therefore a reviewable change, not a decision an agent makes in the moment.

### Effects gate execution

| Effect | Meaning | Gate |
|---|---|---|
| `read` | Observes only: tests, linters, `git log` | Any agent |
| `write` | Changes the working tree or local state | Requires an `rw` grant |
| `destructive` | Irreversible or outward-facing: deploy, push, drop | Requires `rw` **and** explicit confirmation |

A shortcut with no declared effect does not run. Defaulting to `read` would mean a typo in the frontmatter silently downgrades a deploy.

The confirmation requirement on `destructive` exists because those are precisely the commands an agent should not reach by autocompleting a name. Without it, `deploy` is one token away from `test` in a list the model is skimming.

`roles:` is an independent second gate. A backend agent with a write grant still has no business running the frontend deploy — capability and toolkit answer different questions.

## Where shortcuts come from

Mostly not from an agent deciding to write one. **Asking an agent to predict which commands it will repeat does not work** — it has no memory of the last session and no visibility into its siblings.

The intended source is promotion: `dacli` watches the event log for the same command recurring, and suggests it.

```
$ dacli doctor
shortcut candidate: `go test ./internal/spm/ -run TestCPM -v` ran 7 times
across 3 agents in this session. `dacli shortcut promote --from-event 01J8...`
```

This is the same trick as the anti-pattern detectors, over the same log, and it is only possible because every command invocation is already an attributed event.

## Commands

| Command | Purpose |
|---|---|
| `dacli run <name> [--param v]` | Expand and run |
| `dacli run <name> --dry-run` | Print the expanded command without running it |
| `dacli run --list` | Full catalog, including unadvertised entries |
| `dacli shortcut add` | Define one |
| `dacli shortcut promote` | Turn a repeated ad-hoc command into a shortcut |

`--dry-run` matters more than it looks: it lets a reviewing agent inspect what a shortcut *would* do without the effect gate, and it makes the quoting behavior visible when debugging a template.

## This narrows a stated non-goal

DESIGN.md § 2 says `dacli` is **not a job runner** — "no process execution, retries, or timeouts."

Shortcuts execute processes, so that non-goal is now narrower than it was, and pretending otherwise would leave the design doc lying. The amended boundary:

- ✅ `dacli` runs **one named command** and reports its exit status and output.
- ❌ `dacli` does not schedule, retry, time out, run steps in dependency order, or manage a work queue of processes.

Queues remain what they were: ordered step lists with a cursor, where the agent executes and `dacli` records position. The line is that `dacli` never decides *when* something runs or what to do when it fails. It expands a name into a command, checks the gates, runs it, and gets out of the way.
