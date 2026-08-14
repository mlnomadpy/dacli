---
id: f-prompt-templates-receive-an-ambiguous-numeric-mutation-reference
kind: note
note_kind: finding
created: 2026-08-14T01:48:01Z
created_by: a-maintainer-204p4w
about: "[[443]]"
severity: major
---
# Prompt templates receive an ambiguous numeric mutation reference
internal/features/execution/execution.go:2115-2123 and 2175-2184 pass fmt.Sprintf("%03d", t.Seq) as .Ref; internal/prompts/tpl/protocol_preamble.md:29-31 and git_workflow.md consume it in task check/done/accept commands.
