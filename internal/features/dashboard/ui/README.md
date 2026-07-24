# dacli dashboard SPA

The Vue 3 source for the dashboard `dacli dashboard` serves. It builds to a
single self-contained `dist/index.html` (JS + CSS inlined by
`vite-plugin-singlefile`) that Go embeds into the binary.

## How embedding works

`internal/features/dashboard/dashboard.go` embeds the built bundle:

```go
//go:embed all:ui/dist
var spaDist embed.FS
```

`dacli dashboard` serves `ui/dist/index.html` at `/` when that file is present,
and falls back to the legacy self-contained page (`static/index.html`,
`//go:embed static/index.html`) when it is not. Both poll the same
`/api/*` endpoints, so the dashboard is fully functional either way.

`ui/dist/` is gitignored **except** `ui/dist/.gitkeep`: the built `index.html`
is generated, never committed, but the directory must stay tracked so
`go:embed all:ui/dist` always has a target and `go build` succeeds on a fresh
checkout that has not run the frontend build. `vite.config.ts` sets
`emptyOutDir: false` so a build never deletes `.gitkeep` (which would dirty the
tree and make `goreleaser release` refuse to run).

## Build (produces the embedded bundle)

```sh
cd internal/features/dashboard/ui
npm ci
npm run build          # writes dist/index.html
```

CI runs this before `go build`/`go test` (`.github/workflows/ci.yml`) and the
release workflow runs it before goreleaser (`.github/workflows/release.yml`), so
released binaries always embed the current SPA. If you edit the UI, rebuild
before `go build` or the embedded bundle will be stale (or absent → the legacy
fallback page).

## Dev mode (hot reload against the live API)

Run the Go dashboard and the Vite dev server side by side. Vite serves the SPA
with HMR on its own port and proxies `/api` to the Go server (see
`vite.config.ts`):

```sh
dacli dashboard --port 8787   # terminal 1: the Go API + embedded page
npm run dev                   # terminal 2: Vite dev server, /api proxied to :8787
```

Open the URL Vite prints (not the Go one) to edit the UI with live reload while
the real endpoints back it.

## Test / lint

```sh
npm run test:unit      # Vitest component + store tests
npm run lint
npm run type-check
```
