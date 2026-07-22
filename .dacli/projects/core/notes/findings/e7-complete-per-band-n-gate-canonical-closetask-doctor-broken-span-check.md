---
id: f-e7-complete-per-band-n-gate-canonical-closetask-doctor-broken-span-check
kind: note
note_kind: finding
created: 2026-07-22T15:10:24Z
created_by: a-yg7894d7yn
about: [[037]]
severity: moderate
---
# E7 complete: per-band n-gate, canonical CloseTask, doctor broken-span check
3 integrity gaps closed. (1) PER-BAND N-GATE insight.go cmdCalibrate: both size bands (:640) and agent bands (:676) now print a p10-p90 range ONLY at n>=10; a thinner band prints median + '(provisional, n<10 - no calibrated range)' and NO range - killing the n=1 '×0.03-×0.03' confidence theater. (2) CANONICAL CLOSE: new store.CloseTask(w,t,actor) (store.go, after AppendLog) stamps 'completed by <actor>' + SaveTask + MoveTask->done; cmdTaskDone (planning.go:325) AND both accept paths (acceptance.go acceptOne + acceptAll) route through it. This ALSO fixes a latent E1 recurrence: acceptOne (single accept) never stamped 'completed by' - only acceptAll did - so a single-accept silently broke calibration until now. Stamp text 'completed by' unchanged (store/cli tests green). (3) DOCTOR: cmdDoctor (insight.go:788) flags done tasks with 'claimed by' but no 'completed by' via new store.LogHasStamp helper, reporting 'broken-calibration-span' naming them. go build + go test ./internal/... green (store, cli incl arch-isolation).
