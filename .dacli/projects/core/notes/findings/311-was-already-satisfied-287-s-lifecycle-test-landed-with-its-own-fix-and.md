---
id: f-311-was-already-satisfied-287-s-lifecycle-test-landed-with-its-own-fix-and
kind: note
note_kind: finding
created: 2026-08-08T12:13:05Z
created_by: a-root
about: "[[311]]"
---
# 311 was already satisfied: 287's lifecycle test landed with its own fix and passes on main
The stray 33-line test found in main's working tree during the 2026-08-06 audit was 287's own work, mid-flight. 287 merged in PR #385 carrying both the template fix and the test, so internal/prompts/prompts_test.go already declares TestROAndRWPreamblesDescribeDifferentLifecycles and it passes. Re-applying the saved patch produced a duplicate declaration, which is how this was confirmed. No code change needed.
