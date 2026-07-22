---
id: f-searchbymarker-does-a-full-gh-issue-list-per-task-and-per-decision-inside-the
kind: note
note_kind: finding
created: 2026-07-22T18:23:33Z
created_by: a-drg65wknjt
about: [[t-01KY5GP5QJS16DCPAHQMTFBE5X]]
source_event: 01KY5GWS9XB399MGCY74Y7YC9D
github:
  issue: 36
  repo: mlnomadpy/dacli
---
# searchByMarker does a full gh issue list per task and per decision inside the push loop
cmdPush calls searchByMarker(w, marker(w,t)) once per UNMAPPED task (ghmirror.go:179) and mirrorDecisions calls it once per unmapped decision note (ghmirror.go:649). Each searchByMarker (ghmirror.go:722-740) runs 'gh issue list --state all --limit 1000 --json number,body', fetching+parsing up to 1000 issue bodies. On a first push of a project with N unmapped tasks/decisions (none yet found by marker), that is N sequential full-issue-list network round trips = O(N x issues) fetch+parse, when a single pre-loop fetch building a marker->number map would make recovery O(1) per item. Same FindTask-in-a-loop shape siblings flagged elsewhere. Fix: list issues once before the loop and index bodies by marker substring.
