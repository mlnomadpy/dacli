---
id: d-274-a-lightweight-dacli-catchup-ref-since-t-command-reads-fresh-sibling
kind: note
note_kind: decision
created: 2026-08-04T20:47:37Z
created_by: a-maintainer-me4vk0
about: "[[274]]"
---
# 274: a lightweight 'dacli catchup <ref> --since T' command reads fresh sibling findings+tasks directly, not by re-running brief.Assemble
## Chose
274: a lightweight 'dacli catchup <ref> --since T' command reads fresh sibling findings+tasks directly, not by re-running brief.Assemble
## Rejected
Re-run 'dacli context' (full brief.Assemble) mid-task, or push updates into the live child process
## Because
Acceptance explicitly forbids re-assembling a full brief mid-run; dacli is a stateless CLI with no channel into a running child, so a pull command the agent re-runs is the only mechanism. catchup reads eventlog pending findings + finding notes + tasks with created>since, scoped to the task's project (mirroring brief.go's §8 scope), and prints only what is new. The spawn preamble injects the exact brief-assembly timestamp as the --since anchor so a finding already folded into the brief is never re-shown as 'new'.
