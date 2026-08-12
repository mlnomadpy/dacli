---
id: f-markdown-records-acknowledged-before-their-rename-was-power-loss-durable
kind: note
note_kind: finding
created: 2026-08-12T19:23:56Z
created_by: a-codex-maintainer-zf35yj
about: "[[368]]"
severity: major
---
# Markdown records acknowledged before their rename was power-loss durable
internal/mdstore/mdstore.go:618 previously used CreateTemp + close + rename with no file or directory Sync; task and event writers both funnel through this function, so atomic replacement prevented torn reads but did not ensure an acknowledged rename survived machine power loss.
