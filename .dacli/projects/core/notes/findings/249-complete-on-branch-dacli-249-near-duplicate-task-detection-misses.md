---
id: f-249-complete-on-branch-dacli-249-near-duplicate-task-detection-misses
kind: note
note_kind: finding
created: 2026-08-04T11:42:49Z
created_by: a-maintainer-kxp25t
about: "[[249]]"
severity: moderate
---
# 249 complete on branch dacli/249-near-duplicate-task-detection-misses-paraphrase — paraphrase dedup via optional semantic backend
Commit 54d1304 by a-maintainer-kxp25t. Both acceptance criteria met. (1) Same-meaning/no-shared-words tasks are detected: store.FindNearDuplicateTask (internal/store/similarity.go:130-179) now consults an optional store.SemanticScorer alongside lexical Jaccard, and — critically — bypasses the minSharedTitleTokens floor for the semantic path (a zero-word-overlap paraphrase has shared==0 and was previously 'continue'd before any backend could see it, similarity.go pre-fix line ~121). TestFindNearDuplicateTaskCatchesParaphraseViaSemanticBackend proves a no-shared-words pair ('stop the watchdog from reaping healthy agents' vs 'prevent the supervisor killing live workers', TitleSimilarity==0) is matched with a stub backend installed; verified it FAILS against the pre-fix continue-on-floor logic and passes after. (2) The backend is optional/zero-dependency: store.SemanticSimilarity defaults nil and activeSemanticScorer() falls back to envSemanticBackend() which is nil unless $DACLI_SEMANTIC_CMD is set (internal/store/semantic_backend.go) — so the default build stays purely lexical. TestFindNearDuplicateTaskParaphraseInvisibleWithoutBackend + TestEnvSemanticBackend cover both. The external backend runs 'sh -c <cmd> dacli-semantic <a> <b>' (titles as $1/$2, no shell injection), 10s timeout, parses one float in [0,1], any failure => no-opinion fallback to lexical. go build/vet clean, gofmt clean, go test ./... green (the pre-existing catalog/cli DACLI_AGENT env-leak failure is unrelated — green under 'go test -exec env -u DACLI_AGENT'). Owner: dacli accept 249 then integrate --tasks 249.
