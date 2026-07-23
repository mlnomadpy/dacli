# dacli dashboard — UI Design Spec

Status: **draft — the contract for the frontend build.** The frontend engineer
builds to this document; where the shipped UI and this spec disagree, the spec
wins until it is amended. It defines the layout, the Vue component tree, a dark
"mission-control" color and type system, and **every state** (loading, empty,
error, live) for the three surfaces — Overview, Task board + Burndown, and the
Live agent swarm — plus responsive behavior and accessibility requirements.

This is a **spec, not an implementation.** No `.vue` files ship with it. The
current dashboard is a single self-contained page at
[`../static/index.html`](../static/index.html); this document is the plan for
re-expressing that page as a small Vue application without changing what the
server exposes.

---

## 0. Ground truth — the data contract (do not invent fields)

The UI is a **read-only projection** of one JSON snapshot. The server
(`internal/features/dashboard/dashboard.go`) serves the embedded page at `/` and
a JSON snapshot at **`GET /api/state`**. The page polls that endpoint; nothing
in the UI ever mutates the workspace. This mirrors the server-side doctrine:
`buildState` reads the store fresh on every call and never caches, so a poll
always reflects current truth (the same honesty rule `dacli agents` follows —
liveness is re-probed, never trusted from a stale read).

The snapshot shape, verbatim from the Go structs — **these are the only fields
the UI may bind to:**

```jsonc
{
  "generated": "2026-07-23T16:10:00Z",   // RFC3339 UTC; when this snapshot was built
  "pending_events": 3,                    // unsynced child events (eventlog), as `dacli status` reports
  "projects": [
    {
      "slug": "core",
      "title": "dacli remaining backlog",
      "stage": "build",
      "total": 42,                        // task count across all statuses
      "counts": { "open": 12, "active": 3, "blocked": 1, "done": 26 }, // status -> count
      "burndown": {
        "done_points": 88.5,
        "remaining_points": 31.0,
        "unestimated": 4,                 // tasks with no PERT estimate
        "per_day": [                       // done points that landed each day, chronological
          { "day": "2026-07-20", "points": 12.0 },
          { "day": "2026-07-21", "points": 8.5 }
        ]
      }
    }
  ],
  "agents": [
    {
      "run_id": "01KY8KW3W1GSP57K39ZY77NH6S",
      "child": "a-nhkth9j71n",
      "task": "131",
      "role": "designer",
      "runtime": "claude",
      "pid": 48213,
      "started": "2026-07-23T16:00:00Z",  // RFC3339 UTC
      "runtime_secs": 600,                 // uptime, seconds
      "last_activity": "2026-07-23T16:09:40Z" // transcript.log mtime, RFC3339 UTC; falls back to `started`
    }
  ]
}
```

Field-level rules the UI must honor:

- **`counts` is a partial map.** A status key is absent when its count is zero.
  Bind through a `count(status)` helper that returns `0` for a missing key —
  never assume all four keys exist. The four statuses are `open`, `active`,
  `blocked`, `done`, in that order everywhere they appear.
- **`per_day` may be empty** (nothing has been completed yet) and is already
  sorted chronologically by the server — the UI must not re-sort or fabricate
  missing days.
- **`unestimated` tasks contribute to `total` and `counts` but not to points.**
  The burndown bar is drawn from `done_points`/`remaining_points` only;
  `unestimated` is surfaced as a caption, never folded into the bar.
- **All timestamps are RFC3339 UTC.** Render relative ("4m ago") in the
  viewer's locale; keep the absolute UTC value in a `title`/tooltip.
- **`agents` is newest-first and already liveness-filtered** by the server. The
  UI shows exactly what it receives; it does not probe, sort, or hide agents.
- The snapshot has **no error field.** Failure is a transport/HTTP concern
  owned by the poller (§6), not a value inside `state`.

> If a future surface needs a field not listed here, that is a server change
> (`dashboardState` in `dashboard.go`) and a spec amendment first — the UI does
> not derive data the snapshot does not carry.

---

## 1. Product intent — "mission control"

One screen a human glances at to answer three questions, top to bottom:

1. **Overview** — is the workspace healthy right now? (projects, stage, one
   pending-sync signal, connection liveness)
2. **Task board + Burndown** — per project, where is the work? how much is
   done vs. remaining, and is it landing?
