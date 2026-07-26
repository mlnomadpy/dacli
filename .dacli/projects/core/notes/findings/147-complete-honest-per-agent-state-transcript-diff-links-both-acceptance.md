---
id: f-147-complete-honest-per-agent-state-transcript-diff-links-both-acceptance
kind: note
note_kind: finding
created: 2026-07-26T15:41:58Z
created_by: a-1dme0jhygs
about: [[147]]
severity: moderate
---
# 147 complete: honest per-agent state + transcript/diff links, both acceptance criteria met
Commit 1066a46 by a-1dme0jhygs (frontend-engineer). AC1 (handler): dashboard.go buildAgentView now sets agentView.State via deriveAgentState (dashboard.go:~470) — reads the transcript's last rendered line (renderTranscriptLine decodes raw stream-json the same way execution.renderStreamLine does) and the file mtime: [tool: X] marker=acting, assistant prose=thinking, empty=waiting, frozen>stallAfter(120s) while alive=stalled; a text runtime with no output is waiting not stalled (isTextRuntime). agentView also gains transcript_url/diff_url, backed by read-only endpoints /api/agents/transcript and /api/agents/diff (dashboard.go serveRunTranscript/serveRunDiff; runID validated against path traversal, git diff HEAD via shared gitx in worktree.txt path or main checkout, honest note never a fake diff). Tests: TestAPIAgents asserts state=thinking+links, TestAgentStateDerivation covers acting/stream-json/stalled/waiting, TestAgentTranscriptEndpoint (render+404+traversal), TestAgentDiffEndpoint — go test ./internal/... all green. AC2 (component): AgentRow.vue renders the state as a bordered badge (word always shown, color decorative) + a detail cell with read-only transcript/diff links (target=_blank rel=noopener); AgentSwarm.vue adds state+detail columns; types.ts Agent mirrors the new fields. Vitest 60/60 pass, vue-tsc + eslint + prettier clean, npm run build (release bundle) OK. Owner a-root: verify and check boxes 1-2 then close via task done + merge --task 147.
