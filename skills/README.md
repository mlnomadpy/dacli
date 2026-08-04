# Skills

Two skills ship here, for two different readers. They are not
interchangeable — installing the wrong one teaches the wrong thing.

## `dacli/` — for the agent you are driving

Install this into **your own** coding agent so it knows how to use dacli well:
when to reach for it, how to size and parallelize a backlog, how to route cheap
models to easy work, and which mistakes cost the most (retrying a refusal,
unbounded loops, spawning without a worktree).

It is a standard `SKILL.md` with YAML front matter, so it drops into any agent
that reads a skills directory:

| Agent | Install path |
|---|---|
| Claude Code | `~/.claude/skills/dacli/SKILL.md` |
| project-local (any agent that reads a repo skills dir) | `.claude/skills/dacli/SKILL.md` |
| Codex / OpenCode / Gemini CLI | whatever that CLI's skills or context directory is; if it has none, paste the body into its system prompt or `AGENTS.md` |

```bash
mkdir -p ~/.claude/skills/dacli && cp skills/dacli/SKILL.md ~/.claude/skills/dacli/
```

The content is deliberately runtime-neutral: every instruction is a `dacli`
command, so nothing depends on one vendor's tool API.

## `.dacli/skills/lib/using-dacli` — for the agents dacli spawns

A workspace skill, delivered by dacli itself to **children** it spawns. It
teaches the task contract from the inside: read your brief, stay in claim scope,
record findings and decisions, escalate early, and never claim work you did not
verify.

You do not install this by hand. It is compiled to whatever the target runtime
supports:

```bash
dacli skill compile --runtime cc-rw --role fixer
```

Delivery degrades honestly — native skill directory where the runtime has one,
a managed context file where it doesn't, and inline in the brief as the floor.
`min_delivery: inline` on this skill means it reaches **every** runtime,
including ones with no skill system at all.

## Which one do I want?

- Driving a build yourself, with dacli as a tool → **`dacli/`**
- Running `dacli spawn` / `dacli loop` and want the children to behave →
  **`using-dacli`** (already in the workspace; just compile it)
