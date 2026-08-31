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

The page follows the operator's decision order rather than the shape of the API.

1. **Operator pulse** — the first unfinished critical-path task, recorded path,
   attention signals, and system totals.
2. **Projects** — compact workspace progress cards.
3. **Delivery** — selected project task board followed by its dependency graph.
4. **Agents and spend** — burn policy followed by live run evidence.
5. **Team** — role authority, routing, scope, and WIP capacity.

A sticky section rail links to Pulse, Delivery, Agents, and Team. Desktop uses a
wide control-room grid; narrow viewports preserve the same semantic order in a
single column. The first mobile viewport prioritizes next work and attention
over secondary metrics.

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
- Pinia owns independent per-surface polling and retains each surface's last
  good observation. Overview and agents use the fast heartbeat; projects and
  burn are slower; roles are long-lived; only the selected graph is loaded.
- Tailwind utilities and shadcn-compatible primitives use semantic design
  tokens from `src/assets`.
- Vite plus `vite-plugin-singlefile` emits one HTML artifact for Go embedding.
- Components receive data through props; `App.vue` is the only component that
  starts and stops the polling store.
- Project selection is client-only and falls back safely when a project
  disappears between polls.

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

Each surface distinguishes initial loading, live data, stale retained data, and
an unavailable first read. A failed refresh keeps that surface's last good data
visible while healthy sections remain live and usable. Request generations and
abort signals prevent a late response from replacing a newer observation.
Generated timestamps are observation times, not freshness guarantees about
GitHub or a provider. The combined `/api/state` is a legacy compatibility
contract, not the Vue application's heartbeat.

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

Component tests cover all loading/error/empty/live branches and the operator
pulse attention classes. Break the behavior and prove the relevant test goes
red before restoring it.

Landing-page screenshots are generated from
`src/__tests__/fixtures/dashboard-state.json`, the same fixture rendered by the
application test. Captions must label it representative workspace state. Never
present fixture metrics as customer or production evidence.

See [docs/DASHBOARD.md](../../../../docs/DASHBOARD.md) for operator guidance and
[README.md](README.md) for the build and development workflow.
