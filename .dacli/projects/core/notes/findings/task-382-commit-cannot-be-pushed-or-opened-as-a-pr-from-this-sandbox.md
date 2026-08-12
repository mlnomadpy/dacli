---
id: f-task-382-commit-cannot-be-pushed-or-opened-as-a-pr-from-this-sandbox
kind: note
note_kind: finding
created: 2026-08-12T18:28:26Z
created_by: a-codex-maintainer-j8jbvt
about: "[[382]]"
severity: major
---
# task 382 commit cannot be pushed or opened as a PR from this sandbox
Commit 45c7c2d contains the implementation. Required 'dacli github push core 382 --dry-run' failed because gh could not connect to api.github.com; 'dacli push --task 382' failed because github.com could not resolve. GitHub issue 477 likewise could not be read remotely. Manual next step: push branch dacli/382-fix-status-probes-that-finalize-live-runs-when-process-visibility-is-restricted and run dacli pr --task 382 --with-verdicts --auto from a network-enabled context.
