---
template: home.html
title: "dacli — control plane for autonomous coding-agent swarms"
---

<!-- The home page is rendered by overrides/home.html (a custom conversion
     landing). This markdown is a fallback and is not shown on the site. -->

# dacli

<p align="center">
  <img src="assets/logo.svg" width="72" height="72" alt="dacli mark — a coordinated cluster of hexagonal units">
</p>

<p align="center"><strong>The control plane for autonomous coding-agent swarms.</strong></p>

![status: alpha](https://img.shields.io/badge/status-alpha-orange) ![go 1.22+](https://img.shields.io/badge/go-1.22%2B-00ADD8) ![deps: stdlib only](https://img.shields.io/badge/deps-stdlib_only-success) ![license: MIT](https://img.shields.io/badge/license-MIT-blue) ![surfaces: CLI · MCP](https://img.shields.io/badge/surfaces-CLI_·_MCP-6f42c1)

Give an orchestrator AI agent a product direction. dacli gives it durable
context, critical-path planning, isolated execution, model routing, budgets,
recovery, and verified GitHub landing across Codex, Claude Code, Gemini CLI,
Copilot CLI, and generic executable adapters.

```bash
go install github.com/mlnomadpy/dacli/cmd/dacli@latest
```

> Homebrew and prebuilt binaries come with the first tagged release; for now `go install` is the supported path.

<p align="center">
  <img src="assets/dashboard.png" alt="dacli dashboard — mission control for the live agent swarm" width="720">
  <br>
  <em>mission control — the live agent swarm</em>
</p>

> Markdown on disk, folders for structure, a CLI and an MCP server as the two front ends. Zero dependencies outside the Go standard library.

An agent that spawns subagents has one hard problem: **each child starts blind.** It re-reads the codebase, re-derives decisions its siblings already made, and re-attempts work that already failed. `dacli` is the shared workspace that fixes this — a durable, human-readable project state that any agent in the tree can query, and that the parent can slice down to exactly the context a given child needs.

Everything is markdown with YAML frontmatter and `[[wikilinks]]`. That means git diffs it, `grep` searches it, GitHub renders it, Obsidian opens the workspace as a vault with no plugin, and you can fix it by hand when an agent writes something stupid.

## One governed product-building loop

```text
direction → plan and route → execute and review → verify and land → learn ↺
```

- **Human governor:** direction, credentials, exceptions, emergency stop, and
  release policy.
- **Orchestrator agent:** goal interpretation, decomposition, prioritization,
  and product or architecture judgment.
- **dacli:** durable state, policy, routing, budgets, agent lifecycle, recovery,
  and landing requirements.
- **Coding-agent CLIs:** isolated implementation, review, and testing.
- **GitHub:** visible issues, PR review, CI, merge, and release evidence.

## What the control plane provides

|  | |
|---|---|
| 🧠 **Context on tap** | `dacli context <task> --budget N` returns one self-contained, token-budgeted brief — task, goal, constraints, prior decisions, siblings' findings — instead of the whole repo. |
| 🚀 **Replaceable coding CLIs** | Route by capability, cost, context, and health across supported runtimes without changing the workflow. |
| 📊 **Measures its own cost** | `calibrate` learns each *role × model × runtime*'s real cost — in **tokens**, not guesses — then `spawn --advise` / `--max-tokens` size and gate the next launch by it. |
| 🛡️ **Bounded autonomy** | Claims, grants, token windows, trust gates, stop conditions, and recovery journals make every autonomous cycle explicit and inspectable. |
| 🔎 **Resource-safe** | `agents` shows each live tree's RAM/CPU/GPU + last transcript line; `kill` reaps the whole process group — no runaway agents. |
| 🔗 **GitHub, both ways** | `github push` mirrors tasks→issues, decisions→issues, findings→issues (severity-labeled); `github pull` adopts issues as tasks — all behind a disclosure gate. |
| 📓 **Everything recorded** | Every run freezes its brief, invocation, transcript, and outcome; every commit is attributed to the agent and role that authored it. |

## Install

**From source** (requires Go 1.22+) — the supported path today:

```bash
go install github.com/mlnomadpy/dacli/cmd/dacli@latest
```

The two options below come with the first tagged release; until then, use `go install`.

**Homebrew** (macOS/Linux) — *coming with the first tagged release*:

```bash
brew install mlnomadpy/tap/dacli
```

**Direct download** — prebuilt darwin/linux/windows binaries (amd64+arm64) will be attached to each [GitHub release](https://github.com/mlnomadpy/dacli/releases) *once the first release is tagged*:

```bash
curl -sSL https://github.com/mlnomadpy/dacli/releases/latest/download/dacli_<version>_<os>_<arch>.tar.gz | tar xz
```

## Start through one operating profile

```bash
# In the target repository
dacli adopt --provision-roles

# Inspect the resolved roles, runtimes, budgets, verification, and landing policy
dacli start --project <slug> --profile loop --dry-run

# Run bounded, governed cycles using that same resolved policy
dacli start --project <slug> --profile loop
```

## Where to go next

- Start with the [documentation index](README.md) for the full reading order.
- [Architecture](ARCHITECTURE.md) is the normative spec — axioms, layers, build order, the canonical brief.
- [Operator playbook](OPERATOR_PLAYBOOK.md) chooses task, wave, loop, or service
  boundaries and shows the recovery path.
- [Walkthrough](WALKTHROUGH.md) traces one task end to end through the whole system.
- The source lives at [github.com/mlnomadpy/dacli](https://github.com/mlnomadpy/dacli); [DESIGN.md](https://github.com/mlnomadpy/dacli/blob/main/DESIGN.md) is the project's original contract.

## License

MIT
