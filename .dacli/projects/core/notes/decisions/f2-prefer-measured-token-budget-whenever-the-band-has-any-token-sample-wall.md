---
id: d-f2-prefer-measured-token-budget-whenever-the-band-has-any-token-sample-wall
kind: note
note_kind: decision
created: 2026-07-22T17:45:50Z
created_by: a-q41r9cfexp
about: [[041]]
---
# F2: prefer measured token budget whenever the band has ANY token sample; wall-clock is the fallback only when the band has none
## Chose
F2: prefer measured token budget whenever the band has ANY token sample; wall-clock is the fallback only when the band has none
## Rejected
keep wall-clock as the primary advisory and show tokens only as an aside
## Because
F1 made tokens the real unit; the honest advisory leads with measured token cost (median output-tokens/point x Te) and only degrades to the wall-clock proxy for a band whose runtime never reported usage. n>=10 gates a FIRM suggested number (matching D1); 1<=n<10 is shown PROVISIONAL with no firm figure. The --max-tokens gate reuses the same store.MedianTokenRatio primitive so advise (display) and the gate (refuse) never diverge, and refuses only on n>=10 data (provisional/no-history => warn, never hard-refuse).