3. **Live agent swarm** — who is running this instant, on what, for how long,
   and are they still moving?

The aesthetic is a **dark operations console**: calm dark field, restrained
palette, monospace for identifiers, a single pulsing "live" tell, and status
carried by a small fixed set of hues. No decoration that doesn't encode state.
It reads as instruments, not a marketing page.

---

## 2. Technology & delivery constraints

- **Vue 3, `<script setup>` SFCs, Composition API.** TypeScript for the state
  contract types.
- **Ships as a single embedded artifact.** The Go server `go:embed`s one built
  HTML/JS bundle exactly as it embeds `static/index.html` today. The build step
  (e.g. Vite) emits an **inlined, single-file** bundle so the "zero server,
  zero deps, self-contained" property from `docs/PROPOSALS.md` (P8) survives.
  No CDN, no runtime network fetch except `/api/state`.
- **No router, no store library.** One page, one polled snapshot. State is a
  single reactive object from a `usePoll` composable (§6); Vue's reactivity is
  the only "store." Adding Vuex/Pinia/vue-router is out of scope and a smell.
- **No CSS framework.** The design tokens (§3) are hand-authored CSS custom
  properties, carried over from the existing page so the two never drift while
  both exist.
- **Polling, not websockets.** Keep the 2000 ms poll (`POLL_MS` today). The
  server is stateless per request; websockets would add a connection to babysit
  for no gain at this cadence.

---

## 3. Design system — color & type

### 3.1 Color tokens

Defined once as CSS custom properties on `:root` with `color-scheme: dark`.
**These values are carried over verbatim from `static/index.html`** so the Vue
build is visually identical to the page it replaces and neither drifts:

| Token         | Value     | Role                                              |
|---------------|-----------|---------------------------------------------------|
| `--bg`        | `#0f1115` | app background (near-black blue-gray)             |
| `--panel`     | `#161a22` | card / table surface                              |
| `--border`    | `#262b36` | hairline dividers, card borders, empty bar track  |
| `--text`      | `#e6e9ef` | primary text                                      |
| `--muted`     | `#8b93a7` | secondary text, captions, table headers           |
| `--open`      | `#8b93a7` | status: open (same as muted — neutral, "waiting") |
| `--active`    | `#4f8cff` | status: active / remaining-points segment         |
| `--blocked`   | `#e5534b` | status: blocked / error                           |
| `--done`      | `#3fb950` | status: done / done-points segment / live pulse   |

Semantic aliases layered on the base tokens (new, for clarity in components):

| Alias           | Resolves to  | Used by                                    |
|-----------------|--------------|--------------------------------------------|
| `--accent`      | `--active`   | logo mark, focus rings, primary emphasis   |
| `--danger`      | `--blocked`  | error banners, blocked count               |
| `--ok`          | `--done`     | live indicator, success                    |
| `--surface-2`   | `--panel`    | nested surfaces (e.g. table inside a card) |

**Status → color is a single source of truth.** A `statusColor(status)` helper
maps `open|active|blocked|done` to its token. Every status dot, count chip, and
board column reads from it — no per-component hardcoded hex.

