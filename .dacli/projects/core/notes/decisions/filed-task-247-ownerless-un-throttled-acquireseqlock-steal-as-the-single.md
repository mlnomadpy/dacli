---
id: d-filed-task-247-ownerless-un-throttled-acquireseqlock-steal-as-the-single
kind: note
note_kind: decision
created: 2026-08-04T00:43:50Z
created_by: a-go-auditor-7f03df
about: "[[244]]"
---
# Filed task 247 (ownerless/un-throttled acquireSeqLock steal) as the single highest-value evidence-based change
## Chose
Filed task 247 (ownerless/un-throttled acquireSeqLock steal) as the single highest-value evidence-based change
## Rejected
Re-filing any SYSTEM_AUDIT_2026-07-27 item, or the acquireSeqLock deadline being a fixed 5s
## Because
I verified the audit's CONFIRMED-reproduced bugs are already fixed in current code: C1 zero-work-spawn-done is task 241 (checkLanded guard, mid-fix uncommitted in this tree); C2 CRLF frontmatter loss fixed at mdstore.go:381 (ReplaceAll CRLF->LF); C3 Front.Set injection fixed via quoteScalar at mdstore.go:72 (dacli 170); S2 project-add --slug traversal fixed by ProjectDir/CreateProject validation (workspace.go:247-284, store.go:95-101); S3 pr-status arg injection fixed by '--' end-of-options (lifecycle.go:306); S8 catalog disclosureGate now passes repo through with an explicit --repo probe (catalog.go:340-378, dacli 167). The open backlog (239-243) already covers readyTasks/next, checkLanded, firstLine, int flags, supervise gates — so those classes are taken. acquireSeqLock's steal path is genuinely distinct and unfiled: it is the one concurrency defect with a concrete repro (>=2 waiters past a 5s deadline both O_EXCL-succeed after ownerless os.Remove -> duplicate NNN, the exact dacli-209 collision the lock prevents) and it is entirely untested (task_seq_test only covers fast holders). Not the deadline value: shortening it worsens the steal; the real fix is ownership + backoff + staleness.
