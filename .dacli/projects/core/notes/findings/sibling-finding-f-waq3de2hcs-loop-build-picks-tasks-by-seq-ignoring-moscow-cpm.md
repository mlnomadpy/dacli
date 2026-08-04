---
id: f-sibling-finding-f-waq3de2hcs-loop-build-picks-tasks-by-seq-ignoring-moscow-cpm
kind: note
note_kind: finding
created: 2026-08-04T00:37:40Z
created_by: a-8p0kde6tvt
about: "[[t-01KY60QM1Y7DK05WXB954YNDHJ]]"
source_event: 01KY7KSSFJ814Y080N12WMJ5N5
---
# Sibling finding f-waq3de2hcs (loop BUILD picks tasks by seq, ignoring MoSCoW/CPM) is now STALE — already fixed
f-waq3de2hcs claimed runCycle builds ready[:width] with no priority sort. That was fixed by task 103 (commit e9a817a, merged PR #66): orchestration.go:292 now calls rankByPriority(d.w, d.cfg.project, ready) BEFORE d.gov.Before/runCycle, so the ready frontier is ranked by MoSCoW priority + CPM slack before the width slice. Verified in current tree. Do not re-file this; treat f-waq3de2hcs as resolved. Confirmed still-LIVE by contrast: f-0b77j7k11m (idle token uncounting, now task 108) and f-g3ya9r93e3 (driver.git no deadline, orchestration.go:506-511) both remain unfixed in the current tree.
