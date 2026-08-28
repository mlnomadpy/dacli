---
id: t-01KZXRP56538HNHDP3P4FJHGGP
kind: task
created: 2026-08-13T14:33:43Z
created_by: loop
owner: a-root
priority: should
estimate: "{optimistic: 1, probable: 2, pessimistic: 3}"
depends_on: "[t-01M146BA81VWWJ27PWFRFN1E2M, t-01M147RA8749AVXNXB142NS5KT, t-01M147R9WKDPHMMT07HHC6V37B]"
---
# Continuous improvement: file the single highest-value evidence-based change
## Context
Standing anchor for the autonomous review phase. Survey the code, tests, CI, and open findings; identify the ONE highest-value improvement grounded in evidence (a failing test, a reviewer finding, a real defect). Before filing, run `dacli task list --project core --status open` and `dacli task list --project core --status active` to check whether the backlog already queues it — a prior cycle may have filed the same issue under different wording. `dacli task add` refuses (exit 3) a title that scores as a near-duplicate of existing work, so pick real, distinct scope rather than re-filing and re-running with --force. If the audit finds no distinct task after those duplicate checks, that is an honest result: record a finding naming what you audited and the open/active work that already covers it, then finish this anchor without filing placeholder work. Otherwise file the distinct task with concrete acceptance criteria. Do NOT implement anything here, and do NOT invent speculative work.

Just-completed wave (treat this as queued work when checking duplicates):
- task t-01M13YH646HH1BWH0NHCM3DQP6 (534-reduce-pr-runner-minutes-by-moving-fuzz-campaigns-and-collapsing-cross-compiles); status=open; branch=dacli/534-reduce-pr-runner-minutes-by-moving-fuzz-campaigns-and-collapsing-cross-compiles; commit=a79bd62dc37dbc1932fa2d6487ec5c0757ff4f1d; linked_issue=#853; pending_pr_landing=false
## Acceptance
- [ ] Evidenced exactly one outcome: filed a distinct task grounded in an observed defect, finding, or failing check; or recorded a reviewer finding that the audit found no distinct task after checking open and active work for duplicates
- [ ] Did not implement any change in this task
## Log
- 2026-08-13T14:33:44Z claimed by a-go-auditor-hcwqe3
- 2026-08-22T17:21:46Z claimed by a-adversarial-reviewer-261s4h
- 2026-08-22T22:02:16Z claimed by a-adversarial-reviewer-h5rnfr
- 2026-08-26T12:55:19Z claimed by a-adversarial-reviewer-wzq9fh
- 2026-08-26T13:21:25Z claimed by a-adversarial-reviewer-f9a8a9
- 2026-08-26T13:44:04Z claimed by a-adversarial-reviewer-gstekv
- 2026-08-26T14:29:53Z claimed by a-adversarial-reviewer-5zz06w
- 2026-08-26T23:13:15Z claimed by a-go-auditor-4n06wv
- 2026-08-26T23:22:07Z claimed by a-go-auditor-2g8x81
- 2026-08-27T13:13:20Z claimed by a-go-auditor-nqea6c
- 2026-08-27T22:10:27Z claimed by a-adversarial-reviewer-bcbjaf
- 2026-08-27T22:23:14Z claimed by a-adversarial-reviewer-n10ab6
- 2026-08-27T22:52:01Z claimed by a-adversarial-reviewer-r28r4a
- 2026-08-27T23:18:41Z claimed by a-adversarial-reviewer-axpezq
- 2026-08-27T23:44:14Z claimed by a-adversarial-reviewer-rvb458
- 2026-08-28T00:22:06Z claimed by a-adversarial-reviewer-yth0jg
- 2026-08-28T00:44:50Z claimed by a-adversarial-reviewer-ewbb3d
- 2026-08-28T09:36:20Z claimed by a-adversarial-reviewer-w3g81a
- 2026-08-28T09:56:31Z claimed by a-adversarial-reviewer-4fgfsv
- 2026-08-28T10:06:33Z claimed by a-adversarial-reviewer-d7qak1
- 2026-08-28T10:35:53Z claimed by a-adversarial-reviewer-12hhsd
- 2026-08-28T12:44:32Z adopted by a-root (owner loop orphaned)
- 2026-08-28T12:44:48Z dependency edit by a-root (event 01M146DFJD8KAABB63B8RZHR2P)
- 2026-08-28T13:32:47Z dependency edit by a-root (event 01M1495BTNPDM3QW9Y7F9XG96Q)
