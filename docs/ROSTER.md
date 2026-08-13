# Team Roster

_Generated from `.dacli/` by `dacli catalog` — do **not** edit this page. It is a one-way read view: to change a role or skill, edit its file under `.dacli/` (via PR), then regenerate. Versions and last-changed come from git history._

A role's **Grant** must agree with its runtime. A `ro` grant is only honest on a runtime that can enforce read-only, so `dacli spawn --grant ro` on a runtime with no read-only sandbox is refused (exit 3), never downgraded to rw — check it with `dacli runtime doctor` (a runtime shown `✗ no read-only mode` cannot back a `ro` role). An `rw` grant is also refused when the runtime's allowlist grants no write tool (`Edit` or `Write`); a runtime with no allowlist makes no such promise and is treated as writable. `--cooperative` explicitly overrides either capability refusal. The runtime is in the role file, not this table; its allowlist is in `.dacli/runtimes/<name>.md`.

## Roles (25)

| Role | Version | Grant | Kind | Model | Skills | Purpose | Last changed |
|------|---------|-------|------|-------|--------|---------|--------------|
| codex-loop-auditor | v3 | rw | — | gpt-5.6-sol | using-dacli | audit one completed loop wave, file only evidence-backed non-duplicate work, and never edit product code | — |
| codex-maintainer | v3 | rw | — | gpt-5.6-sol | using-dacli | implement one dacli task end to end with Codex and preserve every repository contract | — |
| codex-process-architect | v1 | rw | — | gpt-5.6-sol |  | finish task 375 with durable process-tree identity and no recycled-PGID fallback | — |
| estimator | v1 | rw | planner | sonnet |  | PM-style role: sizes an open task with a three-point estimate derived from the codebase, not typed | 5 days ago · chore: move this repo's workspace off trunk onto the dacli-record ref (#393) |
| fixer | v3 | rw | implementer | gpt-5.6-sol |  | implement one scoped task end to end in Go — failing test first, smallest change, then land it | 5 days ago · chore: move this repo's workspace off trunk onto the dacli-record ref (#393) |
| frontend-engineer | v1 | rw | implementer | opus |  | build the Vue dashboard SPA — TypeScript, Composition API, Pinia, Vite, component-driven, accessible | 5 days ago · chore: move this repo's workspace off trunk onto the dacli-record ref (#393) |
| frontend-reviewer | v1 | ro | reviewer | opus |  | review Vue/TS frontend for best practices, accessibility, performance, and design fidelity; never implements | 5 days ago · chore: move this repo's workspace off trunk onto the dacli-record ref (#393) |
| go-auditor | v3 | ro | reviewer | opus |  | audit Go code for performance and best practices | 5 days ago · chore: move this repo's workspace off trunk onto the dacli-record ref (#393) |
| integrator | v3 | rw | reviewer | opus |  | merge done-task PRs to trunk on green CI, autonomously — never implements | 5 days ago · chore: move this repo's workspace off trunk onto the dacli-record ref (#393) |
| junior | v2 | rw | implementer | haiku |  | small well-scoped tasks on the cheap model | 5 days ago · chore: move this repo's workspace off trunk onto the dacli-record ref (#393) |
| loop-bootstrap-auditor | v2 | rw | — | opus |  | audit one completed bootstrap wave and file only evidence-backed non-duplicate work | — |
| loop-bootstrap-maintainer | v2 | rw | — | opus |  | implement one dacli lifecycle blocker end to end using the mature bootstrap runtime | — |
| maintainer | v3 | rw | implementer | gpt-5.6-sol |  | the dacli agent that builds and commits dacli itself | 5 days ago · chore: move this repo's workspace off trunk onto the dacli-record ref (#393) |
| mutation-auditor | v1 | ro | reviewer | opus |  | prove the test suite actually measures what it claims: break the code a passing test covers and confirm the test fails | — |
| persona-adopter | v1 | rw | researcher | opus |  | INTERVIEW SUBJECT — a new human engineer evaluating/adopting dacli; answers as this user: first-run confusion, trust concerns about autonomous agents, what the dashboard must show to build confidence | 5 days ago · chore: move this repo's workspace off trunk onto the dacli-record ref (#393) |
| persona-implementer-agent | v1 | rw | researcher | opus |  | INTERVIEW SUBJECT — an implementer agent in the swarm; answers from the agent's operational POV: what context/steering signals help it, when human intervention helps vs harms, what it needs surfaced (blockers, budget, claim conflicts) | 5 days ago · chore: move this repo's workspace off trunk onto the dacli-record ref (#393) |
| persona-operator | v1 | rw | researcher | opus |  | INTERVIEW SUBJECT — the human visionary/operator who sets direction and steers the swarm from the dashboard; answers as this user: their goals, pains, moments of low control/visibility, and what steering/interactivity they wish they had | 5 days ago · chore: move this repo's workspace off trunk onto the dacli-record ref (#393) |
| persona-reviewer-agent | v1 | rw | researcher | opus |  | INTERVIEW SUBJECT — a reviewer/auditor agent in the swarm; answers from its POV: what it needs to review effectively, how findings should surface, and how humans should be able to approve/gate/steer review outcomes | 5 days ago · chore: move this repo's workspace off trunk onto the dacli-record ref (#393) |
| prompt-auditor | v1 | ro | reviewer | sonnet |  | audit and sharpen the agent prompt registry | 5 days ago · chore: move this repo's workspace off trunk onto the dacli-record ref (#393) |
| reviewer | v2 | ro | reviewer | opus |  | judgment work on the expensive model; reviews PRs, never implements | 5 days ago · chore: move this repo's workspace off trunk onto the dacli-record ref (#393) |
| role-architect | v1 | rw | designer | opus |  | provision the minimal roster an adopted codebase actually needs — each role justified by code that exists, with method written, not metadata | 5 days ago · chore: move this repo's workspace off trunk onto the dacli-record ref (#393) |
| seam-auditor | v1 | ro | reviewer | opus |  | audit COMPOSITIONS of individually-correct features, where this codebase's expensive bugs actually live | — |
| ui-ux-designer | v1 | rw | designer | opus |  | own the dashboard's UX and visual design — layout, information hierarchy, the mission-control aesthetic, responsive + a11y; produce component/design specs the engineer builds to | 5 days ago · chore: move this repo's workspace off trunk onto the dacli-record ref (#393) |
| ux-researcher | v1 | rw | researcher | opus |  | plan and run dashboard UX + product-discovery research: interview guides, synthesis, personas, opportunity framing, and a prioritized feature roadmap grounded in evidence | 5 days ago · chore: move this repo's workspace off trunk onto the dacli-record ref (#393) |
| visionary | v1 | ro | researcher | opus |  | research and upgrade the product vision, features, and direction | 5 days ago · chore: move this repo's workspace off trunk onto the dacli-record ref (#393) |

## Skills (1)

| Skill | Version | Est. tokens | Purpose | Last changed |
|-------|---------|-------------|---------|--------------|
| using-dacli | v3 | 1968 | How to work inside a dacli workspace: the task contract, exit codes, and how to record work so it counts. Triggers whenever you are spawned onto a dacli task. | 5 days ago · chore: move this repo's workspace off trunk onto the dacli-record ref (#393) |
