---
id: f-brief-assembly-millercap-truncates-findings-decisions-in-alphabetical-filename
kind: note
note_kind: finding
created: 2026-08-04T18:18:12Z
created_by: a-reviewer-664htb
about: "[[t-01KZ6S9FVG533ABEVXZBZZ7SC3]]"
source_event: 01KZ6SS75TY29WCQGH3K884DA8
---
# brief assembly: MillerCap truncates findings/decisions in alphabetical filename order, and trust-floor reflects only survivors
Stage: BRIEF ASSEMBLY. The old finding f-what-siblings-found-section-is-uncapped was fixed by adding MillerCap=7 to the findings section (brief.go:319-353), but the fix truncates in the WRONG order. store.ListNotes (store.go:1426-1444) returns notes in os.ReadDir order = lexical by filename = lexical by title slug (filename is Slugify(title), store.go:1286/1334); brief.go has no sort by severity/recency/trust (grep 'sort' in brief.go: none). So with >7 findings, survival is alphabetical by title: a severity:critical / trust:refuted finding titled late in the alphabet is dropped while a stale minor one survives. The omission is announced only as a bare count (brief.go:352) with no titles/ids/severity, so the agent cannot tell it lost the important one. Compounding: the trust-floor preamble (brief.go:354-357) is computed by noteFloor() which is called ONLY inside the shown<MillerCap loop (brief.go:329,347) -- so a refuted finding cut by the cap never lowers the floor, and the header can advertise 'trust-floor: confirmed' while a refuted finding about this very project exists but was silently omitted. The package doc's promise 'the highest-value content is never what gets cut' (brief.go:4-7) is false WITHIN a section. This is discoverable only by reading brief.go + store.ListNotes.
