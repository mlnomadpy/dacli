# dacli dashboard — shipped UI contract

Status: **implemented and tested.** The Vue source in this directory builds to
one self-contained `dist/index.html` embedded by the Go dashboard feature. This
document records the current interface and the constraints future changes must
preserve.

## 1. Product boundary

The dashboard is a **read-only projection** of dacli's durable workspace. It
helps a human operator understand delivery; it is not a second orchestrator.
There are no spawn, kill, retry, approve, merge, or policy-override controls.
Actions stay in the CLI and MCP surfaces where authority, evidence, and exit
semantics are recorded.

The Go server:

- binds to loopback;
- serves only GET projections and bounded run evidence;
- rejects unrecognized Host headers;
- reads the same store, event log, run records, graph, roster, and calibration
  data as the agent-facing surfaces; and
- embeds the current SPA, with the legacy single-file dashboard as a
  source-build fallback.

## 2. Information architecture

The dashboard is a routed workspace console, not an ever-growing report. Six
stable hash routes follow the operator's decision order while remaining usable
from the embedded, single-file build:

1. **Overview** — lightweight global attention and project portfolio.
2. **Work** — selected-project status board and burndown.
3. **Agents** — measured burn followed by live run evidence.
4. **Team** — role authority, routing, scope, skills, and WIP capacity.
5. **Activity** — the honest durable-event inbox boundary; full chronology is
   added only when the event projection exists.
6. **Delivery** — selected-project dependency graph, schedule, and critical
   path.

Routes use `#/area?project=<slug>` and reserve exact `task`, `agent`, and `role`
selection keys for shared inspection surfaces. The URL is the selection source
of truth: reload, Back, Forward, and direct links do not rely on a private
client navigation stack. Unknown paths and malformed identities fail closed.

The sticky navigation is a six-column desktop header and an explicit two-row
mobile grid; it never hides an off-screen horizontal affordance. Only the
selected route is mounted. Dense tables declare their own labelled scroll
region instead of widening or clipping the page.

## 3. Operator pulse semantics

The pulse is derived only from the current snapshot:

- critical focus is the first non-done node in the selected project's recorded
  `critical_path`;
- attention includes calibrated burn alert, blocked task count, agents in
  `blocked`/`stalled`/`silent`, pending events, and roles whose WIP policy is
  exceeded;
- every attention item links to the explanatory page region;
- “No recorded blockers” says only that these observed signals are calm; and
- no client inference changes scheduling, acceptance, or landing policy.

## 4. Frontend architecture

- Vue 3 single-file components with TypeScript and `<script setup>`.
- A small URL composable owns route parsing and safe selection serialization.
  It deliberately avoids a second router dependency and workflow state machine.
- Pinia owns independent per-surface polling and retains each surface's last
  good observation. The active route defines the observation set: Overview
  requests only overview/projects; Work adds no graph; Agents requests
  overview/agents/burn; Team requests overview/roles; Delivery requests
  overview/projects/the selected graph. Leaving a route aborts its pending
  reads and increments its generation so late responses cannot reappear.
- Tailwind utilities and shadcn-compatible primitives use semantic design
  tokens from `src/assets`.
- Vite plus `vite-plugin-singlefile` emits one HTML artifact for Go embedding.
- Components receive data through props; `App.vue` is the only component that
  starts and stops the polling store.
- Project selection is URL-backed and falls back safely when a project
  disappears between polls. Task, agent, and role query keys are already
  validated for the shared inspection surface added by follow-on slices.

The state contract is typed in `src/types.ts`. Adding a visible fact requires a
real server field and a test fixture update. Decorative counters, fabricated
history, and inferred external health are forbidden.

## 5. Visual system

The “flight ledger” aesthetic shares the public site's graphite/navy base,
signal blue, cyan focus, and explicit success/warning/danger colors. Hierarchy
comes from borders, typography, spacing, and density—not gradients on every
card. Monospace labels identify machine state; readable sans text explains the
operator consequence.

Status never depends on color alone. Focus rings are visible, the document has
a skip link and landmarks, reduced-motion is honored, and no layout may create
horizontal overflow at 390px.

## 6. State honesty

Each route distinguishes initial loading, live data, stale retained data, and
an unavailable first read using only its active surfaces. A failed refresh
keeps that surface's last good data visible while the shell and healthy
observations remain usable. Request generations and abort signals prevent a
late response from a previous route or selection from replacing a newer
observation. Generated timestamps are observation times, not freshness
guarantees about GitHub or a provider. The combined `/api/state` is a legacy
compatibility contract, not the Vue application's heartbeat.

Empty states explain what is absent. An unscheduled graph displays its recorded
note. Missing evidence is never rendered as green.

## 7. Verification and screenshot evidence

Frontend changes must pass:

```bash
npm run format:check
npm run type-check
npm run lint
npm run test:unit
npm run build
```

Component tests cover all loading/error/empty/live branches, safe route
parsing, deep links, focus/current-location semantics, route-to-endpoint request
counts, and stale-response rejection. Break the behavior and prove the relevant
test goes red before restoring it.

Landing-page screenshots are generated from
`src/__tests__/fixtures/dashboard-state.json`, the same fixture rendered by the
application test. Captions must label it representative workspace state. Never
present fixture metrics as customer or production evidence.

See [docs/DASHBOARD.md](../../../../docs/DASHBOARD.md) for operator guidance and
[README.md](README.md) for the build and development workflow.
