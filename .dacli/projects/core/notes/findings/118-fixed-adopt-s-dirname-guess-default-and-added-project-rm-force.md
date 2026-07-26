---
id: f-118-fixed-adopt-s-dirname-guess-default-and-added-project-rm-force
kind: note
note_kind: finding
created: 2026-07-26T23:13:26Z
created_by: a-yms7m1bbzj
about: [[118]]
severity: moderate
---
# 118: fixed adopt's dirname-guess default and added project rm --force
Commit 0ec1535 (branch dacli/118-...). onboard.go:70-96 cmdAdopt now: 0 existing projects -> old dirname-derived-slug behavior (unchanged, needed by TestAdoptExistingRepo); 1 existing project -> defaults to it regardless of dirname (store.ListProjects(w)[0].Slug), fixing the exact repro (dirname 'dacli', project 'core'); 2+ existing projects -> refuses (exit 3) naming the slugs and pointing at --project, rather than guessing. Added store.DeleteProject(w, slug) (store.go, os.RemoveAll(w.ProjectDir(slug)) after an existence check -> ErrNotFound) and planning.go cmdProjectRm ('dacli project rm <slug> --force'): refuses without --force (exit 3, reports task count as the blast radius), refuses for a non-rw grant, ErrNotFound (exit 4) for an unknown slug. New tests: internal/cli/onboard_test.go TestAdoptNoProjectFlagDefaultsToTheOnlyProject / TestAdoptNoProjectFlagRefusesWhenAmbiguous; internal/features/planning/planning_test.go TestProjectRmRefusesWithoutForce / TestProjectRmForceDeletesTheProject / TestProjectRmUnknownSlugNotFound. go build ./... clean; go test ./internal/... all green (incl. architecture tests TestNoDuplicateCommandPaths, TestEveryCommandHasABrief covering the new command registration). Task 118 has no acceptance checkboxes (github-imported report), so task done verifies trivially -- the finding above is the evidence trail.
