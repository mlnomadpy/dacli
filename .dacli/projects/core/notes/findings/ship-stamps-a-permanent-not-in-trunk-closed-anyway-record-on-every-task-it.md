---
id: f-ship-stamps-a-permanent-not-in-trunk-closed-anyway-record-on-every-task-it
kind: note
note_kind: finding
created: 2026-08-10T16:42:06Z
created_by: a-seam-auditor-qzz7rf
about: "[[t-01KZP8HANTQJW7EM8DBRS808ZC]]"
source_event: 01KZP8RF9JFM9X71ZJGWWHCVN0
---
# ship stamps a permanent 'NOT in trunk — closed anyway' record on every task it lands, because accept (step 1) checks the branch before integrate (step 2) merges it
SEAM: dacli ship, accept -> integrate ordering. Trace of the verb end to end.

STEP 1 accept. ship.go:150-163 runs 'dacli accept --all --force' as its FIRST step, with NO --allow-unlanded (contrast the loop). In acceptance/acceptAll (acceptance.go:260-267) each proposed task runs checkLanded(w,t,trunk) BEFORE any merge has happened. At this instant the task's branch dacli/NNN-slug exists and is NOT an ancestor of main (integrate has not run yet), so landed.go:37-72 returns landingUnlanded, and acceptance.go:267 UNCONDITIONALLY appends landingEvidence(landingUnlanded, branch) to the task Log = 'deliverable: dacli/NNN-slug exists but is NOT in trunk — closed anyway' (landed.go:83). CloseTask (acceptance.go:274) flushes that line to the task .md (SaveTask), moving it to done.

STEP 2 integrate. ship.go:180-210 then shells 'dacli integrate --tasks <wave> --into main', which merges each of those same branches into main (lifecycle.go mergeTask / prIntegrateTask). The work NOW lands. Nothing rewrites the task Log: mergeTask on a clean merge only removes the worktree, deletes the branch, and prints (lifecycle.go:1126-1138) — it never touches the task record.

STEP 3 record. ship.go:223 commitRecord stages and commits .dacli, so the false 'NOT in trunk — closed anyway' Log line is COMMITTED into the trajectory (the product) by the very command that landed the work.

WHY EACH SIDE IS CORRECT: accept records the deliverable state truthfully at close time (landed.go:76-78 doc: 'what was known at close time'); integrate lands afterward. Neither is wrong alone.

WHY THE COMPOSITION IS WRONG: ship's fixed order is forced — integrate refuses a non-done task (integrationTasks, lifecycle.go:1505-1517), so accept MUST precede integrate; and accept always snapshots landing BEFORE integrate can land it. So EVERY task ship closes gets a permanent, committed record line asserting its work is NOT in trunk, on a run where ship put it in trunk seconds later. This is the #382 false-record class (record disagreeing with reality) INVERTED: an auditor trained by #382 to distrust done and hunt for 'NOT in trunk' finds this line and wrongly flags landed work as unlanded.

CONTRAST (why ship-specific, not a general accept bug): the loop accepts only AFTER confirming the merge — reconcilePendingAccepts (orchestration.go:904-907) runs 'accept --force' solely on prLandStatus(branch)=='merged', by which point the branch is merged/deleted so checkLanded returns landingLanded or landingNoBranch, never the scary line. ship has no such gate.

HALF-FAILURE STATE: if integrate (step 2) fails (conflict/error), ship stops (ship.go:200-208) and the 'NOT in trunk — closed anyway' line is then ACCURATE (task done but truly unlanded). So the record's truth depends on a later step the record cannot see. On the success path the line is a permanent false negative; nothing on any subsequent run rewrites landingEvidence, so it is unrecoverable unaided.

CHEAPEST FIX DIRECTION: ship's accept step should defer the landing verdict (accept-before-integrate cannot know it), e.g. pass a flag that suppresses writing landingEvidence when the caller will land in the same pipeline, or have ship re-stamp the landing outcome after integrate; --allow-unlanded alone is NOT enough — it only silences the stderr warning (acceptance.go:261-266), the durable Log line at :267 is written regardless.
