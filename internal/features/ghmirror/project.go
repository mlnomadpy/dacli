package ghmirror

// G7: sync mirrored issues into a GitHub Project (v2) board with mapped fields.
//
// The board is the third outbound projection (after task issues and finding
// issues), and it obeys the same two invariants the rest of this slice does:
//
//   - Disclosure-gated. Adding an issue to a Project v2 board publishes nothing
//     new that the issue itself doesn't already publish, but it is still an
//     operator-triggered outbound projection, so it rides the SAME disclosure
//     gate as push (a public repo needs recorded per-project consent).
//   - Idempotent. The board (by stored number, then by title), its fields (by
//     name), and its items (by the issue number the item's content points at)
//     are all resolved before anything is created, so a re-run adds no duplicate
//     board, field, or item. This is the load-bearing property the acceptance
//     names explicitly: "re-run does not duplicate items."
//
// Field mapping (GITHUB.md § 2) is the pure, unit-tested core — no live gh:
//
//	dacli                       →  Project field
//	task status folder          →  Status   (single-select: open/active/blocked/done)
//	finding severity            →  Severity (single-select: major/moderate/minor/unspecified)
//	area: label (task/finding)  →  Area     (free text — area slices are dynamic)
//
// The gh Project v2 surface (project create / list / field-list / field-create /
// item-list / item-add / item-edit) is an assumption until `github doctor`
// probes it, exactly like the issue surface — so setting a field value is
// best-effort: a value that is not a valid option on the resolved field (e.g. a
// board that already carries the built-in "Status" field with Todo/In-Progress/
// Done) is skipped rather than failing the whole sync.

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// --- field definitions (the mapping, unit-tested without gh) ---

// boardFieldKind is how dacli stores a mapped field on the board: a
// single-select (a fixed, enumerable value set) or free text (dynamic values).
type boardFieldKind int

const (
	fieldSingleSelect boardFieldKind = iota
	fieldText
)

// boardFieldDef is one dacli → Project field mapping.
type boardFieldDef struct {
	Name    string
	Kind    boardFieldKind
	Options []string // single-select option names, in canonical order; nil for text
}

// severityOptions is the canonical, total value set of the Severity field —
// the three real severities plus the honest fallback (mirrors severityLabel).
var severityOptions = []string{"major", "moderate", "minor", "unspecified"}

// boardFields are the three dacli → Project field mappings (GITHUB.md § 2):
// Status (from the task folder), Severity (from a finding's severity), and Area
// (from the area: label). Status/Severity are single-selects over a fixed value
// set; Area is free text because area slices are derived per note and unbounded.
func boardFields() []boardFieldDef {
	var statuses []string
	for _, s := range model.AllStatuses {
		statuses = append(statuses, string(s))
	}
	return []boardFieldDef{
		{Name: "Status", Kind: fieldSingleSelect, Options: statuses},
		{Name: "Severity", Kind: fieldSingleSelect, Options: severityOptions},
		{Name: "Area", Kind: fieldText},
	}
}

// severityValue maps a finding's raw severity to its Severity field value —
// severityLabel's normalization (case-fold, trim, total fallback to
// "unspecified") without the label prefix. Total and unit-testable.
func severityValue(severity string) string {
	return strings.TrimPrefix(severityLabel(severity), "severity:")
}

// areaValue maps an area: label to its Area field value — the bare slice
// ("area:ghmirror" → "ghmirror"), or "" for the empty label (the signal to skip
// the field). Shared so a task (slice from the project) and a finding (slice
// from the finding body) set the identical Area text.
func areaValue(label string) string {
	return strings.TrimPrefix(label, "area:")
}

// --- item field assignments (the per-issue mapping) ---

// boardItemFields is the set of field → value assignments for one mirrored
// issue. A task carries a folder status but no severity; a finding carries a
// severity but no folder — an empty value is skipped, never written as blank.
type boardItemFields struct {
	Status   string
	Severity string
	Area     string
}

// assignments returns the non-empty field → value pairs to write for this item,
// keyed by the field Name in boardFields. An empty value means "leave the field
// untouched", so a task never blanks the Severity field and vice versa.
func (b boardItemFields) assignments() map[string]string {
	m := map[string]string{}
	if b.Status != "" {
		m["Status"] = b.Status
	}
	if b.Severity != "" {
		m["Severity"] = b.Severity
	}
	if b.Area != "" {
		m["Area"] = b.Area
	}
	return m
}

