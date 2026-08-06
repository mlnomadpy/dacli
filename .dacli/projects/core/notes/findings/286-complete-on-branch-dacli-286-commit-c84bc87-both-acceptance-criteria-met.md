---
id: f-286-complete-on-branch-dacli-286-commit-c84bc87-both-acceptance-criteria-met
kind: note
note_kind: finding
created: 2026-08-04T20:19:55Z
created_by: a-maintainer-h0se2r
about: "[[286]]"
severity: major
---
# 286 complete on branch dacli/286 — commit c84bc87, both acceptance criteria met, build+test+vet+gofmt green
Branch dacli/286-the-brief-truncates-findings-and-decisions-in-alphabetical-filename-order-so-a, commit c84bc87 (a-maintainer-h0se2r). Files: internal/brief/brief.go, brief_test.go, sort_test.go (new). ACCEPTANCE: (1) the cap now selects by severity>trust>recency, not os.ReadDir order — internal/brief/brief.go sortFindings() ranks finding NOTES by severityRank (major<moderate<minor<unknown), then TrustRank (confirmed>unverified>refuted), then created desc; sortByRecency() ranks decisions newest-first (decisions carry no severity/trust, so recency is their only applicable signal). Both applied before the MillerCap loops (brief.go §4 decisions, §8 findings). (2) what was dropped is NAMED, not a bare count — namedOmission() lists dropped [[id]] (severity) up to omissionNameCap=7 then '+N more' for the low-severity tail (naming all ~312 dropped would defeat the working-memory budget; since drops are worst-last, the named ones are the borderline cases). PROOF (verified by reproduction): stashed brief.go and ran the two Assemble-level tests against OLD code — TestFindingsCapKeepsSevereAndNamesDropped FAILED ('the cap dropped the major finding it should have kept', showing 7 aaa-* minors and dropping zzz major) and TestDecisionsCapNamesDropped FAILED ('reported a bare count'); both PASS after the fix. go build ./... clean; go test ./... all green; go vet ./... clean; gofmt -l internal/ empty. PR-first is off this run — owner: dacli accept 286 then integrate/merge --task 286.
