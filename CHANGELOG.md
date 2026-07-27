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

### Documentation

- Added this changelog.
- DESIGN § 6 documents that the CLI does not escalate its own caller, and the two
  structural guards (path containment, git option-injection) behind it.
- Fixed the `env_passthrough` example in `docs/RUNTIMES.md`, which previously
  showed `ANTHROPIC_API_KEY` — a value the runtime now denies — and documented the
  credential denylist.
