---
id: f-r5-audit-coverage-workspace-clikit-agentid-ulid-model-reviewed-no-new-defects
kind: note
note_kind: finding
created: 2026-07-22T16:17:27Z
created_by: a-0htd7wjqyt
about: [[t-01KY59FNFAEE0KT7PWV8HAAY4A]]
source_event: 01KY59VJAJ0DVFKZHX2YN9P5E1
---
# R5 audit coverage: workspace/clikit/agentid/ulid/model reviewed, no new defects
Read in full, no fileable defect beyond items already tracked: internal/workspace/workspace.go (Find worktree-redirect :53-57 correct; open() format gate :104 fine; mainWorktreeRoot shells git once per Find — known perf, not a bug), internal/clikit/clikit.go (ParseFlags '--value' limitation :104 is the already-open [[f-flag-parser-cannot-take-values-that-start-with]] with '=' as the documented path; ExitCode contract :56-72 correct), internal/agentid/agentid.go (Resolve :52-82 one-shot linear scan of agent files — tolerable for a per-process resolve; Spawn attenuation :109 monotonic and correct; id uses 50 random bits of the ULID :122), internal/ulid/ulid.go (base32 timestamp+80 random, crypto/rand, Valid :63 — solid), internal/model/model.go (Meta.Extra is unused here but unknown frontmatter survives round-trip at the mdstore.Front layer, so no data loss). Defects for this R5 slice are the three separate findings on mdstore.WriteFile, eventlog.List, and sync.go logOnce.
