package ghmirror

import (
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

func mirrorWorkspace(t *testing.T) *workspace.Workspace {
	t.Helper()
	w, err := workspace.Init(t.TempDir(), "test")
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := store.CreateProject(w, "a-root", "Core", "core", "goal", ""); err != nil {
		t.Fatalf("project: %v", err)
	}
	return w
}

// G1: exactly one status label is applied and the other three are stripped, so
// a re-push never stacks a second status: label on the same issue.
func TestStatusLabelDedup(t *testing.T) {
	for _, s := range model.AllStatuses {
		add := statusLabel(s)
		if add != "status:"+string(s) {
			t.Fatalf("statusLabel(%s) = %q", s, add)
		}
		others := otherStatusLabels(s)
		if len(others) != len(model.AllStatuses)-1 {
			t.Fatalf("otherStatusLabels(%s): got %d, want %d", s, len(others), len(model.AllStatuses)-1)
		}
		for _, o := range others {
			if o == add {
				t.Fatalf("otherStatusLabels(%s) must not include the applied label %q", s, add)
			}
			if !strings.HasPrefix(o, "status:") {
				t.Fatalf("stale label %q is not a status: label", o)
			}
		}
	}
}

// G2: the decision marker is keyed on BOTH the note id and the workspace id, and
// is distinct from the task marker so the two mirrors never adopt each other.
func TestDecisionMarkerKeying(t *testing.T) {
	w := mirrorWorkspace(t)
	mk := decisionMarker(w, "d-example")
	if !strings.Contains(mk, "d-example") {
		t.Fatalf("marker %q omits the note id", mk)
	}
	if !strings.Contains(mk, "ws:"+w.ID) {
		t.Fatalf("marker %q omits the workspace id", mk)
	}
	// A task marker for the same id must NOT be a substring of the decision
	// marker (searchByMarker matches by substring), or adoption would cross.
	tk := &store.Task{ID: "d-example"}
	if strings.Contains(mk, marker(w, tk)) {
		t.Fatalf("decision marker %q collides with task marker %q", mk, marker(w, tk))
	}
}

// G2: a decision note read from disk carries no issue mapping (create runs),
// and once the mapping is written back, mappedIssueDoc reports it — the local
// half of the idempotency guarantee (a second push skips create).
func TestDecisionMappingIdempotency(t *testing.T) {
	w := mirrorWorkspace(t)
	if _, err := store.CreateNote(w, "a-root", "core", model.NoteDecision, "use labeled issues", store.NoteOpts{
		Rejected: "GraphQL Discussions",
		Because:  "reuses the existing marker idempotency machinery",
		Body:     "mirror decisions as issues labeled decision",
	}); err != nil {
		t.Fatalf("create decision: %v", err)
	}

	notes, err := decisionNotes(w, "core")
	if err != nil {
		t.Fatalf("decisionNotes: %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("got %d decision notes, want 1", len(notes))
	}
	dn := notes[0]
	if dn.id == "" {
		t.Fatalf("decision note has no id")
	}
	if got := mappedIssueDoc(dn.doc); got != 0 {
		t.Fatalf("unmapped note reports issue %d, want 0", got)
	}

	// The WHY must survive into the issue body.
	body := decisionBody(w, dn)
	for _, want := range []string{"GraphQL Discussions", "reuses the existing marker idempotency", "use labeled issues", dn.id} {
		if !strings.Contains(body, want) {
			t.Fatalf("decision body missing %q:\n%s", want, body)
		}
	}
	if !strings.Contains(body, decisionMarker(w, dn.id)) {
		t.Fatalf("decision body missing its marker")
	}

	// Simulate the write-back and re-read: the second push must see the mapping.
	dn.doc.Front.SetBlock("github", "  issue: 42\n  repo: owner/repo")
	if err := mdstore.WriteFile(dn.path, dn.doc); err != nil {
		t.Fatalf("write back: %v", err)
	}
	reread, err := decisionNotes(w, "core")
	if err != nil {
		t.Fatalf("re-read: %v", err)
	}
	if got := mappedIssueDoc(reread[0].doc); got != 42 {
		t.Fatalf("mapped note reports issue %d, want 42", got)
	}
}

// decisionNotes on a project with no decisions dir is empty, not an error.
func TestDecisionNotesEmpty(t *testing.T) {
	w := mirrorWorkspace(t)
	notes, err := decisionNotes(w, "core")
	if err != nil {
		t.Fatalf("decisionNotes on empty project: %v", err)
	}
	if len(notes) != 0 {
		t.Fatalf("got %d notes, want 0", len(notes))
	}
}

// G4 inbound: pull adopts a human-authored issue but skips (a) an issue dacli
// itself mirrored (its body carries a dacli marker) and (b) an issue already
// bound to a local task (mapped by number) — the idempotency that stops a
// re-pull from re-importing.
func TestShouldImportSkipLogic(t *testing.T) {
	w := mirrorWorkspace(t)
	human := ghIssue{Number: 1, Title: "Human bug report", Body: "steps to repro"}
	// An issue we mirrored outbound carries the task marker in its body.
	ours := ghIssue{Number: 2, Body: marker(w, &store.Task{ID: "t-abc"}) + "\n\nbody"}
	// A decision issue we mirrored also carries a dacli marker.
	ourDecision := ghIssue{Number: 3, Body: decisionMarker(w, "d-x") + "\n\nbody"}
	alreadyLinked := ghIssue{Number: 4, Title: "linked", Body: "no marker but mapped"}

	mapped := map[int]bool{4: true}

	if !shouldImport(human, mapped) {
		t.Fatalf("human-authored issue #1 should import")
	}
	if shouldImport(ours, mapped) {
		t.Fatalf("issue #2 carrying our own task marker must be skipped")
	}
	if shouldImport(ourDecision, mapped) {
		t.Fatalf("issue #3 carrying our decision marker must be skipped")
	}
	if shouldImport(alreadyLinked, mapped) {
		t.Fatalf("issue #4 already mapped to a task must be skipped")
	}
}

// G4 inbound: the marker embedded in a finding comment must NOT be seen as a
// body marker by pull's skip logic, or a task's own finding comment could make
// its issue look dacli-authored. (Distinct concern from the marker prefix; this
// guards the constant against future drift.)
func TestFindingMarkerNotABodyMarkerPrefixCollision(t *testing.T) {
	w := mirrorWorkspace(t)
	fm := findingMarker(w, "f-example")
	// A finding marker DOES share the "<!-- dacli" prefix, but it only ever
	// lives in comments, never issue bodies. What must hold is that it is
	// distinct from task/decision markers so searchByMarker/adoption never cross.
	if strings.Contains(fm, marker(w, &store.Task{ID: "f-example"})) {
		t.Fatalf("finding marker %q collides with task marker", fm)
	}
	if strings.Contains(fm, decisionMarker(w, "f-example")) {
		t.Fatalf("finding marker %q collides with decision marker", fm)
	}
	if !strings.Contains(fm, "ws:"+w.ID) {
		t.Fatalf("finding marker %q omits the workspace id", fm)
	}
}

// G4 findings→comments: a finding already present as a comment (its marker in
// the body) is skipped, an absent one is posted — the idempotency that stops a
// re-push from duplicating finding comments.
func TestCommentsHaveMarker(t *testing.T) {
	w := mirrorWorkspace(t)
	mk := findingMarker(w, "f-leak")
	body := findingComment(mk, "major", "f-leak", "a real leak at foo.go:12")

	// The rendered comment carries the marker, the severity, id and text.
	for _, want := range []string{mk, "major", "f-leak", "a real leak at foo.go:12"} {
		if !strings.Contains(body, want) {
			t.Fatalf("finding comment missing %q:\n%s", want, body)
		}
	}
	existing := []string{"a plain human comment", body}
	if !commentsHaveMarker(existing, mk) {
		t.Fatalf("marker present in an existing comment must be detected (skip)")
	}
	if commentsHaveMarker([]string{"unrelated"}, mk) {
		t.Fatalf("marker absent must NOT be detected (post)")
	}
	// A different finding's marker must not match this comment.
	if commentsHaveMarker(existing, findingMarker(w, "f-other")) {
		t.Fatalf("a different finding's marker must not match")
	}
}

// G4 findings→comments: only findings whose `about` names the task are mirrored;
// a finding about another task is not.
func TestFindingAboutTask(t *testing.T) {
	w := mirrorWorkspace(t)
	if _, err := store.CreateNote(w, "a-root", "core", model.NoteFinding, "mine", store.NoteOpts{
		About: "t-target", Severity: "minor", Body: "detail at x.go:1",
	}); err != nil {
		t.Fatalf("create finding: %v", err)
	}
	if _, err := store.CreateNote(w, "a-root", "core", model.NoteFinding, "theirs", store.NoteOpts{
		About: "t-other", Severity: "minor", Body: "detail at y.go:2",
	}); err != nil {
		t.Fatalf("create finding: %v", err)
	}
	notes, err := store.ListNotes(w, "core", model.NoteFinding)
	if err != nil {
		t.Fatalf("list notes: %v", err)
	}
	target := &store.Task{ID: "t-target", Seq: 1}
	var matched []string
	for _, n := range notes {
		if findingAboutTask(n, target) {
			matched = append(matched, findingText(n))
		}
	}
	if len(matched) != 1 {
		t.Fatalf("got %d findings about the target task, want 1: %v", len(matched), matched)
	}
	if !strings.Contains(matched[0], "detail at x.go:1") {
		t.Fatalf("matched the wrong finding: %q", matched[0])
	}
}

// findingAboutTask must match the task's ref EXACTLY, never as a loose
// zero-padded-seq substring: a finding about task 10007 (whose ref contains the
// digits "007") must NOT be mirrored onto task 007's issue, and neither must a
// finding about a sibling with a different seq.
func TestFindingAboutTaskPrecise(t *testing.T) {
	target := &store.Task{ID: "t-XYZ", Seq: 7, Slug: "the-target"}
	about := func(v string) *mdstore.Doc {
		d := &mdstore.Doc{}
		d.Front.Set("about", "[["+v+"]]")
		return d
	}
	matches := []string{"t-XYZ", "XYZ", "7", "007", "007-the-target", "the-target"}
	for _, v := range matches {
		if !findingAboutTask(about(v), target) {
			t.Errorf("about %q should match task 007 (t-XYZ)", v)
		}
	}
	// The cross-match cases the old loose Contains(about, "007") wrongly matched.
	crossers := []string{"10007", "0070", "008", "t-XYZW", "070", "70"}
	for _, v := range crossers {
		if findingAboutTask(about(v), target) {
			t.Errorf("about %q must NOT match task 007 — loose substring cross-match", v)
		}
	}
	// An unbracketed about (defensive: no wikilink) still matches by exact ref.
	bare := &mdstore.Doc{}
	bare.Front.Set("about", "007")
	if !findingAboutTask(bare, target) {
		t.Errorf("bare (unbracketed) about 007 should still match")
	}
}

// Public-push consent is scoped to the exact repo it was granted for, not a bare
// boolean: consent for repo A must not authorize a push to repo B, and a legacy
// "true" matches no repo (fails closed).
func TestConsentCoversRepo(t *testing.T) {
	if !consentCoversRepo("owner/repo", "owner/repo") {
		t.Errorf("consent for a repo must cover that same repo")
	}
	if !consentCoversRepo("Owner/Repo", "owner/repo") {
		t.Errorf("repo comparison should be case-insensitive (gh nameWithOwner casing)")
	}
	if consentCoversRepo("owner/repo", "owner/other") {
		t.Errorf("consent for one repo must NOT cover a different repo")
	}
	if consentCoversRepo("true", "owner/repo") {
		t.Errorf("a legacy bare-boolean consent must not cover any repo (fail closed)")
	}
	if consentCoversRepo("", "owner/repo") {
		t.Errorf("absent consent must not cover any repo")
	}
}

// mappedBlockChanged is the guard that lets a re-push skip a file write when the
// issue mapping is unchanged — so an idempotent push does not rewrite every task
// file. githubBlock renders the exact bytes both sides compare.
func TestMappedBlockUnchangedSkipsWrite(t *testing.T) {
	d := &mdstore.Doc{}
	desired := githubBlock(42, "owner/repo")
	if !mappedBlockChanged(d, desired) {
		t.Fatalf("an unmapped doc must report the mapping as changed (needs a first write)")
	}
	d.Front.SetBlock("github", desired)
	if mappedBlockChanged(d, desired) {
		t.Fatalf("a doc already carrying the desired mapping must report unchanged (skip the write)")
	}
	if !mappedBlockChanged(d, githubBlock(43, "owner/repo")) {
		t.Fatalf("a different issue number must report changed")
	}
	if !mappedBlockChanged(d, githubBlock(42, "owner/other")) {
		t.Fatalf("a different repo must report changed")
	}
}

// mappedIssues collects the issue numbers already bound to local tasks — the
// skip-set pull uses. A task with no github block contributes nothing.
func TestMappedIssues(t *testing.T) {
	linked := &store.Task{Doc: &mdstore.Doc{}}
	linked.Doc.Front.SetBlock("github", "  issue: 7\n  repo: owner/repo")
	unlinked := &store.Task{Doc: &mdstore.Doc{}}

	mapped := mappedIssues([]*store.Task{linked, unlinked})
	if !mapped[7] {
		t.Fatalf("issue 7 should be in the mapped set")
	}
	if len(mapped) != 1 {
		t.Fatalf("got %d mapped issues, want 1", len(mapped))
	}
}

// G5: severity maps to a valid label — the three real severities pass through
// and anything else (empty, garbage) falls back to a still-valid
// `severity:unspecified` rather than a missing label.
func TestSeverityLabelMapping(t *testing.T) {
	cases := map[string]string{
		"major":    "severity:major",
		"moderate": "severity:moderate",
		"minor":    "severity:minor",
		"MAJOR":    "severity:major", // case-insensitive
		" minor ":  "severity:minor", // trimmed
		"":         "severity:unspecified",
		"critical": "severity:unspecified", // unknown → total mapping, never empty
	}
	for in, want := range cases {
		if got := severityLabel(in); got != want {
			t.Fatalf("severityLabel(%q) = %q, want %q", in, got, want)
		}
	}
}

// G5: the finding-ISSUE marker is keyed on BOTH the note id and the workspace
// id, and is distinct from the task, decision, AND finding-comment markers so
// searchByMarker/adoption never crosses between the standalone-issue mirror and
// any other mirror.
func TestFindingIssueMarkerKeying(t *testing.T) {
	w := mirrorWorkspace(t)
	mk := findingIssueMarker(w, "f-leak")
	if !strings.Contains(mk, "f-leak") {
		t.Fatalf("marker %q omits the note id", mk)
	}
	if !strings.Contains(mk, "ws:"+w.ID) {
		t.Fatalf("marker %q omits the workspace id", mk)
	}
	// Must not be confused with the other three markers (searchByMarker matches
	// by substring, so a substring collision would cross-adopt).
	if strings.Contains(mk, marker(w, &store.Task{ID: "f-leak"})) {
		t.Fatalf("finding-issue marker %q collides with task marker", mk)
	}
	if strings.Contains(mk, decisionMarker(w, "f-leak")) {
		t.Fatalf("finding-issue marker %q collides with decision marker", mk)
	}
	// The finding-COMMENT marker <!-- dacli-finding:… --> must not be a substring
	// of the finding-ISSUE marker <!-- dacli-finding-issue:… -->.
	if strings.Contains(mk, findingMarker(w, "f-leak")) {
		t.Fatalf("finding-issue marker %q collides with finding-comment marker", mk)
	}
}

// G5: the standalone issue body carries the marker (idempotency), the severity,
// the finding detail, and the note-id backlink.
func TestFindingIssueBody(t *testing.T) {
	w := mirrorWorkspace(t)
	dn := noteFile{id: "f-leak", title: "public repo leaks findings", doc: &mdstore.Doc{
		Sections: []mdstore.Section{
			{Level: 1, Title: "public repo leaks findings", Content: ""},
			{Level: 0, Content: "a real leak at ghmirror.go:44\n"},
		},
	}}
	body := findingIssueBody(w, dn, "major")
	for _, want := range []string{findingIssueMarker(w, "f-leak"), "major", "a real leak at ghmirror.go:44", "f-leak"} {
		if !strings.Contains(body, want) {
			t.Fatalf("finding issue body missing %q:\n%s", want, body)
		}
	}
}

// G5: a finding note read from disk carries no issue mapping (create runs), and
// once the issue number is written back, mappedIssueDoc reports it — the local
// half of the idempotency guarantee (a second push skips create, so a re-push
// NEVER duplicates).
func TestFindingIssueMappingIdempotency(t *testing.T) {
	w := mirrorWorkspace(t)
	if _, err := store.CreateNote(w, "a-root", "core", model.NoteFinding, "public repo leaks findings", store.NoteOpts{
		About: "066", Severity: "major", Body: "a real leak at ghmirror.go:44",
	}); err != nil {
		t.Fatalf("create finding: %v", err)
	}

	notes, err := findingNotes(w, "core")
	if err != nil {
		t.Fatalf("findingNotes: %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("got %d finding notes, want 1", len(notes))
	}
	dn := notes[0]
	if dn.id == "" {
		t.Fatalf("finding note has no id")
	}
	if dn.title != "public repo leaks findings" {
		t.Fatalf("finding note title = %q, want the finding title", dn.title)
	}
	if got := mappedIssueDoc(dn.doc); got != 0 {
		t.Fatalf("unmapped finding reports issue %d, want 0", got)
	}
	// The detail must survive into the issue body.
	if !strings.Contains(findingIssueBody(w, dn, "major"), "a real leak at ghmirror.go:44") {
		t.Fatalf("finding detail missing from issue body")
	}

	// Simulate the write-back and re-read: the second push must see the mapping
	// and skip create.
	dn.doc.Front.SetBlock("github", "  issue: 99\n  repo: owner/repo")
	if err := mdstore.WriteFile(dn.path, dn.doc); err != nil {
		t.Fatalf("write back: %v", err)
	}
	reread, err := findingNotes(w, "core")
	if err != nil {
		t.Fatalf("re-read: %v", err)
	}
	if got := mappedIssueDoc(reread[0].doc); got != 99 {
		t.Fatalf("mapped finding reports issue %d, want 99", got)
	}
}

// G5: findingNotes on a project with no findings dir is empty, not an error.
func TestFindingNotesEmpty(t *testing.T) {
	w := mirrorWorkspace(t)
	notes, err := findingNotes(w, "core")
	if err != nil {
		t.Fatalf("findingNotes on empty project: %v", err)
	}
	if len(notes) != 0 {
		t.Fatalf("got %d notes, want 0", len(notes))
	}
}
