---
id: f-250-complete-on-branch-dacli-250-a-role-nothing-catches-it-commit-381acc4
kind: note
note_kind: finding
created: 2026-08-04T11:48:32Z
created_by: a-maintainer-0fh816
about: "[[250-a-role-can-declare-grant-rw-and-a-read-only-runtime-and-nothing-catches-it]]"
severity: major
---
# 250 complete on branch dacli/250-a-role...-nothing-catches-it (commit 381acc4)
All 3 acceptance criteria met, committed 381acc4 by a-maintainer-0fh816. (1) spawn refuses rw on a no-write runtime: sandboxFor (execution.go:1019) now mirrors its ro branch — for a non-ro grant, !store.RuntimeWritable(rt) && !--cooperative => Refusedf exit 3 ('runtime X grants no write tool...'). (2) doctor flags mismatches: cmdDoctor (insight.go, role loop ~line 1081) loads each role's runtime and reports 'grant-runtime-mismatch' both ways — rw grant on a non-writable runtime, and ro grant on a runtime with no read-only sandbox. (3) junior corrected: .dacli/roles/junior.md runtime cc -> cc-rw (write-capable: cc-rw declares Edit,Write in invoke_args). New predicates store.RuntimeWritable / RuntimeEnforcesRO (runtimefiles.go) are the single source both the spawn gate and doctor read: writable = the invoke-args --allowedTools allowlist names a write tool (Edit/Write/MultiEdit/NotebookEdit); a runtime pinning NO allowlist stays writable so generic-exec is never falsely refused. Tests: TestRuntimeWritable/TestRuntimeEnforcesRO (store), TestSandboxFor + a new TestSpawnRefusals case (execution), TestDoctorFlagsGrantRuntimeMismatch (cli) — all verified to FAIL before the change (spawn test showed the exact task-183 burn: run 'ok', child wrote 0 events) and PASS after. Full suite green with -exec 'env -u DACLI_AGENT' (the one non-stripped catalog failure is the known DACLI_AGENT session leak, unrelated). PR-first is off: owner to 'dacli accept' this slug then integrate the branch.
