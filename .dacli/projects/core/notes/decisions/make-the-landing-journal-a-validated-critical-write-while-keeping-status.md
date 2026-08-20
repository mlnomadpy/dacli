---
id: d-make-the-landing-journal-a-validated-critical-write-while-keeping-status
kind: note
note_kind: decision
created: 2026-08-18T14:41:19Z
created_by: a-maintainer-dgyp5f
about: "[[t-01M0AEG5F23TRH6BAR9HT38ZP1]]"
github:
  issue: 713
  repo: mlnomadpy/dacli
---
# Make the landing journal a validated critical write while keeping status snapshots advisory
## Chose
Make the landing journal a validated critical write while keeping status snapshots advisory
## Rejected
Fail the loop on every status or governor snapshot write
## Because
The landing journal is the recovery ledger that prevents duplicate work and uncapped restarts; loop-status and governor snapshot writers retain their documented best-effort compatibility boundary.
