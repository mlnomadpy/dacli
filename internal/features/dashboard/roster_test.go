package dashboard

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// roleByName picks one role out of a roster response, failing the test when the
// roster does not carry it — a missing role is the failure, not a nil deref.
func roleByName(t *testing.T, roles []roleView, name string) roleView {
	t.Helper()
	for _, r := range roles {
		if r.Name == name {
			return r
		}
	}
	t.Fatalf("role %q not in roster %+v", name, roles)
	return roleView{}
}

// TestAPIRoles is the roster contract (dacli 226): every mechanical lever a role
// pulls — kind, grant, runtime/model, caps, scope, skills — plus the live agent
// census that says how much of its WIP budget is spent.
func TestAPIRoles(t *testing.T) {
	w := dashboardEnv(t)
	h := newHandler(w)

	var resp rolesResponse
	getJSON(t, h, "/api/roles", &resp)

	if resp.Generated == "" {
		t.Errorf("generated is empty")
	}
	if len(resp.Roles) != 2 {
		t.Fatalf("roles = %d, want 2", len(resp.Roles))
	}
	// Sorted by name so the list is stable across polls.
	if resp.Roles[0].Name != "builder" || resp.Roles[1].Name != "maintainer" {
		t.Errorf("roster order = %q,%q, want builder,maintainer", resp.Roles[0].Name, resp.Roles[1].Name)
	}

	b := roleByName(t, resp.Roles, "builder")
	if b.Summary != "writes the code" || b.Kind != "implementer" || b.Grant != "rw" {
		t.Errorf("builder identity = %+v", b)
	}
	if b.Runtime != "claude" || b.Model != "sonnet" {
		t.Errorf("builder routing = runtime %q model %q, want claude/sonnet", b.Runtime, b.Model)
	}
	if b.WIP != 1 || b.MaxPoints != 5 {
		t.Errorf("builder caps = wip %d max_points %v, want 1/5", b.WIP, b.MaxPoints)
	}
	if len(b.Scope) != 1 || b.Scope[0] != "internal/**" {
		t.Errorf("builder scope = %v", b.Scope)
	}
	if len(b.OutOfScope) != 1 || b.OutOfScope[0] != "internal/agentid/**" {
		t.Errorf("builder out_of_scope = %v", b.OutOfScope)
	}
	if len(b.Skills) != 1 || b.Skills[0] != "go" {
		t.Errorf("builder skills = %v", b.Skills)
	}
	if len(b.Shortcuts) != 1 || len(b.EscalateTo) != 1 || b.EscalateTo[0] != "maintainer" {
		t.Errorf("builder shortcuts/escalation = %v / %v", b.Shortcuts, b.EscalateTo)
	}

	// The census counts NON-RETIRED agents only: the env spawned two builders and
	// retired one, so the cap of 1 is exactly reached — not exceeded by the file
	// count, which would be the bug of counting agent files instead of slots.
	if b.ActiveAgents != 1 {
		t.Errorf("builder active_agents = %d, want 1 (the retired one freed its slot)", b.ActiveAgents)
	}
	if !b.WIPExceeded {
		t.Errorf("builder wip_exceeded = false, want true (1 active against a cap of 1)")
	}

	m := roleByName(t, resp.Roles, "maintainer")
	if m.ActiveAgents != 0 {
		t.Errorf("maintainer active_agents = %d, want 0", m.ActiveAgents)
	}
	if m.WIPExceeded {
		t.Errorf("maintainer wip_exceeded = true, want false (uncapped role)")
	}
	// An uncapped role's empty lists must marshal as [], never null: the shape is
	// a contract, and a field that is sometimes a list is two contracts.
	if m.Scope == nil || m.Skills == nil || m.EscalateTo == nil {
		t.Errorf("maintainer emitted a null list: %+v", m)
	}
}

// TestRolesHasPromptDistinguishesDescribedFromDefined proves the roster reports
// whether a role carries standing instructions — the difference between a role
// that is merely described by its frontmatter and one that actually defines how
// its agents work. store.CreateRole writes metadata only, so has_prompt is false
// until a body is added below the frontmatter.
func TestRolesHasPromptDistinguishesDescribedFromDefined(t *testing.T) {
	w := dashboardEnv(t)
	h := newHandler(w)

	var before rolesResponse
	getJSON(t, h, "/api/roles", &before)
	if roleByName(t, before.Roles, "builder").HasPrompt {
		t.Errorf("has_prompt = true for a metadata-only role")
	}

	d, err := mdstore.ReadFile(w.RolePath("builder"))
	if err != nil {
		t.Fatalf("read role: %v", err)
	}
	d.Sections = append(d.Sections, mdstore.Section{
		Level: 2, Title: "Method", Content: "Write the failing test first.\n",
	})
	if err := mdstore.WriteFile(w.RolePath("builder"), d); err != nil {
		t.Fatalf("write role: %v", err)
	}

	var after rolesResponse
	getJSON(t, h, "/api/roles", &after)
	b := roleByName(t, after.Roles, "builder")
	if !b.HasPrompt {
		t.Errorf("has_prompt = false for a role with standing instructions")
	}
	// The body itself is never served — the roster is a list, not a reader.
	if roleBodyLeaked(b) {
		t.Errorf("role body leaked into the roster: %+v", b)
	}
}

// roleBodyLeaked reports whether any roleView string field carries the role
// file's prose. Guards the deliberate omission: adding a Prompt field later must
// be a decision, not an accident.
func roleBodyLeaked(r roleView) bool {
	for _, s := range []string{r.Name, r.Summary, r.Kind, r.Grant, r.Runtime, r.Model} {
		if s == "Write the failing test first." {
			return true
		}
	}
	return false
}

// TestAPIStateEmbedsRoles proves the combined snapshot carries the same roster
// the standalone surface serves, so the SPA's single poll needs no second
// request and the two contracts cannot drift.
func TestAPIStateEmbedsRoles(t *testing.T) {
	w := dashboardEnv(t)
	h := newHandler(w)

	var state dashboardState
	getJSON(t, h, "/api/state", &state)
	var standalone rolesResponse
	getJSON(t, h, "/api/roles", &standalone)

	// Compare the encoded rosters: a field added to roleView without a matching
	// embed would show up here rather than needing this test to be updated.
	embedded, err := json.Marshal(state.Roles)
	if err != nil {
		t.Fatalf("marshal embedded: %v", err)
	}
	direct, err := json.Marshal(standalone.Roles)
	if err != nil {
		t.Fatalf("marshal standalone: %v", err)
	}
	if string(embedded) != string(direct) {
		t.Errorf("embedded roster differs from /api/roles:\n%s\n%s", embedded, direct)
	}
	b := roleByName(t, state.Roles, "builder")
	if b.ActiveAgents != 1 || !b.WIPExceeded {
		t.Errorf("embedded builder = %+v, want the same census as /api/roles", b)
	}
}

// TestRolesEmptyWorkspaceIsZeroSafe proves a workspace with no roles yet answers
// 200 with an empty list rather than 500ing or emitting null — a fresh `dacli
// init` has projects before it has a team.
func TestRolesEmptyWorkspaceIsZeroSafe(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "a-root")
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	h := newHandler(w)

	var resp rolesResponse
	getJSON(t, h, "/api/roles", &resp)
	if len(resp.Roles) != 0 {
		t.Errorf("roles = %d, want 0", len(resp.Roles))
	}
	if code := do(t, h, "GET", "/api/roles", "localhost"); code != http.StatusOK {
		t.Errorf("GET /api/roles on an empty workspace = %d, want 200", code)
	}
}
