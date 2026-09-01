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
2. **Work** — searchable selected-project task identities grouped by status,
   burndown, and exact task inspection.
3. **Agents** — selected-project loop operation, measured burn, then live run
   evidence.
4. **Team** — role authority, routing, scope, skills, WIP capacity, and exact
   live occupancy.
5. **Activity** — a bounded newest-first evidence spine over the durable event
   journal, with typed refusal/finding/review/reconciliation/delivery labels.
6. **Delivery** — task-selected attempt waterfall plus the selected-project
   dependency graph, schedule, and critical path.

Routes use `#/area?project=<slug>`; `#/team?role=<name>` selects an exact role,
`#/agents?agent=<id>` selects an exact durable agent, and
`#/work?project=<slug>&task=<exact-id>` selects one task for inspection. The URL is the selection source of truth: reload, Back,
Forward, and direct links do not rely on a private client navigation stack.
Unknown paths and malformed identities fail closed.

Every known route also renders one compact **observation strip** below its
intro. The strip is shared investigation context, not workflow state. Its safe
URL fields are `project`, `q`, `filter_role`, `runtime`, `model`, `state`,
`range` (`24h`, `7d`, or `30d`), activity `kind`, `actor`, `event_state`, and
stable `cursor`, plus `live=paused`. Navigation preserves this context. A
route applies only filters backed by its typed projection and explicitly names
any preserved filter it cannot apply.

Current route capability is deliberately narrow:

- Overview and Delivery apply exact project scope. Work also applies search to
  the complete selected-project task rows.
- Agents apply exact project scope to the loop-operation projection, and search,
  role, runtime, agent state, and time window to the live-agent observation;
  the same time window bounds the displayed burn series.
- Team applies search, exact role filter, runtime, and model.
- Activity sends project, task, kind, actor, pending/applied state, range, and
  cursor to the bounded server projection; it never filters a hidden full
  journal in the browser.

The strip reports filtered/observed counts and the newest local observation.
Pause aborts in-flight reads and stops every active timer while retaining the
last good snapshot; Resume restarts only the current route's declared
surfaces. Pause means “freeze automatic local observations,” never “the
underlying workspace stopped.”

Desktop uses a compact 56px command bar and a narrow sticky left rail so the
route title begins near the viewport top instead of below three stacked hero
bands. Below 1024px the rail becomes an explicit two-row mobile grid; it never
hides an off-screen horizontal affordance. Only the selected route is mounted.
At mobile width the live-agent and role roster tables become evidence cards
with 44px Inspect targets. Their complete desktop tables remain available at
the documented breakpoint rather than forcing a horizontal-only mobile view.

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
  requests only overview/projects; Work adds selected-project task rows and
  lazy selected-task/event detail but no graph; Agents requests
  overview/projects/the selected loop operation/agents/burn; Team requests
  overview/roles/agents; Delivery requests
  overview/projects/the selected graph/the selected task timeline. Leaving a
  route aborts its pending reads and increments its generation so late
  responses cannot reappear.
- Activity requests overview, the bounded project catalog, and one `/api/events`
  page. Cursor pagination is exclusive and stable under concurrent appends. Partial/unreadable,
  truncated, stale, empty, and cold-error states remain distinct; retained
  stale data is never relabelled live.
- The selected graph is bounded by the server contract, not CSS. Operational
  scope is capped at 120 nodes, exact focus is a capped two-hop neighborhood,
  and completed history is paginated at 100 nodes. The `projection` field names
  the rule and complete/visible/hidden node and edge counts; the client never
  downloads the entire historical DAG merely to hide it.
- Tailwind utilities and shadcn-compatible primitives use semantic design
  tokens from `src/assets`.
- Vite plus `vite-plugin-singlefile` emits one HTML artifact for Go embedding.
- Components receive data through props; `App.vue` is the only component that
  starts and stops the polling store.
- Project selection is URL-backed and falls back safely when a project
  disappears between polls. Role selection is URL-backed but never falls back:
  a disappeared role renders an exact missing-state record rather than another
  roster entry.
- Role, agent, and task inspection use the same controlled sheet model. Desktop presents a
  bounded right drawer and mobile a full-width sheet. It has an explicit row
  button, labelled dialog semantics, Escape dismissal, trapped focus, and
  trigger focus restoration. Role detail displays canonical routing policy and
  occupancy. Agent detail is fetched only for the selected exact id, caches the
  current observation, refuses mismatched response identities, and displays
  durable lineage, ownership, and newest-first live/dead run evidence. A
  disappeared live row does not erase its retained durable record. Neither
  sheet exposes workflow mutation. Task rows are fetched once per selected
  project; detail and task-scoped events are fetched only for the selected exact
  id, cached by that id, and identity-checked before they can replace retained
  evidence. Task status movement does not change selection. Parent/dependency
  links and keyboard-focusable graph nodes open the same sheet; dangling edges
  stay visibly unresolved.
- Observation filtering is pure over complete typed surface payloads. The URL
  composable validates and serializes filter values; the filter composable
  declares route capability and transforms observations; Pinia alone owns the
  pause/resume lifecycle.
- Delivery attempt evidence is projected server-side as
  `delivery-attempt-timeline/v1`. The client never reconstructs success from
  transcript text. Each run remains a separate attempt; corrupt or stale phase
  evidence is a refusal, and absent timestamps remain null/unknown. Desktop
  spans are keyboard navigable and mobile renders the same ordered facts. One
  stable diagnosis class binds the current task branch, commit, tree, PR
  generation, checks, review, merge, and acceptance evidence; superseded PRs
  stay historical. The Overview's delivery-attention item is a slow-polled
  local-record projection and never performs a heartbeat GitHub request.
- Loop operation evidence is projected server-side as `loop-operation/v1` from
  the selected project's durable loop status, recovery checkpoint, phase
  journal, complete-cycle preflight, token-reservation ledger, operating
  profile, and sourced progress explanation. The browser does not parse those
  internal files or infer a remaining budget. The 10-second local poll never
  invokes GitHub, a provider, or a mutation. Numeric remaining capacity is
  visible only for enforceable ledgers; advisory, unknown, missing, partial,
  stale, and corrupt states remain explicit. Candidate routes are narrowed by
  the recorded harness policy before rendering.

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

The shell uses three surface levels only: page background, bordered command
surfaces, and raised evidence cards/drawers. The global read-only boundary is
stated once in the footer and documentation rather than repeated as a badge on
every route. Monospace is reserved for identities, observation labels, and
machine evidence; route names and explanatory text use the readable sans face.

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

Event bodies cross an explicit untrusted-data boundary: the Go projection
applies the centralized public-safe sanitizer and a UTF-8-safe bound, and Vue
interpolates the result as text. No `v-html`, external resource fetch, local
path, secret, or journal mutation belongs in the Activity route.

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
`src/__tests__/fixtures/dashboard-state.json` plus the typed loop-operation
fixture in `src/__tests__/App.test.ts`, the same contracts rendered by the
application tests. Captions must label this representative workspace state.
Never present fixture metrics as customer or production evidence.

See [docs/DASHBOARD.md](../../../../docs/DASHBOARD.md) for operator guidance and
[README.md](README.md) for the build and development workflow.
