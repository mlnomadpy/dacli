package store

import (
	"reflect"
	"testing"

	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/team"
)

// TestTaskDependsOnRoundTripsCommaContainingElement proves CreateTask's
// depends_on list goes through the quote-aware encoder: an element
// containing a top-level comma survives write->read as ONE element, not
// re-split into two.
func TestTaskDependsOnRoundTripsCommaContainingElement(t *testing.T) {
	w := runtimeWorkspace(t)
	if _, err := CreateProject(w, "a-root", "Core", "core", "goal", ""); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	dependsOn := []string{"t-abc, t-def", "t-ghi"}
	task, err := CreateTask(w, "a-root", "core", "Task with odd deps", TaskOpts{DependsOn: dependsOn})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	d, err := mdstore.ReadFile(task.Path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	got := d.Front.GetList("depends_on")
	if !reflect.DeepEqual(got, dependsOn) {
		t.Fatalf("depends_on: got %#v, want %#v", got, dependsOn)
	}
}

// TestShortcutParamsAndRolesRoundTripCommaContainingElement proves
// CreateShortcut's params and roles lists go through the quote-aware
// encoder.
func TestShortcutParamsAndRolesRoundTripCommaContainingElement(t *testing.T) {
	w := runtimeWorkspace(t)
	params := []string{"target", "flags=a, b"}
	roles := []string{"maintainer, reviewer", "root"}
	if err := CreateShortcut(w, "a-root", "deploy", "deploy it", "make deploy", "write", params, roles, "body"); err != nil {
		t.Fatalf("CreateShortcut: %v", err)
	}

	sc, err := LoadShortcut(w, "deploy")
	if err != nil {
		t.Fatalf("LoadShortcut: %v", err)
	}
	if len(sc.Params) != len(params) {
		t.Fatalf("params: got %d elements %#v, want %d elements %#v", len(sc.Params), sc.Params, len(params), params)
	}
	if sc.Params[1].Name != "flags" || sc.Params[1].Default != "a, b" {
		t.Fatalf("params[1]: got name=%q default=%q, want name=%q default=%q", sc.Params[1].Name, sc.Params[1].Default, "flags", "a, b")
	}
	if !reflect.DeepEqual(sc.Roles, roles) {
		t.Fatalf("roles: got %#v, want %#v", sc.Roles, roles)
	}
}

// TestRoleListFieldsRoundTripCommaContainingElement proves CreateRole's
// skills/scope/out_of_scope/shortcuts/escalate_to lists go through the
// quote-aware encoder.
func TestRoleListFieldsRoundTripCommaContainingElement(t *testing.T) {
	w := runtimeWorkspace(t)
	r := team.Role{
		Name:       "maintainer",
		Skills:     []string{"go, testing", "review"},
		Scope:      []string{"internal/**, cmd/**"},
		OutOfScope: []string{"docs/**, secrets/**"},
		Shortcuts:  []string{"deploy, rollback"},
		EscalateTo: []string{"root, human"},
	}
	if err := CreateRole(w, "a-root", r); err != nil {
		t.Fatalf("CreateRole: %v", err)
	}

	got, ok := LoadRole(w, "maintainer")
	if !ok {
		t.Fatalf("LoadRole: not found")
	}
	check := func(field string, want, got []string) {
		t.Helper()
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("%s: got %#v, want %#v", field, got, want)
		}
	}
	check("Skills", r.Skills, got.Skills)
	check("Scope", r.Scope, got.Scope)
	check("OutOfScope", r.OutOfScope, got.OutOfScope)
	check("Shortcuts", r.Shortcuts, got.Shortcuts)
	check("EscalateTo", r.EscalateTo, got.EscalateTo)
}
