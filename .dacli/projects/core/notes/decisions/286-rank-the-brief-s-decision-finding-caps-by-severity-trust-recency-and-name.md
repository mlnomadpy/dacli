---
id: d-286-rank-the-brief-s-decision-finding-caps-by-severity-trust-recency-and-name
kind: note
note_kind: decision
created: 2026-08-04T20:11:27Z
created_by: a-maintainer-h0se2r
about: "[[286]]"
---
# 286: rank the brief's decision/finding caps by severity+trust+recency and name the dropped items (top-N + tail count)
## Chose
286: rank the brief's decision/finding caps by severity+trust+recency and name the dropped items (top-N + tail count)
## Rejected
keeping os.ReadDir (alphabetical filename) order and only converting the bare count to a full enumeration of every dropped item
## Because
os.ReadDir sorts by filename, so a high-number/late-alphabet critical finding is dropped while an old 084-* decision survives — the exact silent-drop bug. Sorting findings by severity>trust>recency and decisions by recency guarantees the most-critical survive the MillerCap. Naming EVERY dropped item is unworkable (core already drops 312 findings/164 decisions — ~10k tokens of HTML-comment tail defeats the working-memory budget the cap exists to enforce), so name the most-severe dropped items (the ones a critical-drop concern cares about, now that they sort first) and summarize the long low-severity tail as '+N more'.