**Contrast:** all four status hues and `--text`/`--muted` meet **WCAG AA (≥4.5:1
for text, ≥3:1 for the ≥18px/bold and for non-text UI)** against `--bg`/`--panel`.
`--muted` (#8b93a7) on `--panel` (#161a22) is ~5.0:1 — AA-safe for the small
caption text it carries. Any new foreground token must be contrast-checked
against both `--bg` and `--panel` before use (§8).

### 3.2 Typography

| Role                | Stack                                                             | Size / weight            |
|---------------------|------------------------------------------------------------------|--------------------------|
| App / UI (default)  | `-apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif` | 14px / 1.5 |
| Identifiers (mono)  | `ui-monospace, SFMono-Regular, Menlo, monospace`                 | inherit; used for run_id, pid |
| H1 (page title)     | default stack                                                    | 18px / 600               |
| H2 (section label)  | default stack, `text-transform: uppercase; letter-spacing:.06em` | 13px / 600, `--muted`    |
| H3 (card title)     | default stack                                                    | 14px / 600               |
| Table header        | default stack, uppercase, `letter-spacing:.05em`                 | 10px / 600, `--muted`    |
| Caption / points    | default stack                                                    | 11–12px, `--muted`       |

Rules: **monospace is reserved for machine identifiers** (`run_id`, `pid`) so a
human can tell an ID from prose at a glance. Numbers that are read as
quantities (points, counts, durations) use the default stack. Uppercase +
letter-spacing marks structural labels (section and table headers) only, never
running text.

### 3.3 The mark

The logo is the **hexagon-cluster diamond mark** already inlined in the page
header and favicon (four nested hexagons: one center, three satellites) — "many
coordinated units, not chaos," reinterpreted from `docs/assets/logo.svg`
(decision [[d-124-dashboard-header-inlines-the-mark-svg-plus-a-data-uri-favicon-instead-of]]).
It ships as an inline `<svg>` in the header (`currentColor`, `--accent`) and a
data-URI favicon. **No new static-asset route** — same constraint the current
page holds.

### 3.4 Spacing, radius, motion

- **Radius:** `8px` on cards/tables/banners, `4px` on the burndown bar, `50%`
  on dots. One step down inside chips.
- **Spacing scale:** 4 / 6 / 8 / 12 / 16 / 24 / 32 px. Page padding
  `24px 32px 64px`; card padding `14px 16px`; grid gap `12px`.
- **Motion:** exactly one ambient animation — the **live pulse** (`opacity 1 →
  .35 → 1` over 2s) on the connection dot and per-agent activity dot. All
  motion is wrapped in `@media (prefers-reduced-motion: reduce)`, which
  **disables the pulse** and substitutes a static filled dot (§8). No layout
  animation, no spinners that spin forever, no parallax.

---

## 4. Layout

Single-column, vertically stacked sections on a max content width of **1280px**,
centered on wide screens, full-bleed with side padding below that. Reading order
top-to-bottom **is** the priority order (§1). No sidebar, no tabs — everything a
glance needs is on one scroll.

```
┌──────────────────────────────────────────────────────────────┐
│ Header:  ◈ dacli dashboard        [● live · updated 16:10:03] │  ← AppHeader
│          mission control — the live agent swarm               │     (title, mark, ConnectionStatus)
├──────────────────────────────────────────────────────────────┤
│ OVERVIEW                                                       │  ← section label (h2)
│ ┌───────────┐ ┌───────────┐ ┌───────────┐   auto-fill grid    │
│ │ ProjectCard│ │ ProjectCard│ │ ProjectCard│  minmax(280,1fr) │  ← ProjectGrid
│ │ title/stage│ │           │ │           │                    │     └ ProjectCard × N
│ │ counts     │ │           │ │           │                    │        ├ StatusCounts
│ │ ▁▁▃▅ bar   │ │           │ │           │                    │        └ BurndownBar
│ │ pts caption│ │           │ │           │                    │
│ └───────────┘ └───────────┘ └───────────┘                    │
├──────────────────────────────────────────────────────────────┤
│ TASK BOARD + BURNDOWN   [project switcher ▾ if >1 project]    │  ← section label
│ ┌────────┬────────┬────────┬────────┐                         │
│ │ OPEN 12│ACTIVE 3│BLOCK. 1│ DONE 26│  four columns           │  ← TaskBoard
│ │ ▪ ▪ ▪  │ ▪      │ ▪      │ ▪ ▪ ▪  │  count-driven chips     │     └ BoardColumn × 4
│ └────────┴────────┴────────┴────────┘                         │
│ Burndown ▸ done 88.5 / remaining 31.0 pts · 4 unestimated     │  ← BurndownChart
│  ▁▂▄▇█  per-day landed points (sparkline/bars)                │
├──────────────────────────────────────────────────────────────┤
│ LIVE AGENT SWARM                              ● 2 running      │  ← section label + live count
│ ┌──────────────────────────────────────────────────────────┐ │
│ │ run   child   task  role   runtime  pid   uptime  last-act │ │  ← AgentSwarm (table)
│ │ ●01KY… a-nh…  131   design  claude 48213  10m 0s   20s ago │ │     └ AgentRow × N
│ └──────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────┘
```

The current page already renders **Overview** (project cards with burndown bar)
and the **Live agent swarm** (table). This spec **adds** the explicit
**Task board** (four status columns) and a **per-day burndown chart** driven by
the already-present `per_day` data, and factors the whole page into components.

---

## 5. Component tree

```
App
├─ AppHeader
│   ├─ BrandMark              (inline SVG hexagon-cluster; decorative, aria-hidden)
│   └─ ConnectionStatus       (live dot + "updated HH:MM:SS" + pending-events pill)
├─ OverviewSection
│   └─ ProjectGrid
│       └─ ProjectCard  × N
│           ├─ StatusCounts   (four dot+count chips)
│           └─ BurndownBar    (done/remaining stacked bar + points caption)
├─ BoardSection
│   ├─ ProjectSwitcher        (only rendered when projects.length > 1)
│   ├─ TaskBoard
│   │   └─ BoardColumn × 4    (open | active | blocked | done, count-driven)
│   └─ BurndownChart          (per_day landed points as bar sparkline)
├─ AgentSwarmSection
│   └─ AgentSwarm
│       └─ AgentRow × N       (run/child/task/role/runtime/pid/uptime/last-activity)
└─ (composables, not components)
    ├─ usePoll(url, intervalMs)   → { state, phase, error, lastOk, retry }
    ├─ useRelativeTime()          → ago(iso), duration(secs), clock tick
    └─ useStatusTheme()           → statusColor(status), count(counts, status)
```

Component contracts (props in / events out — all one-way, read-only):

| Component          | Props                                   | Emits | Notes |
|--------------------|-----------------------------------------|-------|-------|
| `AppHeader`        | `phase`, `generated`, `pendingEvents`   | —     | hosts ConnectionStatus |
| `ConnectionStatus` | `phase`, `generated`, `pendingEvents`   | —     | renders the live/lost tell (§6.2) |
| `ProjectGrid`      | `projects[]`, `phase`                   | —     | owns Overview empty/loading (§7) |
| `ProjectCard`      | `project`                               | —     | pure; no fetching |
| `StatusCounts`     | `counts`                                | —     | uses `count()` helper for missing keys |
| `BurndownBar`      | `burndown`                              | —     | 0-safe when total points = 0 |
| `ProjectSwitcher`  | `projects[]`, `selectedSlug`            | `update:selectedSlug` | client-only selection; not persisted server-side |
| `TaskBoard`        | `counts`                                | —     | four columns, count-driven |
| `BurndownChart`    | `burndown`                              | —     | empty when `per_day` is empty |
| `AgentSwarm`       | `agents[]`, `phase`                     | —     | owns swarm empty/loading (§7) |
| `AgentRow`         | `agent`                                 | —     | activity dot freshness from `last_activity` |

**No component fetches.** Only `App` (via `usePoll`) touches the network; data
flows down as props. This keeps every leaf component trivially testable with a
static prop and makes the four global states (§7) a property of `App`'s single
`phase`, not scattered per-component booleans.

**Project selection is client-only.** `ProjectSwitcher` picks which project the
Board + Burndown section renders; Overview always shows all projects. Selection
lives in `App` local state, defaults to the first project, and resets safely if
that slug disappears between polls (fall back to first, never render a stale
slug).

---

## 6. Data flow & the four states

### 6.1 The poller — `usePoll`

A single composable owns the fetch loop and is the **sole source of the app's
`phase`.** It fetches `/api/state` with `{cache: 'no-store'}` every 2000 ms and
exposes:

```ts
type Phase = 'loading' | 'live' | 'error'
interface Poll {
  state: Ref<DashboardState | null>  // last good snapshot, or null before first success
  phase: Ref<Phase>                   // drives every global state view
  error: Ref<string | null>          // last transport/HTTP message
  lastOk: Ref<number | null>         // epoch ms of last successful poll
  retry: () => void                   // manual re-poll (error state button)
}
```

Phase transitions:

- **`loading`** — before the first successful response (`state === null`).
- **`live`** — a poll succeeded; `state` holds the latest snapshot. Polling
  continues on the interval.
- **`error`** — a poll threw (network) or returned non-2xx. **The last good
  `state` is retained and kept on screen** (dimmed, per §6.2); the app does not
  blank out on a transient failure. Polling continues; the first success flips
  back to `live`.

This is exactly the current page's `try/catch/finally` loop, promoted to a
composable with an explicit tri-state instead of only a status string.

### 6.2 Connection status (header) — always visible

`ConnectionStatus` renders the current phase in the header, independent of the
per-section states:

- **loading:** muted dot (no pulse) + "connecting…".
- **live:** `--ok` dot **pulsing** + "live · updated `HH:MM:SS`" (local time of
  `generated`) + if `pending_events > 0`, a pill "· N pending event(s)". The
  pending-events pill uses `--muted`, not an alarm color — it is an information
  signal (unsynced events), not an error.
- **error:** `--danger` dot (static) + "connection lost: `<message>`" + a small
  **Retry** button that calls `poll.retry()`. When a prior good `state` exists,
  the three sections stay rendered but the whole `<main>` gets
  `opacity: .6; pointer-events: none` so a stale-but-present read is visually
  distinct from a live one — honesty about freshness, no fabricated data.

### 6.3 Per-section states (§7 details each)

The three sections each resolve one of four states from `phase` + their slice
of `state`:

| State     | Condition                                              |
|-----------|--------------------------------------------------------|
| `loading` | `phase === 'loading'` (first snapshot not yet in)      |
| `empty`   | `phase !== 'loading'` and the section's slice is empty |
| `error`   | `phase === 'error'` and no prior good `state` exists   |
| `live`    | `phase !== 'loading'` and the slice has data           |

Note that once any snapshot has arrived, a later poll failure does **not** send
sections to their `error` view — they stay `live`/`empty` on the retained
snapshot while the header shows the connection error and dims the page. Sections
only show their own `error` view on a **cold** failure (never any good data).

---

## 7. Per-surface state specifications

Every surface specifies all four states. Skeletons use `--panel` blocks with a
subtle shimmer that is **disabled under `prefers-reduced-motion`** (becomes a
static `--panel` block).

### 7.1 Overview (`OverviewSection` / `ProjectGrid` / `ProjectCard`)

- **loading:** 3 skeleton `ProjectCard`s — panel rectangles echoing the card
  shape (title bar, count row, bar). No text.
- **empty:** `projects.length === 0` → a single muted line, "no projects yet"
  (matches the current page copy), in a bordered empty panel.
- **error (cold):** danger-tinted panel: "couldn't load projects — `<message>`"
  + Retry. Only when no snapshot has ever arrived.
- **live:** auto-fill grid, `minmax(280px, 1fr)`, gap 12px. Each `ProjectCard`:
  - **H3** title (`title` or fallback `slug`); sub-line `slug · stage: <stage>`
    (em-dash when stage empty).
  - **StatusCounts:** four chips `● open N / ● active N / ● blocked N / ● done N`,
    each dot `statusColor(status)`, count via `count(counts,status)` so a missing
    key shows `0`. Chips wrap on narrow cards.
  - **BurndownBar:** a single stacked bar. `total = done_points +
    remaining_points`. Done segment `--done` at `done/total`, remaining
    `--active` at the complement. **When `total === 0`** (nothing estimated):
    render the empty `--border` track at full width and caption "no estimated
    points yet" — never divide by zero, never a NaN width.
  - **Caption:** `88.5 done · 31.0 remaining pts` (one decimal); append
    `· N unestimated` only when `unestimated > 0`.

### 7.2 Task board + Burndown (`BoardSection`)

Selected project = `ProjectSwitcher` choice (or the only project). All figures
below are that project's.

**TaskBoard** — four fixed columns in canonical order `open, active, blocked,
done`:

- **loading:** four skeleton column headers with a shimmer count block.
- **empty:** `total === 0` for the selected project → "no tasks in this project"
  in an empty panel spanning the four columns.
- **error (cold):** danger panel + Retry (only cold).
- **live:** each `BoardColumn` shows its status label (uppercase, colored dot)
  and count; below it, a compact run of `count` small `--panel` chips (capped —
  render at most e.g. 24 chips then "+K more" so a 500-task column can't blow
  out the layout; the count in the header is always the true total). A column
  with `0` renders its header and an em-dash placeholder, not a blank gap — the
  board's shape is fixed so the eye learns it.

> The board is **count-driven, not task-list-driven**, because `/api/state`
> carries per-status counts, not individual task rows. The chips visualize
> magnitude, not identity. If a future snapshot adds a task list, `BoardColumn`
> gains a `tasks?` prop and renders real cards; until then the spec forbids
> inventing task identities the snapshot doesn't carry.

**BurndownChart** — the `per_day` series as vertical bars (a bar sparkline):

- **loading:** a single skeleton bar strip.
- **empty:** `per_day.length === 0` → "nothing completed yet — burndown starts
  when the first task closes" (muted), plus the numeric caption still shows
  `done / remaining` from the bar totals.
- **live:** one bar per `per_day` entry, height ∝ `points`, x-order chronological
  (already sorted server-side — **do not re-sort**). Bars colored `--done`.
  Above/beside: the numeric summary `done_points` / `remaining_points` and
  `unestimated`. Each bar has an accessible label (`<day>: <points> pts`) via
  `<title>`/`aria-label`. This chart is **additive per-day landed points**, not
  a classic descending-ideal-line burndown — the snapshot gives landed-per-day,
  so the honest visualization is a "what closed each day" bar series, captioned
  as such, rather than a fabricated ideal line.

### 7.3 Live agent swarm (`AgentSwarmSection` / `AgentSwarm` / `AgentRow`)

- **loading:** skeleton table — header row + 3 shimmer body rows.
- **empty:** `agents.length === 0` → "no live agents" (current copy) in an empty
  panel. This is the **common resting state** and must read as calm/normal, not
  as an error — muted text, no danger color.
- **error (cold):** danger panel + Retry (only cold).
- **live:** a table, columns exactly:
  `run · child · task · role · runtime · pid · uptime · last activity`.
  - `run` → first 10 chars of `run_id`, **monospace**, prefixed by an
    **activity dot** whose color encodes freshness of `last_activity`:
    `--ok` (pulsing) if < 60s, `--muted` if < 5m, `--blocked` (static) if older
    — "still moving vs. possibly hung," the swarm's whole point. (Freshness
    thresholds are UI-local, documented here, not server fields.)
  - `child`, `task`, `role`, `runtime` → plain, em-dash when empty.
  - `pid` → **monospace**.
  - `uptime` → `duration(runtime_secs)` → `10m 0s` / `1h 4m`.
  - `last activity` → `ago(last_activity)` → "20s ago"; `title` = absolute UTC.
  - Rows keep the server's newest-first order; no client sort in v1 (a sortable
    header is a later enhancement, not a v1 state).

