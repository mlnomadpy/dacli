---
id: f-338-complete-on-branch-dacli-338-gosec-curated-g104-g301-g302-g306-g702-g703
kind: note
note_kind: finding
created: 2026-08-10T19:27:18Z
created_by: a-fixer-k0d8zr
about: "[[338]]"
severity: major
---
# 338 complete on branch dacli/338-...: gosec (curated: G104/G301/G302/G306/G702/G703/G705 excluded, reasoned) and errorlint enabled in .golangci.yml, 0 issues
Commit efd6a07 (a-fixer-k0d8zr). All 4 acceptance criteria met.

(1) gosec: enabled with excludes G104/G301/G302/G306 (already-established reasoning kept) plus three NEW taint-based rule IDs this gosec version reports that the prior exclusion list did not cover: G702 (command-injection twin of G204), G703 (path-traversal twin of G304), G705 (XSS via taint). Measured 15 findings after the original 4 excludes; disposition: G114 (dashboard.go:95, http.Serve with no timeouts) FIXED — wrapped in http.Server{ReadHeaderTimeout/ReadTimeout/WriteTimeout/IdleTimeout}. G702 x2, G703 x7, G705 x4 verified false-positive and EXCLUDED with reasoning in .golangci.yml (argv-slice subprocess invocation, workspace-internal/validated paths, text/plain-only response bodies on a 127.0.0.1-only listener). G122 x1 (skills.go:242, WalkDir TOCTOU) REFUSED via targeted //nolint:gosec — a local operator-to-operator directory copy crossing no privilege boundary; Go's os.Root-scoped fix needs Go 1.24, repo pins 1.22. Final: golangci-lint run -> 0 issues.

(2) errorlint: all 39 sites fixed — 29 fmt.Errorf %v->%w conversions (acceptance.go, catalog.go, collab.go, ghmirror.go x9, project.go x9, release.go, driver_test.go x4, selfreport.go, shortcuts.go x2), 8 sentinel-error == / != comparisons converted to errors.Is (agentid_test.go x6, teamops.go, spm_test.go), 2 type assertions converted to errors.As (skills_test.go, store/readiness.go's isNotFound helper).

(3) internal/clikit/clikit_test.go: TestExitCodeSurvivesWrapping — table-driven over unwrapped/wrapped-once/wrapped-twice, asserts ExitCode(Refusedf(...)) stays 3 through fmt.Errorf("...: %w", ...) chains of depth 1 and 2. Red-green verified by hand: mutated the two wrapped cases from %w to %v, reran -> both failed with 'ExitCode(...) = 1, want 3 (refused, never retried)', confirming the test actually exercises the errors.As chain-walk rather than passing vacuously; reverted to %w, green again.

(4) both linters enabled in .golangci.yml (not deferred/documented-omission).

PROOF: go build ./... clean, go vet ./... clean, gofmt -l . clean, golangci-lint run (v2.12.2, matching CONTRIBUTING.md pin) -> 0 issues. go test -exec 'env -u DACLI_AGENT' ./... all green (every package ok; DACLI_AGENT-unset convention needed because this is a live dogfood session with an ambient agent token, same as noted in prior cycles' findings).

Also filed dacli bug (github.com/mlnomadpy/dacli/issues/427): 'dacli commit --task 338' derived a bogus file claim by scanning the task body's prose for a slash-separated substring ('G104/G301/G302/G306', literally quoted from the acceptance criteria) and treated it as a claimed-path allowlist, though no real claim was ever set on this task. Worked around with --force; every touched file is exactly what this task's acceptance criteria require.

Owner: dacli accept 338 (task check is gated to a-root). PR-first is off — branch dacli/338-stage-the-deferred-linters-gosec-with-a-curated-rule-list-and-errorlint-against is ready for accept + integrate/merge --task 338.
