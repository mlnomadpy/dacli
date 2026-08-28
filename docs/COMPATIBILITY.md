# CLI and MCP schema compatibility policy

This is the promise to anyone building against `dacli` from the outside: a
script parsing `--json`, an agent branching on an exit code, an MCP client
that caches tool schemas across a session. It says what you may depend on,
what you may not, and — because the promise is worthless if nobody checks it
— which test breaks when the promise is broken.

`format: 0` in `config.yml` (see [FORMAT.md](FORMAT.md)) governs the on-disk
object files. This document governs the *interface* on top of them: exit
codes, command paths, `--json` shapes, and the MCP tool surface.

---

## 1. What is stable

### Exit codes

Fixed and enumerated in [ARCHITECTURE.md § 4](ARCHITECTURE.md):

| Code | Meaning |
|---|---|
| 0 | Success |
| 1 | Operational failure — the thing exists, the operation failed, retrying may help |
| 2 | Usage error — unknown command/flag/value, the caller's bug |
| 3 | Refused by policy — an answer, never retried |
| 4 | Not found |
| 5 | Conflict |

A command's exit code for a given class of input does not change without a
migration note below. `internal/clikit.ExitCode` is the single place that
maps an error to a code; `internal/clikit/clikit_test.go` pins that the
mapping survives error wrapping.

### Command paths

Once a command path (e.g. `task list`, `context`) ships, it is not renamed
or removed without a migration note. The command table in
`internal/cli/cli.go` (aggregated from every feature slice, per
[ARCHITECTURE.md § 2b](ARCHITECTURE.md)) is the single source of truth —
`dacli catalog` and `--help` are both generated from it, never hand-maintained
copies that can drift.

### JSON shapes

A command's `--json` output is stable **only if** it sets `Command.JSON` and
is enumerated in `internal/cli/json_invariant_test.go`'s
`jsonHonoringCommands`. Every other command either refuses `--json` outright
(exit 2, "does not support --json") or, for `init`/`new`, merely suppresses
human decoration — those two are not structured documents and carry no shape
promise beyond "still human-readable text."

Today that is three document emitters:

**`context --json`**

```json
{
  "task_id": "001",
  "sections": [{"title": "...", "content": "..."}],
  "omitted": ["..."]
}
```

`task_id` is a non-empty string. `sections` is an array of `{title,
content}` objects, in brief-render order. `omitted` is an array (possibly
empty, never `null`) of section names trimmed to fit the budget.

**`task list --json`**

```json
[
  {
    "id": "t-01J...",
    "seq": 1,
    "slug": "add-ledger-shim",
    "project": "p",
    "status": "open",
    "priority": "must",
    "title": "Add the ledger write shim",
    "acceptance_done": 0,
    "acceptance_total": 2
  }
]
```

An array, one object per task. `priority` is omitted when the task has none;
every other field is always present. `status` is one of the values in
`model.AllStatuses`.

**`metrics --json`**

```json
{
  "schema_version": 1,
  "window": {"name": "current", "since": null, "until": null},
  "runs": 2,
  "terminal_runs": 2,
  "completion": {"value": 0.5, "samples": 2},
  "retry": {"value": 0, "samples": 1},
  "failures": {"classes": {"failed": 1}, "samples": 1},
  "wall_time": {"median_seconds": 42, "total_seconds": 72, "samples": 2},
  "tokens": {"output": 1200, "samples": 1, "budget": 4000, "budget_samples": 1},
  "human_intervention": {"value": null, "samples": 0}
}
```

Every metric carries its denominator/sample count. Nullable metric values are
`null` exactly when no sample was captured; they are never replaced by a
fabricated zero. Token usage and configured `--max-tokens` budgets have
separate sample counts. `failures.classes` is always an object (possibly
empty), and `schema_version` changes only for an incompatible revision. `--name` gives a
scenario window a caller-defined comparison label; `--since` records the
resolved UTC bounds in `window`.

**Additive only.** A future field may be *added* to either shape; an
existing field is never renamed, retyped, or removed without a migration
note, and no field silently changes from present to absent (`priority`
already documents its own absence). This mirrors FORMAT.md's rule for
frontmatter: a shape consumer must ignore fields it does not recognize
rather than reject the document.

### MCP tool surface

[MCP.md § 2](MCP.md) tiers the tool catalog, but both tiers inherit the
promise above rather than making a separate one:

- The eighteen Tier-1 tools (`get_context`, `list_tasks`, `claim_task`,
  `check_task`, …) are a manually maintained table in
  `internal/mcp/tools.go`. Their names and parameter shapes are part of this
  compatibility promise and are checked against the documented catalog.
- The Tier-2 `cli` escape hatch returns exactly the `--json` document of
  whatever `argv` it was given — its stability is the "JSON shapes" rule
  above, transitively, for whichever command name the caller passed.
- Exit-code-to-MCP mapping (0 → result, 1/2/4 → `isError: true`, 3 →
  normal result carrying `{"refused": {...}}`) is fixed, per MCP.md § 3.

---

## 2. What may change without notice

- **Human (non-JSON) text output** — column widths, wording, line ordering.
  [ARCHITECTURE.md § 4](ARCHITECTURE.md): "carries no stability promise at
  all — anything parsing it has already lost."
