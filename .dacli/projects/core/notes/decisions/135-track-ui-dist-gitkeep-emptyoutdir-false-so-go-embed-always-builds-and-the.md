---
id: d-135-track-ui-dist-gitkeep-emptyoutdir-false-so-go-embed-always-builds-and-the
kind: note
note_kind: decision
created: 2026-07-24T01:35:02Z
created_by: a-qn95ewfptr
about: [[135]]
---
# 135: track ui/dist/.gitkeep + emptyOutDir:false so go:embed always builds and the release tree stays clean
## Chose
135: track ui/dist/.gitkeep + emptyOutDir:false so go:embed always builds and the release tree stays clean
## Rejected
hand-commit the built 93KB dist/index.html and point go:embed at it (the alternative task 134 deferred)
## Because
a committed bundle silently drifts from src on every UI change (the d-134 objection). Instead: gitignore ui/dist/* but track a .gitkeep placeholder, embed via '//go:embed all:ui/dist' (the all: prefix is required to match a dotfile-only dir — verified empirically), and set vite emptyOutDir:false so a build never deletes the tracked .gitkeep. Result: go build succeeds on a fresh checkout (embeds .gitkeep, serves legacy fallback), the real index.html is gitignored so npm build in CI leaves a clean tree (goreleaser refuses a dirty repo), and the SPA is regenerated — never committed — on every CI/release run.
