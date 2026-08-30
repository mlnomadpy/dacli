# Dashboard

`dacli dashboard` is the local, read-only projection over the same durable
workspace used by the CLI and MCP server. It does not keep a second database,
authorize transitions, or mutate tasks. Its job is narrower: make the current
delivery state understandable at a glance, then link an operator to the exact
evidence behind it.

```bash
dacli dashboard
# or pin the loopback port
dacli dashboard --port 8787
```

The command prints the local URL and serves only on loopback. Keep the terminal
open while using the page; press Ctrl+C to stop it.

## Read the page from top to bottom

### Operator pulse

The first panel answers the three questions that should precede any new wave:

1. **What is next?** The first unfinished task on the recorded critical path,
   its estimate, project duration, and path sequence.
2. **What needs attention?** Existing burn alerts, blocked tasks, unhealthy
   agents, pending events, and roles at their WIP cap. Each signal links to the
   surface that explains it.
3. **What is moving?** Running-agent, active-task, open-task, and project counts.

“No recorded blockers” means the observed signals are within their recorded
policy. It is not a guarantee that unrecorded external state is healthy.

### Delivery

The project cards and task board show status and estimated work. Select a
project to keep its board and dependency graph together. The graph highlights
the computed critical path and reports when the schedule cannot be computed;
it never invents a path from incomplete data.

### Agents and spend

Burn compares measured run intensity with the calibrated ceiling. The swarm
view shows each recorded worker's task, role, runtime, state, last activity,
and resource observations. Transcript and diff links are read-only evidence
views backed by that run's durable record.

`blocked`, `stalled`, and `silent` agents appear in the attention summary. The
dashboard does not kill, retry, or replace them. Use the CLI or MCP workflow so
the action remains governed and recorded.

### Team

The roster answers who may perform work: grant, runtime, model, scope, skills,
active workers, and WIP policy. A role at its cap is an explanation for why a
new worker should not be scheduled, not an invitation to bypass the cap in the
browser.

## Freshness and failure states

The page polls the local API. The header reports whether the current snapshot
is loading, live, stale, or unavailable:

- **Loading** — no complete snapshot has arrived yet.
- **Live** — the last poll completed successfully.
- **Stale** — the last good snapshot remains visible but a later poll failed.
- **Unavailable** — no trustworthy snapshot can be shown.

A stale snapshot is deliberately dimmed and inert. The generated timestamp is
the observation time, not proof that GitHub, a provider, or a worker has not
changed since then.

## Authority and security boundary

The HTTP server binds to `127.0.0.1` and accepts `GET` requests from recognized
loopback hosts. The API exposes workspace projections and bounded run evidence;
it has no write route. Use `dacli capabilities --json` to negotiate the live
CLI/MCP surface before an agent relies on a command.

The dashboard does not replace:

- `dacli explain --project <slug> --json` for sourced routing and next-action
  evidence;
- `dacli loop status --project <slug> --json` for durable recovery state;
- `dacli pr diagnose --task <ref> --json` for current GitHub/CI diagnosis; or
- the explicit verify, review, accept, and ship gates.

## Responsive and accessible use

The same information order is preserved on desktop and mobile. The section
rail remains keyboard-focusable, all status colors have text labels, and the
page honors reduced-motion preferences. Use the skip link to move directly to
the dashboard content.

## Troubleshooting

- **The page is empty:** run the command from a repository adopted into the
  intended dacli workspace and inspect the printed URL.
- **The page is stale:** keep the last snapshot as evidence, inspect the error
  in the header, and retry. Do not treat stale data as launch authority.
- **A run link returns 404:** the run may have been reconciled or its durable
  record may be unavailable. Use the CLI's run and journal views.
- **The released binary shows the legacy page:** release builds embed the Vue
  bundle. Source builds that skip the frontend build can fall back to the
  legacy self-contained page; see the
  [frontend README](https://github.com/mlnomadpy/dacli/blob/main/internal/features/dashboard/ui/README.md).

The screenshots on the landing page use the representative test fixture in the
repository. They demonstrate layout and states, not production usage metrics.