- **Usage/help text and refusal message wording** — the exit code is
  covered above; the sentence describing it is not. (`withUsage` appending a
  `usage: ...` synopsis to an exit-2 message, landed 2026-08-10, is exactly
  this kind of change and needed no migration note.)
- **Flag-parsing edge cases beyond the documented forms** — `--key value`,
  `--key=value`, a boolean `--flag`, and the `--key -- value` escape are the
  only forms this policy covers; anything else (e.g. how a bare `--` is
  tokenized) is an implementation detail, fuzzed for crashes but not frozen
  for shape.
- **Internal Go APIs** — anything outside `cmd/` and the command table.
  [FORMAT.md](FORMAT.md): "The Go API is unstable; this is not [the on-disk
  format]."
- **Command paths and JSON shapes not enumerated above** — unlisted means
  not yet promised, not "safe to assume stable." A command gains a
  compatibility promise only when it is added to `jsonHonoringCommands` (for
  `--json`) or documented here (for command paths generally).

---

## 3. How the promise is enforced

Documentation alone lets a shape drift silently while the sentence describing
it stays put. Each numbered guarantee above has a test that fails if the
guarantee is broken:

| Guarantee | Enforced by |
|---|---|
| Exit code survives error wrapping | `internal/clikit/clikit_test.go`: `TestExitCodeSurvivesWrapping` |
| `--json` is honored or refused, never silently dropped | `internal/cli/json_invariant_test.go`: `TestJSONFlagIsHonoredOrRefused` |
| The JSON-honoring commands actually emit/adapt | `internal/cli/json_invariant_test.go`: `TestJSONHonoringCommandsEmitOrAdapt` |
| `context --json` / `task list --json` / `metrics --json` match the documented field names and types | `internal/cli/compat_json_shape_test.go`: `TestDocumentedJSONShapesStillParse` |

`TestDocumentedJSONShapesStillParse` decodes both responses field-by-field
against the shapes in § 1 and fails on a missing field or a type mismatch; it
does not fail on an unrecognized *extra* field, matching the additive-only
rule.

---

## 4. Migration notes

Dated by the day the change landed. Each entry names what changed, why, and
what a caller must do differently. Entries are never deleted — a caller
reading old integration notes needs the history, not just the current state.

### 2026-08-10 — unknown/typo'd flags now refuse instead of running with defaults

**Before:** `dacli task add "x" --projct p` (typo) silently dropped the
unrecognized flag and ran as if `--project` had never been given, exit 0.
**After:** the same call refuses with exit 2, naming the flag:
`unknown flag(s): --projct`. Landed across 51 command handlers via a single
dispatcher-level gate (`Flags.Reject`), rather than per-handler, because a
rule applied by convention at ~100 call sites had already drifted once
before.
**Caller action:** if a script depended on an unrecognized flag being
ignored, that call now exits 2 instead of 0 — treat any newly-appearing exit
2 in existing automation as a real typo to fix, not a regression to work
around.

### 2026-08-10 — `task list --status <value>` refuses an unrecognized status instead of returning an empty list

**Before:** `dacli task list --status closed` (meaning `done`) matched no
status folder, printed nothing, and exited 0 — indistinguishable from an
empty backlog. Applied to both the text and `--json` paths.
**After:** an unrecognized `--status` value refuses with exit 2, naming the
bad value and the allowed set (`model.AllStatuses`): open, active, blocked,
done. Omitting `--status` is unchanged — it still means "every status."
**Caller action:** a caller filtering on `--status` should already be using
one of the four canonical values; if it exits 2 now, the value was already
wrong and previously read as a false empty result.

### 2026-08-10 — entity-layer policy refusals exit 3, not 1

**Before:** a refusal originating below the CLI dispatcher — e.g. `dacli
note add decision "..." --project p` without `--rejected` — exited 1
(operational failure, "retrying may help"), because `internal/store` cannot
import `internal/clikit` to raise a proper exit-3 error.
**After:** `store.Refusedf`/`store.ErrRefused` gives that layer a way to mark
a refusal without importing the dispatcher package, and `clikit.ExitCode`
maps it to 3 next to the other refusal-marker types. The message text is
unchanged; only the exit code moved from 1 to 3.
**Caller action:** any supervisor loop that retried exit 1 from `note add
decision` (or another entity-layer refusal) was retrying a condition that
retrying could never fix. It now correctly reads as exit 3 and should
escalate or ask instead — no code change needed if the loop already
respects the 1-vs-3 distinction in [ARCHITECTURE.md § 4](ARCHITECTURE.md);
change needed if it was treating "not 0" as uniformly retryable.

### 2026-08-11 — flag parser: a bare `--` is strictly positional again

**Before:** a stray `--` outside the documented `--key -- value` escape
parsed as a flag with an empty name and could swallow the next token (e.g.
`dacli task check 001 -- --dry-run` silently dropped `--dry-run`); the escape
form could also set a boolean flag to a value, bypassing the check that
boolean flags take no value.
**After:** POSIX semantics — everything after a bare `--` is positional,
full stop. A boolean flag can no longer be given a value through the escape.
Found by `FuzzParseFlags` (6 seconds to first failure); both cases are now
permanent regression-corpus entries under `internal/clikit/testdata/fuzz`.
**Caller action:** only relevant to callers relying on the previous
mis-parse (none should have been, since it silently dropped arguments); the
documented escape forms in § 1/§ 2 above are unaffected.
