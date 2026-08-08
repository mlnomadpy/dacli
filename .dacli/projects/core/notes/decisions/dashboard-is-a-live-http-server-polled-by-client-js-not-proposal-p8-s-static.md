---
id: d-dashboard-is-a-live-http-server-polled-by-client-js-not-proposal-p8-s-static
kind: note
note_kind: decision
created: 2026-07-23T19:23:04Z
created_by: a-qr6b08292c
about: [[122]]
---
# dashboard is a live HTTP server polled by client JS, not proposal P8's static-HTML regenerate
## Chose
dashboard is a live HTTP server polled by client JS, not proposal P8's static-HTML regenerate
## Rejected
generate a static dashboard.html snapshot on each run, like P8 in docs/PROPOSALS.md described
## Because
the acceptance criteria requires a running loop's agents to appear live without re-invoking a generate step; a tiny stdlib net/http server (127.0.0.1, ephemeral port unless --port pins one) serving an embedded page that polls /api/state every 2s meets that with zero external dependencies and no new process-lifecycle concerns beyond Ctrl+C