// taskItemFields maps a task to its board fields: Status from the task's folder,
// Area from the project-derived area label. A task carries no severity.
func taskItemFields(t *store.Task, areaLbl string) boardItemFields {
	return boardItemFields{Status: string(t.Status), Area: areaValue(areaLbl)}
}

// findingItemFields maps a finding to its board fields: Severity from the
// finding's severity, Area from the area label derived from its detail. A
// finding carries no folder status.
func findingItemFields(severity, areaLbl string) boardItemFields {
	return boardItemFields{Severity: severityValue(severity), Area: areaValue(areaLbl)}
}

// --- gh project JSON shapes + pure parsers (unit-tested without gh) ---

// ghProject is the subset of a Project v2 dacli reads: its number and node id
// (both needed to add items and edit fields) plus title/url for display.
type ghProject struct {
	Number int    `json:"number"`
	ID     string `json:"id"`
	Title  string `json:"title"`
	URL    string `json:"url"`
}

// parseProjectList parses `gh project list --format json` output.
func parseProjectList(data []byte) ([]ghProject, error) {
	var v struct {
		Projects []ghProject `json:"projects"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return v.Projects, nil
}

// findProjectByTitle returns the project whose title matches exactly, or nil —
// the adoption path that reuses an existing board instead of creating a second.
func findProjectByTitle(projects []ghProject, title string) *ghProject {
	for i := range projects {
		if projects[i].Title == title {
			return &projects[i]
		}
	}
	return nil
}

// ghField is a Project v2 field: its node id, name, type, and (for a
// single-select) its options with their ids — the map from a value NAME to the
// option id item-edit needs.
type ghField struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Options []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"options"`
}

// parseFieldList parses `gh project field-list --format json` output.
func parseFieldList(data []byte) ([]ghField, error) {
	var v struct {
		Fields []ghField `json:"fields"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return v.Fields, nil
}

// fieldsByName indexes fields by name, so ensureFields reuses an existing field
// (including the built-in "Status") instead of creating a duplicate.
func fieldsByName(fields []ghField) map[string]ghField {
	m := make(map[string]ghField, len(fields))
	for _, f := range fields {
		m[f.Name] = f
	}
	return m
}

// isSingleSelect reports whether a field carries an enumerable option set —
// either GitHub types it a single-select, or it already lists options. A text
// field has neither, so item-edit writes it with --text, not an option id.
func isSingleSelect(f ghField) bool {
	return strings.Contains(f.Type, "SingleSelect") || len(f.Options) > 0
}

// optionID returns the option id whose name matches value (case-insensitively),
// or "" if the value is not an option on this field. "" means skip: a value
// that is not a valid option (e.g. dacli's "active" against a built-in Status
// field of Todo/In-Progress/Done) is left unset rather than erroring the sync.
func optionID(f ghField, value string) string {
	for _, o := range f.Options {
		if strings.EqualFold(o.Name, value) {
			return o.ID
		}
	}
	return ""
}

// ghItem is a board item: its node id and the issue its content points at (the
// number is the idempotency key — an item already pointing at issue N is not
// re-added).
type ghItem struct {
	ID      string `json:"id"`
	Content struct {
		Number int    `json:"number"`
		URL    string `json:"url"`
		Type   string `json:"type"`
	} `json:"content"`
}

// parseItemList parses `gh project item-list --format json` output.
func parseItemList(data []byte) ([]ghItem, error) {
	var v struct {
		Items []ghItem `json:"items"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return v.Items, nil
}

// itemIndexByNumber maps each board item's issue number to its item id — the
// snapshot that makes item-add idempotent: an issue already on the board is
// reused by id, never added a second time. Items whose content is not an issue
// (no number) are skipped.
func itemIndexByNumber(items []ghItem) map[int]string {
	m := map[int]string{}
	for _, it := range items {
		if it.Content.Number > 0 {
			m[it.Content.Number] = it.ID
		}
	}
	return m
}

// --- board identity (pure) ---

// projectTitle is the board's stable title, derived from the dacli project slug
// so list-by-title adoption is deterministic across runs.
func projectTitle(slug string) string {
	return "dacli: " + slug
}

// ownerOf extracts the owner login from a "owner/repo" nameWithOwner — the
// --owner every gh project subcommand needs. Returns "" when repo has no owner
// segment, so the caller refuses rather than calling gh with an empty owner.
func ownerOf(repo string) string {
	owner, _, found := strings.Cut(repo, "/")
	if !found {
		return ""
	}
	return strings.TrimSpace(owner)
}

// issueURL renders the canonical issue URL from a repo and number — what
// item-add binds a board item to.
func issueURL(repo string, num int) string {
	return fmt.Sprintf("https://github.com/%s/issues/%d", repo, num)
}

// --- stored board mapping (idempotency, local + versioned) ---

// storedProjectBlock renders the `github_project:` frontmatter block that binds
// a dacli project to its board — number and node id (both needed for item-add /
// item-edit) plus the owner. Local, diffable, versioned with the project, so a
// re-run reuses the same board without a list call.
func storedProjectBlock(pr ghProject, owner string) string {
	return fmt.Sprintf("  number: %d\n  id: %s\n  owner: %s", pr.Number, pr.ID, owner)
}

// blockValue reads a single `key: value` line from a frontmatter block (the
// same shape mappedIssueDoc reads the github: block with).
func blockValue(block, key string) string {
	for _, line := range strings.Split(block, "\n") {
		if k, v, found := strings.Cut(strings.TrimSpace(line), ":"); found && strings.TrimSpace(k) == key {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// storedProject reads a project's bound board from its `github_project:` block,
// or a zero ghProject when unbound. The board is reused only when BOTH the
// number and node id are present — either missing means re-resolve.
func storedProject(p *store.Project) ghProject {
	block, ok := p.Doc.Front.GetBlock("github_project")
	if !ok {
		return ghProject{}
	}
	num, _ := strconv.Atoi(blockValue(block, "number"))
	return ghProject{Number: num, ID: blockValue(block, "id")}
}

// writeStoredProject binds the board to the project file (idempotent: it rewrites
// only when the stored block actually changes, so a re-run churns no mtime).
func writeStoredProject(p *store.Project, pr ghProject, owner string) error {
	desired := storedProjectBlock(pr, owner)
	if cur, _ := p.Doc.Front.GetBlock("github_project"); cur == desired {
		return nil
	}
	p.Doc.Front.SetBlock("github_project", desired)
	return mdstore.WriteFile(p.Path, p.Doc)
}

// --- missing project scope (an actionable error, not gh's cryptic one) ---

// Projects v2 needs the `project` token scope, granted SEPARATELY from repo. A
// token authed for repo but not project fails every `gh project` subcommand
// with the opaque "unknown owner type" (gh cannot resolve the owner without the
// scope, and surfaces no hint about why). We detect that signal and translate.
const projectScopeHint = "gh's token is missing the 'project' scope Projects v2 needs (granted separately from repo) — run `gh auth refresh -s project` and retry"

// missingProjectScope reports whether gh's combined output is the "unknown owner
// type" failure it emits when the token lacks the `project` scope. Matched
// case-insensitively so a wording/case drift in gh does not silently regress the
// detection back to surfacing the cryptic message.
func missingProjectScope(ghOutput string) bool {
	return strings.Contains(strings.ToLower(ghOutput), "unknown owner type")
}

// ghProjectCmd runs a `gh project` subcommand, translating the opaque "unknown
// owner type" failure (a missing `project` token scope) into an actionable error
// that names the fix, instead of surfacing gh's cryptic message. Every project
// gh call routes through here so the hint appears no matter which subcommand is
// the first to hit the missing scope.
func ghProjectCmd(w *workspace.Workspace, args ...string) (string, error) {
	out, err := gh(w, args...)
	if err != nil && missingProjectScope(out) {
		return out, fmt.Errorf("%s (gh: %s)", projectScopeHint, out)
	}
	return out, err
}

// --- the command ---

// cmdProject creates or links a GitHub Project v2 board for the linked repo and
// adds every mirrored issue to it with its mapped Status/Severity/Area fields.
// Operator-triggered and disclosure-gated like the other projections; idempotent
// at every step (board, fields, items), so a re-run never duplicates.
func cmdProject(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if len(f.Pos) == 0 {
		return clikit.Usagef("usage: dacli github project <project>")
	}
	p, err := store.LoadProject(w, f.Pos[0])
	if err != nil {
		return err
	}
	repo, _ := p.Doc.Front.Get("github_repo")
	if repo == "" {
		return clikit.Usagef("project %s is not linked — `dacli github link %s` first", p.Slug, p.Slug)
	}
	// A board is an outbound projection, so it rides the SAME disclosure gate as
	// push: a repo flipped public after linking re-trips it here too.
	if err := disclosureGate(w, p); err != nil {
		return err
	}
	owner := ownerOf(repo)
	if owner == "" {
		return fmt.Errorf("cannot derive owner from repo %q — expected owner/name", repo)
	}

	// 1. Resolve the board: stored number → list-by-title → create. Idempotent.
	proj, err := ensureProject(w, p, owner)
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "board: %s (#%d) %s\n", proj.Title, proj.Number, proj.URL)

	// 2. Ensure the three mapped fields exist (reusing any by name, incl. a
	//    built-in Status). Best-effort — a field that cannot be created just
	//    leaves its values unset rather than aborting the sync.
	fields, err := ensureFields(w, owner, proj)
	if err != nil {
		return err
	}

	// 3. Snapshot the board's existing items ONCE — the idempotency key set that
	//    makes item-add add no duplicate on a re-run.
	itemsOut, err := ghProjectCmd(w, "project", "item-list", strconv.Itoa(proj.Number), "--owner", owner, "--format", "json", "--limit", "1000")
	if err != nil {
		return fmt.Errorf("gh project item-list: %v (%s)", err, itemsOut)
	}
	items, err := parseItemList([]byte(itemsOut))
	if err != nil {
		return fmt.Errorf("parse item list: %v", err)
	}
	itemByNum := itemIndexByNumber(items)

	// 4. Every mirrored TASK issue → a board item with Status + Area.
	tasks, err := store.ListTasks(w, p.Slug, "")
	if err != nil {
		return err
	}
	taskArea := areaLabel(p.Slug)
	added, synced, unmirrored := 0, 0, 0
	for _, t := range tasks {
		num := mappedIssue(t)
		if num == 0 {
			unmirrored++
			continue // not on GitHub yet — `dacli github push` first
		}
		itemID, isNew, err := ensureItem(w, owner, proj, repo, num, itemByNum)
		if err != nil {
			return err
		}
		if isNew {
			added++
		}
		setItemFields(w, proj, fields, itemID, taskItemFields(t, taskArea))
		synced++
	}

	// 5. Every mirrored FINDING issue → a board item with Severity + Area.
	findings, err := findingNotes(w, p.Slug)
	if err != nil {
		return err
	}
	for _, dn := range findings {
		num := mappedIssueDoc(dn.doc)
		if num == 0 {
			continue // finding not filed as its own issue — `github push --findings-as-issues`
		}
		itemID, isNew, err := ensureItem(w, owner, proj, repo, num, itemByNum)
		if err != nil {
			return err
		}
		if isNew {
			added++
		}
		severity, _ := dn.doc.Front.Get("severity")
		area := areaLabel(areaSlice(findingText(dn.doc)))
		setItemFields(w, proj, fields, itemID, findingItemFields(severity, area))
		synced++
	}

	fmt.Fprintf(ctx.Stdout, "items: %d added, %d field-synced", added, synced)
	if unmirrored > 0 {
		fmt.Fprintf(ctx.Stdout, ", %d task(s) not yet mirrored (run `github push` first)", unmirrored)
	}
	fmt.Fprintln(ctx.Stdout)
	return nil
}

// ensureProject resolves the board idempotently: the stored number+id first (no
// network), then a list-by-title adoption (so a board created out-of-band or by
// a prior run whose write-back was lost is reused, not duplicated), and only
// then a create. The resolved board is written back to the project file.
func ensureProject(w *workspace.Workspace, p *store.Project, owner string) (ghProject, error) {
	if pr := storedProject(p); pr.Number > 0 && pr.ID != "" {
		pr.Title = projectTitle(p.Slug)
		return pr, nil
	}
	title := projectTitle(p.Slug)
	// Adoption by title: reuse an existing board before creating a second one.
	if out, err := ghProjectCmd(w, "project", "list", "--owner", owner, "--format", "json", "--limit", "1000"); err == nil {
		if list, perr := parseProjectList([]byte(out)); perr == nil {
			if found := findProjectByTitle(list, title); found != nil {
				if err := writeStoredProject(p, *found, owner); err != nil {
					return ghProject{}, err
				}
				return *found, nil
			}
		}
	}
	out, err := ghProjectCmd(w, "project", "create", "--owner", owner, "--title", title, "--format", "json")
	if err != nil {
		return ghProject{}, fmt.Errorf("gh project create: %v (%s)", err, out)
	}
	var pr ghProject
	if err := json.Unmarshal([]byte(out), &pr); err != nil {
		return ghProject{}, fmt.Errorf("parse project create output %q: %v", out, err)
	}
	if pr.Number == 0 || pr.ID == "" {
		return ghProject{}, fmt.Errorf("could not parse board number/id from gh output %q", out)
	}
	if err := writeStoredProject(p, pr, owner); err != nil {
		return ghProject{}, err
	}
	return pr, nil
}

// ensureFields makes sure the three mapped fields exist, reusing any field that
// already carries the name (so the built-in "Status" field is adopted, never
// duplicated) and creating the rest. Best-effort per field: a create failure is
// swallowed so the sync proceeds — the missing field's values are simply left
// unset. Returns the resolved fields by name (with their option ids).
func ensureFields(w *workspace.Workspace, owner string, proj ghProject) (map[string]ghField, error) {
	out, err := ghProjectCmd(w, "project", "field-list", strconv.Itoa(proj.Number), "--owner", owner, "--format", "json", "--limit", "100")
	if err != nil {
		return nil, fmt.Errorf("gh project field-list: %v (%s)", err, out)
	}
	list, err := parseFieldList([]byte(out))
	if err != nil {
		return nil, fmt.Errorf("parse field list: %v", err)
	}
	byName := fieldsByName(list)
	for _, def := range boardFields() {
		if _, ok := byName[def.Name]; ok {
			continue // reuse an existing field (incl. the built-in Status)
		}
		args := []string{"project", "field-create", strconv.Itoa(proj.Number), "--owner", owner, "--name", def.Name, "--format", "json"}
		switch def.Kind {
		case fieldSingleSelect:
			args = append(args, "--data-type", "SINGLE_SELECT", "--single-select-options", strings.Join(def.Options, ","))
		case fieldText:
			args = append(args, "--data-type", "TEXT")
		}
		cout, cerr := ghProjectCmd(w, args...)
		if cerr != nil {
			continue // best-effort: leave this field's values unset rather than abort
		}
		var fld ghField
		if json.Unmarshal([]byte(cout), &fld) == nil && fld.ID != "" {
			byName[def.Name] = fld
		}
	}
	return byName, nil
}

// ensureItem returns the board item id for a mirrored issue, adding it only if
// the board does not already carry it (the snapshot in byNum is the idempotency
// key). A freshly added item is recorded in byNum so the same issue referenced
// twice in one run is not added twice.
func ensureItem(w *workspace.Workspace, owner string, proj ghProject, repo string, num int, byNum map[int]string) (string, bool, error) {
	if id, ok := byNum[num]; ok {
		return id, false, nil
	}
	out, err := ghProjectCmd(w, "project", "item-add", strconv.Itoa(proj.Number), "--owner", owner, "--url", issueURL(repo, num), "--format", "json")
	if err != nil {
		return "", false, fmt.Errorf("gh project item-add #%d: %v (%s)", num, err, out)
	}
	var it struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(out), &it); err != nil || it.ID == "" {
		return "", false, fmt.Errorf("could not parse item id from gh output %q", out)
	}
	byNum[num] = it.ID
	return it.ID, true, nil
}

// setItemFields writes an item's mapped field values. Each assignment is
// best-effort and idempotent (item-edit to the same value is a no-op effect): a
// field dacli could not resolve, or a single-select value that is not an option
// on the resolved field, is skipped rather than failing the sync.
func setItemFields(w *workspace.Workspace, proj ghProject, fields map[string]ghField, itemID string, vals boardItemFields) {
	if itemID == "" {
		return
	}
	for name, value := range vals.assignments() {
		fld, ok := fields[name]
		if !ok || fld.ID == "" {
			continue
		}
		args := []string{"project", "item-edit", "--id", itemID, "--project-id", proj.ID, "--field-id", fld.ID}
		if isSingleSelect(fld) {
			opt := optionID(fld, value)
			if opt == "" {
				continue // value not an option on this field — leave it unset
			}
			args = append(args, "--single-select-option-id", opt)
		} else {
			args = append(args, "--text", value)
		}
		_, _ = ghProjectCmd(w, args...)
	}
}
