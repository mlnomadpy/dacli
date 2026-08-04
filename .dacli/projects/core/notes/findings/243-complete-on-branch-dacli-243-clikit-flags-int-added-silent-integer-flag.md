---
id: f-243-complete-on-branch-dacli-243-clikit-flags-int-added-silent-integer-flag
kind: note
note_kind: finding
created: 2026-08-04T10:20:28Z
created_by: a-maintainer-0a7fqr
about: "[[243]]"
severity: moderate
---
# 243 complete on branch dacli/243-...: clikit.Flags.Int added, silent integer-flag sites routed through it
Commit ea559e7 by a-maintainer-0a7fqr. Acceptance met: (1) clikit gains an Int helper — Flags.Int(k, def) (internal/clikit/clikit.go:170) parses a flag as base-10 int, returns def when absent/empty, and returns a usage error (exit 2) on garbage rather than silently zeroing. (2) The discarded-error integer-FLAG sites now use it: execution.go resolveLaunch budget+timeout (378/380), cmdSupervise max-turns (851), cmdRunsPrune keep, cmdLogs tail, cmdKill grace, cmdWait interval+timeout; verify.go require/timeout/budget; briefing.go budget x2 (Atoi + Sscanf); collab.go limit (Sscanf); insight.go parallel (Sscanf); teamops.go wip (Sscanf). This collapses the four ad-hoc parses (Atoi-dropped, Sscanf-ignored, Atoi-guarded-by-n>0) into one. Motivating bug fixed: spawn --timeout 30s previously discarded the Atoi error and ran on the 300s default; now it refuses with 'must be an integer, got "30s"'. Left as-is (not discarded-error sites): planning --n (range check), dashboard --port, dashboard parseLimit (HTTP 400) already fail loudly; mcp/tools.go i() reads an MCP args map, not *Flags; store/procmon/state/ghmirror/gates parse internal file records, not user flags. New test TestFlagsInt (clikit_test.go) covers absent/valid/garbage. Verified: go build ./... clean, go vet clean, gofmt -l internal/ clean, go test ./... green EXCEPT pre-existing catalog TestCatalogRefuses... which fails identically on a stashed clean tree (DACLI_AGENT env leak, catalog has no TestMain to clear it) — not a regression. PR-first off: owner to run dacli accept 243 then integrate/merge.
