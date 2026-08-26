---
id: 01M0Z184083BBFEY9W7XY8Q4N2
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-26T12:38:17Z
created_by: a-root
about: "[[t-01M0CZANEM3TFEMGTW3NTNXGXM]]"
origin: agent
applied: true
checksum: sha256:dc0bdb881b23c96e76985bea9fa7bff5f53503d1a01f24f1c95b2ed4aed60824
---
4073288 fix: probe Claude authentication before spawning

Mutation proof: removing the Claude behavioral strategy made TestUnauthenticatedClaudeIsRefusedBeforeSpawn fail at preflight_test.go:41.
role: root
