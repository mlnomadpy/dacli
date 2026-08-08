---
id: f-teestreamjson-silently-loses-token-usage-on-a-scanner-error-or-an-over-long
kind: note
note_kind: finding
created: 2026-07-22T16:17:27Z
created_by: a-1s5dxes66y
about: [[t-01KY59FNEAAYDZ0PCTKE0HCBVA]]
source_event: 01KY59PH15Q60XJSYV0G3NFQCM
---
# teeStreamJSON silently loses token usage on a scanner error or an over-long line; sc.Err() is never checked
execution.go teeStreamJSON (914-965) scans the child's stream with a 16MB line cap (:918 sc.Buffer(..., 16*1024*1024)). The terminating 'result' event — which carries the ONLY usage numbers (:956-961) — arrives LAST. If any single earlier line exceeds 16MB (a very large assistant/tool message) or the reader errors mid-stream, sc.Scan() returns false and the loop ends BEFORE reaching the result event; u.found stays false and writeUsage is never called, so usage.txt is silently absent and calibration falls back to the wall-clock proxy with no signal that capture failed. sc.Err() (:963-964, return) is never inspected, so a real read error is indistinguishable from clean EOF. Same code path runs at wait-time over the whole transcript in finalizeRun (:1550). Fix: check sc.Err() and either raise the cap or surface a 'usage capture truncated' warning so a missed result event is visible rather than looking like a text runtime.
