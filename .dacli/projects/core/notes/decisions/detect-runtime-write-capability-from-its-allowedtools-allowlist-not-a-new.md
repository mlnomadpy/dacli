---
id: d-detect-runtime-write-capability-from-its-allowedtools-allowlist-not-a-new
kind: note
note_kind: decision
created: 2026-08-04T11:47:50Z
created_by: a-maintainer-0fh816
about: "[[250]]"
---
# detect runtime write-capability from its --allowedTools allowlist, not a new frontmatter field
## Chose
detect runtime write-capability from its --allowedTools allowlist, not a new frontmatter field
## Rejected
adding an explicit write_capable/sandbox_rw frontmatter field to Runtime
## Because
the write-capability signal already lives on disk in the runtime's --allowedTools list (cc-rw declares Edit,Write in invoke_args; cc declares only a read-only sandbox_ro allowlist). A new field would need every existing adapter migrated and would still not catch the shipped cc/junior bug. store.RuntimeWritable reads the existing allowlist: writable = the invoke-args allowlist names a write tool; a runtime that pins NO allowlist anywhere stays writable so generic-exec is never falsely refused. sandboxFor (spawn gate) and cmdDoctor share the one predicate, so shown and enforced can never diverge.
