---
id: d-183-exclude-node-modules-via-npm-postinstall-go-mod-marker-not-a-nested-ui-go
kind: note
note_kind: decision
created: 2026-08-04T12:14:50Z
created_by: a-maintainer-y2t0se
about: "[[183]]"
---
# 183: exclude node_modules via npm postinstall go.mod marker, not a nested ui/ go.mod
## Chose
183: exclude node_modules via npm postinstall go.mod marker, not a nested ui/ go.mod
## Rejected
Add go.mod to ui/ (standard nested-module exclude), or move ui/dist out of ui/ so the embed survives a nested ui/ module
## Because
Verified on go1.26 that go build/test/vet ./... DOES descend into node_modules and a stray .go there breaks the build (not local-dev-only: CI runs npm ci then go vet/test ./... with node_modules present). A go.mod at ui/ makes it a nested module, which excludes node_modules AND breaks //go:embed all:ui/dist in dashboard.go with 'cannot embed directory: in different module' (reproduced). Moving dist out of ui/ would fix that but rewrites the heavily-documented dist/embed/goreleaser-clean-tree convention across ~5 files. A lone go.mod inside node_modules is a nested-module boundary the parent ./... walk skips (verified) without touching the embed. node_modules is gitignored so it cannot be committed; an npm postinstall regenerates the marker after every install/ci and, being under gitignored node_modules, never dirties the tree so goreleaser stays happy.
