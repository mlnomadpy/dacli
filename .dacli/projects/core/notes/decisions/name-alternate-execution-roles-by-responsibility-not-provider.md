---
id: d-name-alternate-execution-roles-by-responsibility-not-provider
kind: note
note_kind: decision
created: 2026-08-27T11:56:10Z
created_by: a-root
---
# Name alternate execution roles by responsibility, not provider
## Chose
Replace claude-maintainer with continuity-maintainer while preserving the same maintainer skills, scope, capacity, consequence tier, and currently selected cc-rw/opus metadata.
## Rejected
Keep the provider-specific claude-maintainer role name
## Because
dacli doctor correctly flags provider names encoded in responsibilities; provider belongs in runtime/model metadata while the role remains portable.
