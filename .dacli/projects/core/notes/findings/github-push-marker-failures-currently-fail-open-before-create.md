---
id: f-github-push-marker-failures-currently-fail-open-before-create
kind: note
note_kind: finding
created: 2026-08-18T14:18:23Z
created_by: a-maintainer-fm4hfq
about: "[[t-01M088WV632VPCXW0Y37P3DSCC]]"
severity: major
---
# github push marker failures currently fail open before create
internal/features/ghmirror/ghmirror.go markerIndex.preflight previously ignored fetch/JSON failures and cmdPush proceeded to issue create from an untrusted empty snapshot; focused regression now requires refusal before create.
