---
id: f-calibrate-prints-two-contradictory-authoritative-is-the-estimate-lines-for-one
kind: note
note_kind: finding
created: 2026-07-22T18:23:33Z
created_by: a-1zhjz6t2va
about: [[t-01KY5GP5S0V2MFQC2R2EFVEA5A]]
source_event: 01KY5GXVN1H2FS2CD7Y01DTSVY
---
# calibrate prints two contradictory AUTHORITATIVE 'IS the estimate' lines for one band
internal/features/insight/insight.go:700 and :734: for a band with n>=10 wall-clock samples AND n>=10 token samples, cmdCalibrate emits BOTH 'by agent band (role/model/runtime): <band> ... <- AUTHORITATIVE (n>=10: this distribution IS the estimate)' (wall-clock, :700) and 'by agent band (tokens/point) - PREFERRED: <band> ... <- AUTHORITATIVE (n>=10: tokens ARE the estimate)' (:734). Two 'AUTHORITATIVE ... IS the estimate' claims for the SAME band contradict each other's authority, since the token section is explicitly the PREFERRED unit and wall-clock the fallback (:725). When any token sample exists for a band, the wall-clock line should drop the AUTHORITATIVE/'IS the estimate' wording and read as the fallback, so only one unit claims to be the estimate.
