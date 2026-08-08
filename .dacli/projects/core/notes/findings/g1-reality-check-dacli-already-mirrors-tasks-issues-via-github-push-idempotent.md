---
id: f-g1-reality-check-dacli-already-mirrors-tasks-issues-via-github-push-idempotent
kind: note
note_kind: finding
created: 2026-07-22T16:43:12Z
created_by: a-root
severity: moderate
---
# G1 reality check: dacli ALREADY mirrors tasks->issues via 'github push' — idempotent (marker search), backlinks the issue number onto the task, closes the issue on done, and is disclosure-gated (github link --allow-public + live visibility re-check). The ONLY G1 gap is per-status LABELS (it closes on done but sets no status:doing/blocked label). The genuine GitHub gaps are G2 (decisions->Discussions), G3 (verify->PR review comments), and inbound sync (github sync/pull are planned() stubs)
