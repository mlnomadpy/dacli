---
id: r-vendor-cli-flag-drift-breaks-adapters-silently
kind: risk
created: 2026-07-21T14:43:37Z
created_by: a-root
impact: high
likelihood: high
---
# Vendor CLI flag drift breaks adapters silently
## Indicators
- spawn failures with unknown-flag errors in transcripts
## Action
runtime doctor before every spawn session; adapters carry assumption notes
