---
id: 01KY5GY5EXMRHP90KG8EJG8ZE2
kind: event
event_kind: finding
created: 2026-07-22T18:20:49Z
created_by: a-cw27djtx7d
about: [[t-01KY5GP5SE357058NVZKAG0WJ5]]
origin: agent
applied: true
---
ghmirror findingAboutTask matches a task's zero-padded seq as a bare substring, mis-attributing findings across tasks

internal/features/ghmirror/ghmirror.go:406-409 findingAboutTask reports a finding belongs to a task when about contains fmt.Sprintf('%03d', t.Seq) as a plain substring. The %03d form is only 3 digits, so for a workspace with >=1000 tasks a finding about task 1005 (about contains '1005') is mirrored onto the issue for task 005 ('005' is a substring of '1005'). Also a note about NNN-slug where the slug itself contains the padded seq would false-match. The ULID branch (strings.Contains(about, t.ID)) is safe; the seq branch is the loose one. Same loose match is reused by the PR body and verify per the code comment, so the mis-attribution is not isolated to the mirror. Fix: match the seq form on a word boundary (e.g. compare the resolved about ref's leading NNN token, or resolve about via BuildTaskIndex and compare t.ID) rather than substring-containing the padded seq.
