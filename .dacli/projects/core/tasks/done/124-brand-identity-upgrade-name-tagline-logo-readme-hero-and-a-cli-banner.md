---
id: t-01KY8536JMPYT3RTGQF705HHG1
kind: task
created: 2026-07-23T18:51:34Z
created_by: a-root
owner: a-root
priority: must
---
# Brand & identity upgrade: apply the "autonomous engineering team" identity across README, docs landing, CLI, and dashboard

## So that
dacli reads instantly — to a human landing on the repo or the site — as a self-governing team of AI agents that ships software, not a generic agent framework. The identity is the product's positioning made visible.

## Context
POSITIONING (the north star — everything below serves it):
dacli is an **autonomous engineering team** — a disciplined swarm of specialized AI agents (implementers, reviewers, auditors, an integrator, a visionary) that runs a repository like a real engineering org: sprints, PRs, code review, CI gates, retros — and that GOVERNS ITSELF, knowing when to stop. It self-hosts: it built and hardened itself across 80+ merged PRs. Own the metaphor "engineering team / mission control" — NOT a chaotic "swarm", NOT generic agent orchestration. The moat is governance (a loop that knows when to stop, review that audits its own code, trust/taint gates, calibrated budgets), which is what makes it safe to run unattended on real code.

Voice: confident, concrete, engineer-to-engineer. No hype adjectives — let the self-built proof and the governance do the talking.

Deliverables:
1. TAGLINE — one line on the positioning, e.g. "Your autonomous engineering team — set the direction; it plans, builds, reviews, and ships." Refine one, then use it IDENTICALLY in the README hero, docs/index.md hero, and the CLI banner.
2. MARK/LOGO (SVG, monochrome, terminal-friendly) — reads as coordinated units shipping: a small constellation/cluster or a hive/hexagon motif. Commit under docs/assets/ (or assets/). Must look right next to a `$ brew install` line and as a small browser-tab favicon (provide the favicon too).
3. CLI BANNER — a tasteful, restrained ASCII banner shown by `dacli` (no args) and/or `dacli --version`, carrying the tagline. Stdlib string art only; no figlet wall, no new deps.
4. README HERO — top-of-README block: mark + tagline + the one-line "this tool built itself" proof + `brew install` + the existing loop diagram. First thing a visitor sees.
5. DOCS LANDING (docs/index.md — the site home, already scaffolded by the docs task): above the fold — hero (mark + tagline), `brew install`, a short "This tool built itself (N merged PRs)" proof paragraph, and a HERO-IMAGE slot for the dashboard (task 122): reference docs/assets/dashboard.png with caption "mission control — the live agent swarm". If that PNG does not exist yet, wire the <img> to the documented path and add an HTML comment that it is captured manually from `dacli dashboard`. DO NOT fabricate a screenshot.
6. CONSISTENCY — same tagline, mark, and voice across README, docs landing, `dacli` help/banner, and the dashboard header where trivial.

Constraints: identity/brand ONLY — do NOT rename the binary, module path, or repo (keep `dacli`). Human-facing surfaces only; leave --json / agent-facing output untouched.

## Acceptance
- [x] A single tagline expressing the "autonomous engineering team" positioning appears identically in the README hero, docs/index.md hero, and the CLI banner
- [x] An SVG mark/logo (monochrome, terminal-friendly) + a favicon are committed and referenced by the README, docs site, and dashboard header
- [x] `dacli` (no args) or `dacli --version` prints a tasteful ASCII banner carrying the tagline (restrained; no external deps)
- [x] README hero and docs/index.md lead with mark + tagline + the "this tool built itself" proof + `brew install`, in a confident engineer-to-engineer voice
- [x] docs/index.md wires a dashboard hero-image slot (real screenshot if present, else an <img> to the documented path with a comment that it is captured manually — no fabricated image)
- [x] The binary/module/repo name is unchanged (identity only); go build and go test stay green
## Log
- 2026-07-23T21:41:33Z claimed by a-ksvdbbt934
- 2026-07-23T21:51:02Z accepted by a-root
- 2026-07-23T21:51:02Z completed by a-root
