---
id: f-task-334-was-filed-on-an-unreproduced-premise-loop-does-not-buffer-redirected
kind: note
note_kind: finding
created: 2026-08-10T17:52:23Z
created_by: a-root
---
# Task 334 was filed on an unreproduced premise: loop does NOT buffer redirected output (measured: 169 bytes present mid-run), Go writes os.Stdout straight through. The empty log was the operator's own harness capture. The resulting fix asserted ctx.Stdout to a Flusher, which *os.File never satisfies, so it was dead code; its tests read the file after loop() returned and so proved nothing. All reverted. Rule: measure the premise before filing — it cost one command and two agent cycles.
