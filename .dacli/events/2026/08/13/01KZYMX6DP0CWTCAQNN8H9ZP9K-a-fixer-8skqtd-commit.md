---
id: 01KZYMX6DP0CWTCAQNN8H9ZP9K
kind: event
schema_version: 1
event_kind: commit
created: 2026-08-13T22:46:54Z
created_by: a-fixer-8skqtd
about: "[[t-01KZYMP0X162ZPK17GFX0MFY8C]]"
origin: agent
applied: true
checksum: sha256:287833a0a99105b7bc670928e62c91a1b0ed6600f7ada3b9616429a8e13b64c3
---
7054b8e 438: pin CI security scan to patched Go 1.25.13

The workflow contract failed before the fix with:
lint job must use Go 1.25.13 or newer within the 1.25 line
role: fixer
