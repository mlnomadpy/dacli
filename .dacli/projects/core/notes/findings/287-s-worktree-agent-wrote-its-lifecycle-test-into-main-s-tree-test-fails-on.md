---
id: f-287-s-worktree-agent-wrote-its-lifecycle-test-into-main-s-tree-test-fails-on
kind: note
note_kind: finding
created: 2026-08-06T09:11:25Z
created_by: a-root
about: "[[311]]"
origin: internal/prompts/prompts_test.go
---
# 287's worktree agent wrote its lifecycle test into main's tree; test fails on merged trunk
The escaped 33-line test (TestROAndRWPreamblesDescribeDifferentLifecycles) is preserved verbatim in the session scratchpad patch 287-stray-test.patch and in PR #385's history context. It asserts the ro preamble never mentions committing; on current trunk the ro render still contains 'commit' (likely reintroduced by a template merge), so landing it means aligning template and test, not just adding the file.
