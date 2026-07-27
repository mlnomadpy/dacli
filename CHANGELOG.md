# Changelog

All notable changes to dacli are recorded here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project aims
to follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html) once a
first version is tagged.

## [Unreleased]

### Security

Hardened the CLI's own privileged surface so that routing a command **through**
`dacli` can never grant a caller more than its capability allows. The grant
model remains cooperative at the filesystem (see DESIGN § 6), but the tool no
longer acts as an escalation path.

- **Grant gate on privileged subcommands.** `shortcut add`, `runtime add`,
  `project add`, `project rm`, and `kill` — plus the remote-write levers of
  `report` (`--repo`/`--disclose`) and `escalate --github` — now refuse a
  read-only caller with exit 3 (`clikit.RequireRW`). Previously these had no
  capability check, so a read-only agent could define and run executable
  shortcuts/runtimes as the operator.
- **Path-traversal containment.** A project slug (from an explicit `--slug` or a
  forged `--project`) is validated as a single path segment (`workspace.SafeSegment`);
  a value carrying `..` or a separator can no longer read, write, or delete
  outside `.dacli`. `CreateProject` rejects such a slug up front.
- **Git option-injection closed.** Every caller-supplied ref reaches git after a
  `--` end-of-options marker (`fetch`, `push` in `gitx`; the land-status fetch
  in `vcs`), so a value like `--upload-pack=<cmd>` is treated as a refspec, never
  executed.
- **Credential env passthrough denied.** `runtime add` refuses an
  `env_passthrough` naming a known credential variable (`ANTHROPIC_API_KEY` and
  similar). Children run under the operator's own Claude Code login; the
  no-inherited-key rule is now a checked invariant rather than a default value a
  runtime edit could undo.
- **Roster wiki disclosure gate fixed.** `catalog` now probes the visibility of
  the repository it actually publishes to (via `gh repo view --repo`), instead of
  whatever the working directory's remote resolved to — so a private working repo
  can no longer publish the role/skill roster to a public wiki.

### Fixed

Correctness fixes from the same system audit. Several of these mean the tool no
longer misreports its own state.

- **The loop no longer records a zero-commit spawn as done.** A worktree/branch
  is created at spawn time, so branch existence was not evidence of work; a child
  that died before committing left an empty branch that read as an ancestor of
  trunk and was force-accepted as done. Progress is now gated on commits beyond
  trunk, and an empty branch is treated as a failed spawn.
- **CRLF files no longer lose all frontmatter.** `mdstore.Parse` normalizes line
  endings, so a Windows checkout (`core.autocrlf`) no longer yields empty
  frontmatter — every id/owner/status blank — with no error.
- **Free-text flag values can no longer corrupt a task file.** `Front.Set`
  quotes/escapes newlines and `#`, so a value like a pasted multi-line string no
  longer makes a task silently unparseable and invisible.
- **The thrash guard can fire again.** Trunk-progress excludes the loop's own
  per-cycle `.dacli` bookkeeping commit, so `--no-progress-halt` measures code
  reaching trunk rather than the loop narrating itself.
- **`loop --max-cycles N` now bounds an empty-backlog run.** An unproductive idle
  tick counts toward the bound (a productive one that files work does not), so a
  bounded run terminates instead of idling forever.
- **The loop's retro phase runs** instead of exiting with a usage error every
  cycle, and **`ship`/record-ship receive the resolved trunk** (`--into`), so the
  land phase works on repos whose trunk is not `main`.
- **`dacli wait` waits on the whole process group,** not just the leader PID, so
  the loop no longer proceeds to land while a child is mid-commit.
- **Landed task worktrees/branches are garbage-collected** on confirmed merge,
  instead of accumulating one per completed task.
- **A merge-conflict block surfaces a failed persist** instead of reporting the
  task "blocked" while it stays runnable.
- **Unknown/typo'd flags are rejected** (exit 2, naming the flag) on 51 command
  handlers, instead of being silently dropped and the command running against
  wrong or default values.

### Documentation

- Added this changelog.
- DESIGN § 6 documents that the CLI does not escalate its own caller, and the two
  structural guards (path containment, git option-injection) behind it.
- Fixed the `env_passthrough` example in `docs/RUNTIMES.md`, which previously
  showed `ANTHROPIC_API_KEY` — a value the runtime now denies — and documented the
  credential denylist.
- Install docs now lead with `go install` (which works today) and mark the
  Homebrew tap and binary downloads as arriving with the first tagged release,
  so a new user is not sent to a path that 404s.
- Corrected the merged-PR count to the true figure across the landing page,
  README, and docs, and removed an unsourced "6 bugs in its own governor" claim.
- Fixed stale "not implemented / specification only" status headers on the MCP,
  SPM, TEAM, WALKTHROUGH, ARCHITECTURE, and FORMAT pages — all describe shipped
  subsystems.
- `.gitignore` now excludes the `site/` mkdocs build output.

### Known / deferred

- The `dacli` binary sits on the child agents' Bash allowlist at a writable
  path; hardening this is a deployment change (install to a non-writable
  location and allowlist that), not a code change.
- Under local development, `go build ./...` compiles a vendored `.go` file
  inside `ui/node_modules`. It is gitignored (absent in CI and clean clones),
  and the nested-module fix breaks the `ui/dist` embed, so it is left as-is.
