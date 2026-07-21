# Project templates and stage gates

**Status: specification. Nothing here is implemented.**

A template defines what kind of project this is: which documents must exist, which stages the work passes through, what must be true to leave each stage, which roles staff it, and what "done" means.

---

## 1. The idea

`dacli init --template <name>` seeds a workspace with a process, not just empty folders. The template carries:

- **Stages** with entry and exit conditions — the "controlled steps"
- **Required documents**, generated from doc templates with placeholders
- **A role roster** and its runtime routing
- **Shortcuts** the project needs
- **A definition of done** that every task inherits
- **Optional GitHub Project mapping** ([GITHUB.md](GITHUB.md))

The point is that an agent starting work in a templated workspace cannot skip the parts that are load-bearing, because the gate will not open.

## 2. The overhead warning, first

Stage gates are bureaucracy. Bureaucracy is sometimes correct and usually not, and a tool that makes it free to add will accumulate it.

The SPM model-selection rule is explicit: *small and simple → Waterfall; large, complex, change-prone → UP or Agile; **never default to the most advanced, because the overhead may exceed the value***. Two of the group anti-patterns are directly about this failure — **Analysis Paralysis** (stuck producing requirements) and **Viewgraph Engineering** (time spent on documents rather than the thing).

So:

- **`solo` is the default template**: one stage, two required docs, no gates. Most work should use it.
- Heavier templates are opt-in and their cost is stated in the template's own description.
- **`dacli doctor` flags a project that has spent more agent turns on gate documents than on tasks.** That is Viewgraph Engineering, it is now mechanically detectable, and a tool that ships stage gates without that detector is selling the disease with the cure.

## 3. Template layout

```
.dacli/templates/<name>/
  template.md          # the manifest
  docs/                # document templates with {{placeholders}}
    vision.md
    architecture.md
  roles/               # seeded roles
  shortcuts/           # seeded shortcuts
  backlog.md           # optional seed tasks
```

Templates resolve from three places, nearest wins: the workspace, `~/.dacli/templates/`, then the built-ins. A project can vendor and edit a template without touching the global one.

## 4. The manifest

```markdown
---
id: tpl-research-paper
kind: template
name: research-paper
process: unified-process       # informational: which model this encodes
summary: theory paper with proof obligations and an experiment protocol
cost: "~4 gate documents; use for work intended for external review"

roles: [theorist, experimentalist, reviewer]

definition_of_done:
  - Acceptance criteria all checked
  - "`dacli run test` passes"
  - "`dacli lint` clean"
  - A finding or decision note recorded

# The DoD binds at `dacli task done`, which VERIFIES rather than records:
# checkable items re-run, and an unmet one is a refusal (exit 3) naming the
# criterion. A definition of done that nothing enforces is a poster on a wall.

stages:
  - name: inception
    cone: definition           # which Cone-of-Uncertainty stage this maps to
    exit:
      - doc: vision.md
        sections: [Vision, Goal, Out of scope]
      - glossary: {min_terms: 5}
      - risks: {min_rank1_with_action: all}

  - name: elaboration
    exit:
      - doc: architecture.md
        sections: [Approach, Rejected alternatives]
      - decisions: {min: 1}
      - tasks: {all_have_acceptance: true, all_have_estimate: true}

  - name: construction
    exit:
      - tasks: {priority: must, status: done}
      - shortcut: test

  - name: transition
    exit:
      - doc: results.md
      - retro: required
---

Why this template exists and when NOT to use it. If a project does not
need external review, use `solo` instead — these four gates cost more
than they return on work nobody else will read.
```

`process:` is informational. It records which model the template encodes (Waterfall, V-Model, Spiral, Unified Process, Scrum, XP, Lean, Kanban) so that a human reading it knows the lineage, but `dacli` does not behave differently based on the name. The behavior lives entirely in the stages.

Stage names are template-local; each stage declares `cone:` to say which Cone-of-Uncertainty stage (`definition` | `elicitation` | `approach` | `design`) it corresponds to, and passing the gate updates the project's `stage:` field, narrowing how every estimate inside it is reported — the honest version of "we know more now." An earlier draft said the stages "map onto" the cone stages and left the mapping implied, which was a bug in this spec: `inception` is not the string `definition`, and an implementer would have had to guess. The mapping is now explicit, and `cone:` is required on every stage.

## 5. Gate predicates

