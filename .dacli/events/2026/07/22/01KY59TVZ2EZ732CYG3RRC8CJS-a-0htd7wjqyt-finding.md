---
id: 01KY59TVZ2EZ732CYG3RRC8CJS
kind: event
event_kind: finding
created: 2026-07-22T16:16:41Z
created_by: a-0htd7wjqyt
about: [[t-01KY59FNFAEE0KT7PWV8HAAY4A]]
origin: agent
applied: true
---
R5 scope: six prior sibling findings verified FIXED in current tree

Verified by reading current source (no runtime — go test/build blocked by headless sandbox). RESOLVED: (1) [[f-embedded-immutable-templates-re-read-re-parsed-on-every-call-prompts-mcpdesc-gates-get-and-skill-dirs-scanned-twice-per-load]] — prompts.go now memoizes compiled templates (tplCache, :40-61) and parses mcp_tools.md once (mcpDescOnce, :112-135). (2) [[f-findtask-reads-parses-the-entire-task-tree-per-call-amplified-to-o-events-tasks-inside-sync-taint-replay-loops]] for the Sync path — sync.go:36 builds store.BuildTaskIndex once, O(1) per event. (3) [[f-eventlog-apply-is-non-atomic-a-mid-apply-failure-leaves-the-event-pending-and-re-runs-it-duplicating-notes-log-lines-on-next-sync]] — apply() is now idempotent: logOnce dedupes on full event id (sync.go:186-192), CreateNote dedupes via NoteOpts.SourceEvent (:120-126), MoveTask-to-current is a no-op; a re-run no longer duplicates. (4) [[f-security-critical-prompt-lines-are-formatted-as-html-comments-which-llms-may-read-as-inert-metadata]] — brief_header.md:2 and supervise_correction.md:2 now use a bold '**SYSTEM:**' block, not '<!-- -->'. (5) [[f-review-workflow-md-s-default-pr-search-string-never-matches-a-real-pr]] — review_workflow.md:4 now 'gh pr list --head "{{.Search}}"' with Search bound to the branch name 'dacli/%03d-%s' (execution.go:1007), not t.ID. (6) [[f-protocol-preamble-md-never-tells-agents-to-file-decision-notes-only-findings-and-asks]] — protocol_preamble.md:11-12 now has the 'note add decision --rejected --because' bullet. These should move from unverified toward confirmed-fixed in the trust floor.