---

## 8. Accessibility

Non-negotiable, verified at build:

- **Semantics.** One `<h1>` (page title). Each section is a `<section>` with an
  `<h2>` and `aria-labelledby` pointing at it. The swarm is a real `<table>`
  with `<thead>`/`<th scope="col">`. The board is a list of labelled regions
  (`role="group"` + `aria-label="open — 12 tasks"`), not a table (it isn't
  tabular data). Cards are `<article>`s with the title as their accessible name.
- **Color is never the only signal.** Every status dot is paired with its text
  label ("open 12", not a bare dot). The burndown bar and chart carry text
  captions and `aria-label`s stating the numbers. The agent activity dot's
  meaning is also available as row text (`last activity` column + a
  `title="active <60s"` on the dot). A colorblind or screen-reader user loses
  nothing.
- **Contrast.** All text meets **WCAG AA** against its background
  (`--text`/`--muted` on `--bg`/`--panel`; the four status hues verified on both
  surfaces). Non-text UI (bars, dots, borders that carry meaning) meets **≥3:1**.
  A CI/lint check (§10) re-verifies token pairs; adding a foreground token
  without a passing check is a spec violation.
- **Keyboard.** Nothing critical is hover-only. Interactive elements — Retry
  button, `ProjectSwitcher`, and any future sortable header — are native
  `<button>`/`<select>` (or have `tabindex="0"` + key handlers), reachable in
  DOM order, with a visible **focus ring** (`--accent`, 2px, honoring
  `:focus-visible`). Tab order follows the reading order (§1). No focus trap;
  no positive `tabindex`.
