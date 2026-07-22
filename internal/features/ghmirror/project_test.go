package ghmirror

import (
	"testing"

	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
)

// G7: the three mapped fields are exactly Status (single-select over the four
// task folders), Severity (single-select over the total severity value set) and
// Area (free text). This is the mapping the acceptance names.
func TestBoardFieldsMapping(t *testing.T) {
	fields := boardFields()
	byName := map[string]boardFieldDef{}
	for _, f := range fields {
		byName[f.Name] = f
	}
	for _, name := range []string{"Status", "Severity", "Area"} {
		if _, ok := byName[name]; !ok {
			t.Fatalf("board fields missing %q", name)
		}
	}

	status := byName["Status"]
	if status.Kind != fieldSingleSelect {
		t.Errorf("Status must be a single-select")
	}
	if len(status.Options) != len(model.AllStatuses) {
		t.Fatalf("Status options = %v, want the %d task folders", status.Options, len(model.AllStatuses))
	}
	for i, s := range model.AllStatuses {
		if status.Options[i] != string(s) {
			t.Errorf("Status option %d = %q, want %q", i, status.Options[i], s)
		}
	}

	sev := byName["Severity"]
	if sev.Kind != fieldSingleSelect {
		t.Errorf("Severity must be a single-select")
	}
	wantSev := []string{"major", "moderate", "minor", "unspecified"}
	if len(sev.Options) != len(wantSev) {
		t.Fatalf("Severity options = %v, want %v", sev.Options, wantSev)
	}
	for i, s := range wantSev {
		if sev.Options[i] != s {
			t.Errorf("Severity option %d = %q, want %q", i, sev.Options[i], s)
		}
	}

	area := byName["Area"]
	if area.Kind != fieldText {
		t.Errorf("Area must be free text (area slices are dynamic), got kind %d", area.Kind)
	}
	if len(area.Options) != 0 {
		t.Errorf("Area (text) must carry no fixed options, got %v", area.Options)
	}
}

// G7: severityValue is the total, normalized Severity mapping — the three real
// severities pass through (case-folded, trimmed) and anything else falls back to
// "unspecified", never empty. Mirrors severityLabel without the prefix.
func TestSeverityValue(t *testing.T) {
	cases := map[string]string{
		"major":    "major",
		"moderate": "moderate",
		"minor":    "minor",
		"MAJOR":    "major",
		" minor ":  "minor",
		"":         "unspecified",
		"critical": "unspecified",
	}
	for in, want := range cases {
		if got := severityValue(in); got != want {
			t.Errorf("severityValue(%q) = %q, want %q", in, got, want)
		}
	}
}

// G7: areaValue strips the label prefix to the bare slice text, and maps the
// empty label to "" (the signal to skip the Area field).
func TestAreaValue(t *testing.T) {
	cases := map[string]string{
		"area:ghmirror": "ghmirror",
		"area:store":    "store",
		"":              "",
	}
	for in, want := range cases {
		if got := areaValue(in); got != want {
			t.Errorf("areaValue(%q) = %q, want %q", in, got, want)
		}
	}
}

// G7: a task maps to Status (its folder) + Area, with no Severity; a finding
// maps to Severity + Area, with no Status. An empty value is dropped from the
// assignment set so a task never blanks Severity and vice versa.
func TestItemFieldAssignments(t *testing.T) {
	task := &store.Task{Status: model.StatusActive}
	ta := taskItemFields(task, "area:ghmirror").assignments()
	if ta["Status"] != "active" {
		t.Errorf("task Status = %q, want active", ta["Status"])
	}
	if ta["Area"] != "ghmirror" {
		t.Errorf("task Area = %q, want ghmirror", ta["Area"])
	}
	if _, ok := ta["Severity"]; ok {
		t.Errorf("a task must not set Severity, got %q", ta["Severity"])
	}

	fa := findingItemFields("major", "area:store").assignments()
	if fa["Severity"] != "major" {
		t.Errorf("finding Severity = %q, want major", fa["Severity"])
	}
	if fa["Area"] != "store" {
		t.Errorf("finding Area = %q, want store", fa["Area"])
	}
	if _, ok := fa["Status"]; ok {
		t.Errorf("a finding must not set Status, got %q", fa["Status"])
	}

	// A finding with no severity still gets the honest fallback, and an empty
	// area is dropped entirely (not written as a blank Area).
	fb := findingItemFields("", "").assignments()
	if fb["Severity"] != "unspecified" {
		t.Errorf("finding with no severity → Severity %q, want unspecified", fb["Severity"])
	}
	if _, ok := fb["Area"]; ok {
		t.Errorf("empty area must be dropped, got %q", fb["Area"])
	}
}