The predicate vocabulary is deliberately small and entirely checkable. **No scripting.** A gate that can run arbitrary code becomes a place people hide logic, and it stops being auditable.

| Predicate | Checks |
|---|---|
| `doc: <name>` | The document exists |
| `doc: <name>` + `sections: [...]` | Those headings exist **and are filled** (§ 5.1) |
| `glossary: {min_terms: N}` | At least N defined terms |
| `risks: {min_rank1_with_action: all}` | Every rank-1 risk has an action plan |
| `decisions: {min: N}` | At least N decision notes with `Rejected` filled |
| `tasks: {all_have_acceptance: true}` | Every task has at least one acceptance box |
| `tasks: {all_have_estimate: true}` | Every task has a three-point estimate |
| `tasks: {priority: must, status: done}` | Every `must` task is done |
| `shortcut: <name>` | That shortcut exits zero |
| `lint: clean` | `dacli lint` reports nothing above the threshold |
| `retro: required` | A retro note exists for this stage |

### 5.1 Placeholder detection is the whole game

The obvious way to defeat a documentation gate is to generate the document and fill it with "TBD". An agent under budget pressure will do this, not from malice but because the gate asked for a heading and a heading is what it produced.

So a `sections:` check verifies the section is **filled**, not present:

- Not empty, and not only whitespace
- Not still containing template placeholders (`{{...}}`, `TODO`, `TBD`, `FIXME`, `...`)
- Above a minimum length
- **Not failing the ambiguity linter at major severity** — a Vision section reading "make the thing better and handle the edge cases properly" is exactly as empty as "TBD", and [the ambiguity scanner](SPM.md) already detects that

That last check is where the SPM layer earns its place in the template system: it is the difference between a gate that checks for documents and a gate that checks for content.

## 6. Document templates

```markdown
---
kind: doc
template_for: vision.md
---

# {{project_title}}

## Vision
{{vision}}

## Goal
{{goal}}

## Out of scope
- {{out_of_scope_1}}
```

Generated docs land in the project folder as ordinary markdown with frontmatter and wikilinks, so they open in Obsidian as vault notes with no export step. Placeholders that survive generation are what § 5.1 detects.

Doc templates should carry **prompts, not just blanks**. A section whose placeholder reads `{{what was rejected, and why the rejection still holds}}` produces better content than one reading `{{rejected}}`, because the agent filling it is reading the placeholder as an instruction. This is the cheapest quality intervention in the whole template system.

## 7. Shipped templates

| Template | Stages | Use when |
|---|---|---|
| `solo` | 1, no gates | **Default.** One person or one agent tree, no external review |
| `standard` | 3 | A team, a real backlog, work others depend on |
| `research-paper` | 4 | Output is a paper: proof obligations, experiment protocol, reproducibility |
| `service` | 4 | Deployed software: runbook, rollback plan, on-call notes required |
| `regulated` | 5 | Audit trail matters; every decision needs a recorded rejection |

`dacli template list` prints their stated costs. `dacli template add` vendors one into the workspace for editing.

## 8. Commands

| Command | Purpose |
|---|---|
| `dacli init --template <name>` | Seed a workspace from a template |
| `dacli template list\|show\|add` | Manage templates |
| `dacli stage` | Current stage and unmet exit conditions |
| `dacli stage advance` | Advance if the gate opens; otherwise list what is missing |
| `dacli stage advance --force` | Override, recording who and why as a decision note |

`--force` exists because a gate that cannot be overridden gets worked around, and a documented override is better than a fake document. The override is recorded as a decision note with `Rejected: waiting for the gate` and reasoning, so it lands in every future brief in scope — the next agent learns the gate was skipped and why.

## 9. Open questions

1. **Gate checks cost tokens.** Placeholder and ambiguity checks are cheap (local, no model call), but `shortcut: test` runs a suite and `lint: clean` may not be free at scale. Gates evaluated on every `dacli stage` call could get expensive; caching against content hashes is the likely answer.
2. **Templates encode opinion, and stale opinion is worse than none.** A vendored template drifts from the global one with no signal. Some form of `dacli template diff` is needed.
3. **Stage-gated work fights the small-task model.** Small, well-scoped tasks want to flow continuously; stages want batch checkpoints. Kanban's continuous flow and UP's phase gates are genuinely different philosophies and this design currently ships both without saying which wins when they conflict.
