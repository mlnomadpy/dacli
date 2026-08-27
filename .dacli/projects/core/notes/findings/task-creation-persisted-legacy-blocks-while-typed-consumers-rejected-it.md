---
id: f-task-creation-persisted-legacy-blocks-while-typed-consumers-rejected-it
kind: note
note_kind: finding
created: 2026-08-27T22:05:57Z
created_by: a-fixer-dqsb6g
about: "[[t-01M12K8SH454ZH3Z1MB1Q3D4TG]]"
severity: major
---
# Task creation persisted legacy :blocks while typed consumers rejected it
internal/store/store.go previously wrote TaskOpts.DependsOn unchanged; internal/store/dependency.go validates only FS/SS/FF/SF, so later unrelated edits and internal/features/insight critical-path could fail on a newly created :blocks record.
