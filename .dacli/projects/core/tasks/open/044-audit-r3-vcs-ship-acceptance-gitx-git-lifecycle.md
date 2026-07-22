---
id: t-01KY59FNENE0C7CRCSXM3WH9DD
kind: task
created: 2026-07-22T16:10:34Z
created_by: a-root
owner: a-root
priority: should
---
# AUDIT R3: vcs + ship + acceptance + gitx — git lifecycle
## Acceptance
- [ ] findings filed with file:line in internal/features/{vcs,ship,acceptance}/** and internal/gitx/**
## Log
- 2026-07-22T16:11:07Z claimed by a-8b74h81fsz
- 2026-07-22T16:17:27Z finding by a-8b74h81fsz: ship half-ships when accept dirties the .dacli tree (event 01KY59NVWM71HRGTKAQENNBX0Z)
- 2026-07-22T16:17:27Z finding by a-8b74h81fsz: ship half-ships: accept dirties tracked .dacli tree so integrate's clean-tree guard silently no-ops, yet ship still commits+pushes the record (event 01KY59P3YVHHPHDF4SGSJYV831)
- 2026-07-22T16:17:27Z finding by a-8b74h81fsz: cmdIntegrate mislabels every non-conflict merge failure as a conflict and swallows it to exit 0 (event 01KY59PBHF8BV5MJRPVS718T6N)
- 2026-07-22T16:17:27Z finding by a-8b74h81fsz: ship passes bare per-project seqs as --tasks refs from a cross-project done list; multi-project workspaces resolve them ambiguously (event 01KY59PKTGP214C9MKA1P61VSY)
- 2026-07-22T16:17:27Z finding by a-8b74h81fsz: ship's record commit message reports len(done) as 'integrated N task(s)' even when zero branches actually merged (event 01KY59PV8TAQQVHZ26ZFATPQGB)
