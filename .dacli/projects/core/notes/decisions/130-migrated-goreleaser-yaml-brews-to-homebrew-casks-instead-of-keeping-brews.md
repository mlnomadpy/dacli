---
id: d-130-migrated-goreleaser-yaml-brews-to-homebrew-casks-instead-of-keeping-brews
kind: note
note_kind: decision
created: 2026-07-23T22:32:55Z
created_by: a-53n5n76v8q
about: [[130]]
---
# 130: migrated .goreleaser.yaml brews to homebrew_casks instead of keeping brews with fewer fields
## Chose
130: migrated .goreleaser.yaml brews to homebrew_casks instead of keeping brews with fewer fields
## Rejected
trimming brews to only its non-legacy fields (drop directory/license/install/test) while keeping the brews: key
## Because
read the installed goreleaser v2.17.0 source directly (internal/pipe/brew/brew.go:60, module cache at /Users/tahabsn/go/pkg/mod/github.com/goreleaser/goreleaser/v2@v2.17.0): Default() calls deprecate.Notice(ctx, "brews") unconditionally whenever len(ctx.Config.Brews) > 0, regardless of which sub-fields are set -- pkg/config/config.go:186-188 and internal/pipe/brew/doc.go both mark the whole Homebrew/brew package 'Deprecated: in favor of HomebrewCask'. Only removing the brews: key entirely and using homebrew_casks: (pkg/config/config.go:1347, internal/pipe/cask) clears the warning. Cask supports darwin+linux archives via a binaries: stanza (no install/test Ruby DSL, license has no effect per config.go:243-244), so the new config sets name/repository/homepage/description/binaries: [dacli] and omits directory (defaults to "Casks") and license.
