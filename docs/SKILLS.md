# Skills: one source format, compiled per runtime

**Status: specification. Nothing here is implemented.**

Roles have declared `skills:` since [TEAM.md](TEAM.md) was written — and that field silently assumed every runtime speaks one vendor's skill system. A `theorist` role loading `math-paper-audit` works on a CLI with native skills and means *nothing* on one without. This document closes that hole: skills are authored once in the workspace, and **compiled to whatever delivery mechanism the target runtime actually has** when a role is spawned onto it.

---

## 1. The fifteenth-standard problem, named before it names us

A "canonical skill format" is the classic move that produces n+1 standards. The mitigation is to refuse to invent one:

**The canonical format is the richest existing target format — a skill directory with a frontmattered `skill.md` — plus dacli-specific frontmatter keys that other tools ignore.** Under the unknown-keys-are-preserved rule the format already lives by, this means:

- A dacli skill **is** a valid native skill for the richest runtime, verbatim. Compiling to that target is a copy.
- `dacli skill import` ingests an existing native skill library (e.g. a `.claude/skills/` tree, or the global one in `~/.claude/skills/`) losslessly — adopt without rewriting anything.
- Compilation only ever goes **out**, toward poorer targets. Nobody is asked to author in a format only dacli reads, because the authoring format is one the ecosystem already reads.

## 2. Canonical layout

```
.dacli/skills/<name>/
  skill.md            # frontmatter + the knowledge
  <resources...>      # reference files, scripts, templates
```

```markdown
---
name: tikz-figures
description: Designing TikZ figures for ML papers — palette, layout archetypes,
  compile-error triage. Use when creating or fixing paper figures.
# dacli extensions (ignored by native hosts):
min_delivery: context        # native | context | inline — worst acceptable (§ 4)
est_tokens: 2100             # compiler-maintained, drives the § 5 warnings
---

The knowledge itself…
```

`name` + `description` are the portable core — the description is the *trigger*, and on every target it survives as the header that tells the agent when this knowledge applies.

## 3. Delivery modes — the fidelity ladder

Adapters declare what they can carry, probed like every other capability:

```yaml
capabilities:
  skills:
    native: { supported: true, dir: ".claude/skills" }   # lazy-loaded skill system
    context_file: { supported: true, path: "AGENTS.md" } # startup memory file
    # prompt-inline needs no support: it always works
```

| Mode | Mechanism | What survives | What it costs |
|---|---|---|---|
| **native** | Materialize the skill directory where the runtime loads skills | Everything: lazy loading, resources, scripts | ~Nothing — the description alone sits in context until triggered |
| **context** | Generate a dacli-managed section in the runtime's startup file (`AGENTS.md`, `GEMINI.md`, …) | The prose, always loaded | **The full body, every turn.** Progressive disclosure is gone; a 2k-token skill is a 2k-per-turn tax |
| **inline** | Prepend to the spawn brief | The prose, this session only | Same tax, plus it competes with the brief's own budget |

Degradation is explicit, per the standing rule: a skill delivered below `native` is announced at spawn with its per-turn cost, and a skill whose `min_delivery` the target cannot meet is **omitted and announced** — a silently absent skill is a role lying about its own competence.

### Executable parts compile to shortcuts

The part of this design I'd defend hardest: a skill's scripts cannot ride a context file — prose travels, executables don't. So on non-native targets, **a skill's executable resources compile into dacli shortcuts**: named, POSIX-quoted, effect-gated, role-scoped ([SHORTCUTS.md](SHORTCUTS.md)). The skill's prose says *when*; the shortcut carries *how*; the effect guard decides *whether*. Nothing new was built — the two subsystems were already the same shape, and compilation is where they meet. A skill script that would arrive ungated on a poor target instead arrives *more* guarded than it was natively.

## 4. Compilation

- **When:** at spawn, as part of profile assembly ([TEAM.md § 4](TEAM.md)) — the role names the skills, the adapter names the mode, the compiler does the rest.
- **Where:** `.dacli/build/skills/<runtime>/<role>/` — **gitignored, regenerable, never edited.** Compiled output is a projection with exactly the standing GitHub doctrine: delete it and rebuild it; the workspace is the only source.
- **Cached** by content hash of (skill set × target mode), so a hundred spawns of the same role cost one compile.
- **Per-role gating is real only under worktree isolation.** In an isolated worktree, the child sees exactly its role's compiled skills. In a shared checkout, a native skill directory is visible to every agent in the repo — so there, skill gating is *advisory*, and spawn says so rather than pretending. Same honesty rule as the permission model, same wording discipline.

## 5. Token economics

The shortcut-catalog argument applies with bigger numbers: on `context` and `inline` targets a skill's *entire body* is an every-turn tax. So the compiler enforces what the catalog enforces:

- A per-role compiled-size budget; exceeding it is a spawn-time warning naming the heaviest skill.
- `est_tokens` maintained on every skill so role authors see the price where they shop.
- The practical guidance stands: role skill lists should be short, and a skill that exists to serve one task is a brief section, not a skill.

## 6. Skills are instructions — the promotion gate

Everything else agents exchange through dacli is **data** — findings and lessons arrive in briefs quote-fenced and attributed, explicitly marked *not instructions*. Skills are the one object whose entire purpose is to be instructions.

That makes the boundary between them a security boundary, and it must be one-way with a human on the gate: **a lesson ([PROPOSALS.md P1](PROPOSALS.md)) never auto-promotes to a skill.** The escalation path to block is concrete — a hostile file poisons a finding, the finding distills into a lesson, the lesson auto-promotes into a skill, and now the injection is *standing instructions compiled into every future agent of that role, on every runtime*. So promotion (`dacli skill promote <lesson>`) is an explicit act by the workspace owner, the produced skill carries `origin:` like any event, and `dacli taint` walks through skills the same as everything else. Compiled output inherits the provenance of its sources.

## 7. Commands

| Command | Purpose |
|---|---|
| `dacli skill add\|list\|show` | Author and inspect workspace skills |
| `dacli skill import <dir>` | Ingest a native skill tree, losslessly |
| `dacli skill compile --role R --runtime T [--dry-run]` | Materialize; `--dry-run` prints modes, sizes, and omissions |
| `dacli skill promote <lesson>` | Owner-gated lesson→skill promotion (§ 6) |

## 8. Open questions

1. **Context-file cohabitation.** `AGENTS.md` often already exists and is human-authored. The dacli-managed section must be marker-delimited and idempotently rewritten — the same never-touch-a-human's-text rule as GitHub comments — but merge conflicts between the human's half and the compiled half have no clean answer yet.
2. **Per-runtime prompt dialects.** The same prose lands differently across vendors' models. Compilation currently treats skill text as portable; if per-target phrasing ever matters, that's a template layer this design does not want and might need.
3. **Native-format drift.** The richest target's skill format is itself evolving; tracking it is the same treadmill as adapter flags, and `runtime doctor` should probe skill-dir conventions the way it probes everything else.
