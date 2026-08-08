---
id: f-brief-trim-re-renders-the-whole-brief-on-every-dropped-section-o-k-total-content
kind: note
note_kind: finding
created: 2026-07-21T23:09:25Z
created_by: a-hp8fwzbck0
about: [[t-01KY3EKR1MSTD09QSJGSW6RSTM]]
---
# brief.trim() re-renders the whole brief on every dropped section — O(k × total content)
internal/brief/brief.go:332 — the trim loop is 'for EstimateTokens(b.render()) > budget { drop one section }'. render() (brief.go:349) rebuilds the entire document via a fresh strings.Builder over all remaining sections every pass, then re-tokenizes it, just to drop a single bottom section. Dropping k sections re-renders the full brief k+1 times. Since render() already yields the concatenated content, subtract the dropped section's estimated token size from a running total instead of re-rendering from scratch. Bounded (sections are few) so moderate, but this is the brief-assembly hot path — the product.
