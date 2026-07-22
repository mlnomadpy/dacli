---
id: f-g6-done-type-severity-area-labels-reliable-pre-creation-on-branch-dacli-070
kind: note
note_kind: finding
created: 2026-07-22T19:50:30Z
created_by: a-jfsgjqqya0
about: [[070]]
severity: moderate
---
# G6 done: type/severity/area labels + reliable pre-creation on branch dacli/070
internal/features/ghmirror/ghmirror.go: severityLabel already mapped major/moderate/minor correctly (root of public-repo unspecified was stale labels on adopted issues, not the map). Added: (1) precreateLabels() creates the full static set (finding,decision,type:finding|task|decision,severity:major|moderate|minor|unspecified,status:*) with stable labelColor() ONCE at push start (cmdPush after disclosureGate) so no issue-create races a missing label; (2) type: labels on all three issue kinds; (3) best-effort area:<slice> from areaSlice() parsing the first internal/<...> path (last dir segment) for findings and area:<project> for tasks; (4) applyFindingLabels() strips otherSeverityLabels() so an issue first filed as severity:unspecified is corrected, not left double-labeled. ensureLabel() now applies --color. Unit tests: TestAreaSliceFromPath, TestAreaLabel, TestBaseLabelsAndColors, TestOtherSeverityLabels (no live gh). go build + go test ./internal/... green.