// G7: projectTitle is stable and slug-derived (so list-by-title adoption is
// deterministic), and ownerOf splits a nameWithOwner, refusing a bare repo.
func TestProjectTitleAndOwner(t *testing.T) {
	if got := projectTitle("core"); got != "dacli: core" {
		t.Errorf("projectTitle(core) = %q, want %q", got, "dacli: core")
	}
	if got := ownerOf("mlnomadpy/dacli"); got != "mlnomadpy" {
		t.Errorf("ownerOf(mlnomadpy/dacli) = %q, want mlnomadpy", got)
	}
	if got := ownerOf("no-slash"); got != "" {
		t.Errorf("ownerOf(no-slash) = %q, want empty (refuse)", got)
	}
	if got := issueURL("mlnomadpy/dacli", 42); got != "https://github.com/mlnomadpy/dacli/issues/42" {
		t.Errorf("issueURL = %q", got)
	}
}

// G7: the item snapshot makes item-add idempotent — an issue already on the
// board maps to its item id (never re-added), and an issue absent from the
// snapshot has no id (add it). Items whose content is not an issue are skipped.
func TestItemIndexByNumber(t *testing.T) {
	items := []ghItem{
		{ID: "PVTI_a"},                                            // no content number — a draft/PR, skip
		{ID: "PVTI_b", Content: struct {
			Number int    `json:"number"`
			URL    string `json:"url"`
			Type   string `json:"type"`
		}{Number: 42, Type: "Issue"}},
		{ID: "PVTI_c", Content: struct {
			Number int    `json:"number"`
			URL    string `json:"url"`
			Type   string `json:"type"`
		}{Number: 7, Type: "Issue"}},
	}
	idx := itemIndexByNumber(items)
	if idx[42] != "PVTI_b" {
		t.Errorf("issue 42 → %q, want PVTI_b", idx[42])
	}
	if idx[7] != "PVTI_c" {
		t.Errorf("issue 7 → %q, want PVTI_c", idx[7])
	}
	if _, ok := idx[99]; ok {
		t.Errorf("issue 99 is not on the board — must have no item id")
	}
	if len(idx) != 2 {
		t.Errorf("index size = %d, want 2 (the no-content item is skipped)", len(idx))
	}
}

