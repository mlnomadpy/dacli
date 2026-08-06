---
id: f-294-complete-on-branch-dacli-294-dry-run-added-to-all-six-remote-mutating
kind: note
note_kind: finding
created: 2026-08-04T20:27:54Z
created_by: a-maintainer-hrnt6j
about: "[[294]]"
severity: major
---
# 294 complete on branch dacli/294-...: --dry-run added to all six remote-mutating github commands, derived from the real code path
Commit ccce388 (a-maintainer-hrnt6j). Both acceptance criteria met. (1) push, sync, pull, project, release, codeowners all accept --dry-run and print what they would create/adopt/close and write NOTHING (no mutating gh call, no local file mutation). sync inherits it free — it delegates to cmdPull then cmdPush with the same args, both of which read --dry-run. (2) Preview is the SAME code path, not a parallel description: each command runs its real read+decide loop and only the terminal write is swapped for a 'would ...' print. Finding-comment preview reuses a new shared decision helper findingsToPost() that mirrorFindings() (real) also calls, so they can never drift. release --dry-run skips the rw-grant check (writes nothing) but the disclosure gate still runs for push/project so a preview that would be refused reports the refusal. Files: internal/features/ghmirror/{ghmirror.go,project.go,release.go,codeowners.go}, docs/GITHUB.md. VERIFICATION: 6 new tests in internal/features/ghmirror/dryrun_test.go drive the real cmdPush/cmdPull/cmdProject/cmdRelease/cmdCodeowners through their actual code paths (only gh + filesystem stubbed); each asserts zero mutating gh calls / no local write AND the 'would' preview text. Confirmed by git-stashing the 4 source files: all 6 fail before the change (pull 'created 1 task', project actually created a board, others 'unknown flag: --dry-run') and pass after. Full go build ./... && go test ./... && go vet ./... && gofmt -l internal/ all clean. Binary smoke was sandbox-blocked (exec restricted to the worktree; running dacli there would touch the shared .dacli), so verification is via the unit tests that exercise the real command functions end-to-end. Owner: dacli accept 294 then integrate/merge --task 294.