- **Live regions.** `ConnectionStatus` is an `aria-live="polite"` region so a
  screen reader announces "connection lost" / "live" transitions without
  hijacking focus. The 2s poll does **not** re-announce on every tick — only
  phase changes and the pending-events count crossing zero are announced;
  routine count updates stay silent to avoid a chatty reader.
- **Motion.** `prefers-reduced-motion: reduce` disables the live pulse and
  skeleton shimmer (static dot / static block). No essential information is
  conveyed by motion alone — the pulse is redundant with color+text.
- **Zoom / reflow.** Layout is fluid to **400% zoom** and a **320px** viewport
  (§9) without horizontal scroll for the primary content; the wide swarm table
  is the one permitted horizontal-scroll region on the smallest screens, with
  `run`/`task` frozen as the identifying columns.

---

## 9. Responsive behavior (mobile → wide)

Breakpoints (min-width), mobile-first:

| Range            | Overview grid            | Task board                | Swarm table                          |
|------------------|--------------------------|---------------------------|--------------------------------------|
| **< 640px** (mobile) | 1 column                 | 2×2 stacked columns       | horizontal-scroll table; `run`+`task` sticky |
| **640–1024px** (tablet) | auto-fill `minmax(280,1fr)` (1–2 cols) | 4 columns, condensed | full table, no scroll                |
| **> 1024px** (desktop/wide) | auto-fill (2–4 cols), max content 1280px centered | 4 columns, full        | full table, comfortable padding      |

