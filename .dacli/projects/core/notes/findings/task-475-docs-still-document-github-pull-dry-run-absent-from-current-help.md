---
id: f-task-475-docs-still-document-github-pull-dry-run-absent-from-current-help
kind: note
note_kind: finding
created: 2026-08-19T12:35:07Z
created_by: a-fixer-xfw8k7
about: "[[t-01M0CX031NDQ5PQ8VRX1PQNWXE]]"
severity: major
---
# Task 475 docs still document github pull --dry-run absent from current help
Fresh forward test on 2026-08-19 ran /tmp/dacli-current-bin github pull --help, which reports exact usage 'dacli github pull <project>' (no --dry-run), while the playbook docs/OPERATOR_PLAYBOOK.md:19 and skill references critical-path-github.md:3 and github-landing.md:53 document 'github pull <project> --dry-run'. dacli github pull

Inbound: adopt human-authored issues as local tasks (--dry-run previews the adoptions)

dacli github pull <project>
Needs an rw grant (a --dry-run, where supported, is a read). exited 2. This prevents claiming the task's exact-current-help acceptance; reconcile the CLI Usage contract or change the canonical preview flow before owner acceptance.
