---
id: f-130-goreleaser-check-release-literally-blocked-by-sandbox-verified-brews
kind: note
note_kind: finding
created: 2026-07-23T22:33:06Z
created_by: a-53n5n76v8q
about: [[130]]
severity: minor
---
# 130: goreleaser check/release literally blocked by sandbox; verified brews deprecation fix via direct source read + go build/test
Same sandbox limitation as f-128 (task 128): invoking the installed 'goreleaser' binary (2.17.0 at /opt/homebrew/Cellar/goreleaser/2.17.0) or any new shell utility (strings, mkdir, rm, env, printf) requires human approval, which this headless session cannot grant -- so 'goreleaser check' and 'goreleaser release --snapshot --clean' could not literally be executed. Verified the fix a different way: fetched the exact installed version's source (go get github.com/goreleaser/goreleaser/v2@v2.17.0 into a throwaway module, since only 'go' subcommands are permitted) and read internal/pipe/brew/brew.go:60 -- Default() calls deprecate.Notice(ctx, "brews") unconditionally for any non-empty brews: config, which is exactly the warning that fails 'goreleaser check' (cmd/check.go:62-65 exits 2 when ctx.Deprecated). The new homebrew_casks: config (.goreleaser.yaml:61-70) uses only current, non-deprecated HomebrewCask fields (pkg/config/config.go:216-251), so no deprecate.Notice call is reachable for it. go build ./... and go test ./... both green on the unrelated Go code (unchanged). Owner should still run 'goreleaser check' and 'goreleaser release --snapshot --clean' once outside this sandbox to get the literal exit-code confirmation before tagging v0.1.0.
