---
id: f-135-complete-spa-embedded-via-go-embed-all-ui-dist-with-legacy-fallback
kind: note
note_kind: finding
created: 2026-07-24T01:35:12Z
created_by: a-qn95ewfptr
about: [[135]]
severity: moderate
---
# 135 complete: SPA embedded via go:embed all:ui/dist with legacy fallback; frontend build wired into CI + release
Commit ae4331f. dashboard.go: '//go:embed all:ui/dist' + indexPage() serves ui/dist/index.html when a build produced it, else the legacy static/index.html (dashboard.go:34-63). ui/dist/.gitkeep tracked (ui/.gitignore: 'dist/*' + '!dist/.gitkeep'); vite.config.ts emptyOutDir:false keeps it. Verified: npm ci && npm run build -> dist/index.html 93KB; gofmt -l . clean; go vet ./... clean; go test ./... all green (dashboard test asserts GET / == resolved indexPage()). Confirmed empirically that '//go:embed all:<dir>' compiles when the dir holds only a dotfile (scratch pkg built OK), so go build succeeds on a fresh checkout that never ran npm (serves fallback). CI (.github/workflows/ci.yml test + cross-compile) and release.yml now run setup-node + 'npm ci && npm run build' before go build/test/goreleaser; built index.html is gitignored so the release tree stays clean. Could not smoke-run the built binary (headless sandbox blocks arbitrary exec); handler verified via httptest. Dev mode documented in ui/README.md + vite proxy (/api -> :8787).
