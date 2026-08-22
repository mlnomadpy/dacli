---
id: f-task-claim-omits-runtime-adapter-capability-storage-path
kind: note
note_kind: finding
created: 2026-08-22T22:01:31Z
created_by: a-fixer-76xc6t
about: "[[t-01M0CZANEM3TFEMGTW3NTNXGXM]]"
severity: major
---
# Task claim omits runtime adapter capability storage path
dacli commit refused exit 3: internal/store/runtimefiles.go is outside claim [internal/features/execution, internal/cli]. The provider-neutral claude-print-v1 capability is defined in internal/store/runtimefiles.go:284, while execution consumes it. Do not force or broaden the claim silently; owner recovery is to expand/transfer the claim to internal/store, then commit the reviewed four-file diff with the stated mutation proof.
