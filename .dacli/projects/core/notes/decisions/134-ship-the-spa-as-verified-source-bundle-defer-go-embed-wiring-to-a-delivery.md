---
id: d-134-ship-the-spa-as-verified-source-bundle-defer-go-embed-wiring-to-a-delivery
kind: note
note_kind: decision
created: 2026-07-24T01:13:38Z
created_by: a-j846nahs42
about: [[134]]
---
# 134: ship the SPA as verified source+bundle, defer go:embed wiring to a delivery task
## Chose
134: ship the SPA as verified source+bundle, defer go:embed wiring to a delivery task
## Rejected
hand-commit the 93KB built dist/index.html and repoint dashboard.go's go:embed at it now
## Because
no CI/goreleaser step builds the UI (grep of .github/workflows + .goreleaser.yaml finds no npm/node/vite), and ui/.gitignore excludes dist — so a hand-committed bundle would silently drift from src on every UI change with nothing to regenerate or diff it. Go CI runs gofmt+go test, never npm build. The legacy self-contained static/index.html + /api/state keep the dashboard fully functional today; npm run build already emits the single-file inlined bundle (dist/index.html, 93KB) per DESIGN.md §2. Wiring it belongs in a task that first adds a UI build step to CI/release so the embedded artifact is generated, not committed by hand.