Rules:

- **The section order never changes** across breakpoints — Overview, Board,
  Swarm, top to bottom, always (priority = order, §1). No off-canvas nav, no
  collapsing sections behind accordions on mobile; the whole console is one
  scroll on every size.
- Page side padding steps: `16px` mobile → `24px` tablet → `32px` desktop.
- Card min-width `280px` is the reflow driver for Overview (the existing
  `auto-fill minmax(280px,1fr)` already does this — keep it).
- On mobile the board's four columns become a **2×2 grid** (open/active top,
  blocked/done bottom) rather than a 4-wide squeeze, preserving the canonical
  order left-to-right, top-to-bottom.
- The swarm table is the only element allowed to scroll horizontally on mobile;
  it does so inside its own `overflow-x:auto` wrapper with the two identifying
  columns (`run`, `task`) `position: sticky; left: 0` so a row stays legible
  while scrolling the metrics.
- Touch targets (Retry, switcher, any sort header) are **≥44×44px** on
  touch-pointer devices.

---

## 10. Verification checklist (definition of done for the build)

The frontend engineer's build is done against this spec when:

- [ ] `GET /api/state` is the only data source; no field outside §0 is bound; no
      mutation call exists anywhere in the UI.
- [ ] All four states render for **each** of the three surfaces (12 states)
      exactly as §6–§7 specify, including the `total === 0` burndown and the
      empty `per_day` chart, and the calm (non-error) empty-swarm state.