// G7: parsing the gh JSON shapes (list / field-list / item-list) is pure and
// covered without a live gh, per the acceptance ("no live gh in tests").
func TestParseProjectList(t *testing.T) {
	data := []byte(`{"projects":[{"number":3,"id":"PVT_x","title":"dacli: core","url":"https://github.com/users/o/projects/3"},{"number":4,"id":"PVT_y","title":"other"}],"totalCount":2}`)
	projects, err := parseProjectList(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(projects) != 2 {
		t.Fatalf("got %d projects, want 2", len(projects))
	}
	found := findProjectByTitle(projects, "dacli: core")
	if found == nil {
		t.Fatalf("dacli: core board not found by title")
	}
	if found.Number != 3 || found.ID != "PVT_x" {
		t.Errorf("found = %+v, want number 3 id PVT_x", *found)
	}
	if findProjectByTitle(projects, "dacli: missing") != nil {
		t.Errorf("a non-existent title must not match — triggers create, not adopt")
	}
}

// G7: field-list parsing exposes the option ids item-edit needs, and optionID
// resolves a value NAME to its option id case-insensitively — returning "" for a
// value that is not an option on the field (the skip-not-fail path).
func TestParseFieldListAndOptionID(t *testing.T) {
	data := []byte(`{"fields":[
		{"id":"PVTF_title","name":"Title","type":"ProjectV2Field"},
		{"id":"PVTF_status","name":"Status","type":"ProjectV2SingleSelectField","options":[{"id":"opt_open","name":"open"},{"id":"opt_active","name":"active"}]},
		{"id":"PVTF_area","name":"Area","type":"ProjectV2Field"}
	]}`)
	fields, err := parseFieldList(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	byName := fieldsByName(fields)
	status, ok := byName["Status"]
	if !ok {
		t.Fatalf("Status field not parsed")
	}
	if !isSingleSelect(status) {
		t.Errorf("Status must be recognized as a single-select")
	}
	if got := optionID(status, "active"); got != "opt_active" {
		t.Errorf("optionID(active) = %q, want opt_active", got)
	}
	if got := optionID(status, "ACTIVE"); got != "opt_active" {
		t.Errorf("optionID must match case-insensitively, got %q", got)
	}
	if got := optionID(status, "done"); got != "" {
		t.Errorf("optionID for a missing option must be empty (skip), got %q", got)
	}
	area, ok := byName["Area"]
	if !ok {
		t.Fatalf("Area field not parsed")
	}
	if isSingleSelect(area) {
		t.Errorf("Area (a plain text field with no options) must not be a single-select")
	}
}

func TestParseItemList(t *testing.T) {
	data := []byte(`{"items":[{"id":"PVTI_1","content":{"type":"Issue","number":10,"url":"u10"}},{"id":"PVTI_2","content":{"type":"Issue","number":11,"url":"u11"}}],"totalCount":2}`)
	items, err := parseItemList(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	idx := itemIndexByNumber(items)
	if idx[10] != "PVTI_1" || idx[11] != "PVTI_2" {
		t.Errorf("item index = %v, want 10→PVTI_1, 11→PVTI_2", idx)
	}
}

// G7: gh's opaque "unknown owner type" failure (a token missing the `project`
// scope) is detected case-insensitively, and any other gh failure passes through
// untouched — so the actionable hint fires only for the real missing-scope case.
func TestMissingProjectScope(t *testing.T) {
	positives := []string{
		"unknown owner type",
		"gh: unknown owner type",
		"Error: Unknown Owner Type\n",
		"failed to run git: UNKNOWN OWNER TYPE",
	}
	for _, s := range positives {
		if !missingProjectScope(s) {
			t.Errorf("missingProjectScope(%q) = false, want true (the missing-scope signal)", s)
		}
	}
	negatives := []string{
		"",
		"GraphQL: Could not resolve to a Repository",
		"HTTP 404: Not Found",
		"owner type is known",
	}
	for _, s := range negatives {
		if missingProjectScope(s) {
			t.Errorf("missingProjectScope(%q) = true, want false (not a missing-scope failure)", s)
		}
	}
}

// G7: the stored board mapping round-trips through the project frontmatter, so a
// re-run reuses the same board (idempotent, no second create) with zero network.
func TestStoredProjectRoundTrip(t *testing.T) {
	p := &store.Project{Doc: &mdstore.Doc{}, Slug: "core"}

	// Unbound: no board, so ensureProject would resolve and create.
	if pr := storedProject(p); pr.Number != 0 || pr.ID != "" {
		t.Fatalf("unbound project must report no board, got %+v", pr)
	}

	// Bind it and read it back — the load-bearing idempotency: a re-run sees the
	// board by number+id without a list/create call.
	p.Doc.Front.SetBlock("github_project", storedProjectBlock(ghProject{Number: 5, ID: "PVT_z"}, "mlnomadpy"))
	pr := storedProject(p)
	if pr.Number != 5 || pr.ID != "PVT_z" {
		t.Fatalf("stored board = %+v, want number 5 id PVT_z", pr)
	}
	if got := blockValue(func() string { b, _ := p.Doc.Front.GetBlock("github_project"); return b }(), "owner"); got != "mlnomadpy" {
		t.Errorf("stored owner = %q, want mlnomadpy", got)
	}

	// A block missing the id must NOT be treated as a resolved board (re-resolve).
	p2 := &store.Project{Doc: &mdstore.Doc{}, Slug: "core"}
	p2.Doc.Front.SetBlock("github_project", "  number: 9\n  owner: o")
	if pr := storedProject(p2); pr.ID != "" {
		t.Errorf("a block with no id must yield no board id, got %q", pr.ID)
	}
}