- [ ] `phase` is owned solely by `usePoll`; a dropped poll retains the last good
      snapshot and dims `<main>` rather than blanking (cold-only section errors).
- [ ] Color tokens equal the §3.1 values (identical to `static/index.html`);
      status→color flows through one helper; no hardcoded status hex in
      components.
- [ ] Contrast check passes AA for every foreground/background token pair on both
      `--bg` and `--panel`; non-text UI ≥3:1.
- [ ] Keyboard: every interactive control is reachable, has a visible
      `:focus-visible` ring, and tab order follows reading order; nothing is
      hover-only.
- [ ] `prefers-reduced-motion` disables pulse + shimmer with static fallbacks.
- [ ] Responsive: verified at 320px, 640px, 1024px, and ≥1280px, and at 400%
      zoom, with no unexpected horizontal scroll outside the swarm-table wrapper.
- [ ] Screen-reader pass: sections labelled, table has header scope, status is
      conveyed by text (not color alone), `ConnectionStatus` announces phase
      changes politely without per-tick chatter.
- [ ] Ships as one `go:embed`-able self-contained bundle; no runtime deps beyond
      `/api/state`; the 2000ms poll cadence is preserved.

---

## 11. Non-goals (this UI)

- **No write actions.** No spawn/kill/claim buttons — the dashboard is a
  read-only projection (server doctrine). Control stays in the CLI.
- **No historical/time-travel view.** The UI shows the current snapshot;
  history is git's job (top-level DESIGN.md §2). `per_day` is the only
  time-series and it comes pre-computed in the snapshot.
- **No auth / multi-user.** Localhost, single operator, same as the current
  server (`127.0.0.1`).
- **No theming beyond the one dark system.** `color-scheme: dark` only; a light
  theme is explicitly out of scope for a mission-control console.
- **No per-task drill-down** until the snapshot carries task rows (§7.2). The
  spec forbids fabricating task identities the data doesn't include.

---

*Cross-refs: server & data contract —
[`../dashboard.go`](../dashboard.go); current single-file implementation this
replaces — [`../static/index.html`](../static/index.html); proposal origin —
`docs/PROPOSALS.md` P8; mark decision —
[[d-124-dashboard-header-inlines-the-mark-svg-plus-a-data-uri-favicon-instead-of]].*
