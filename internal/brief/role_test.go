package brief_test

import (
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/brief"
	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/team"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// A role's standing instructions used to be parsed and discarded, so every
// role behaved identically — specialization was routing and permissions, not
// expertise. The brief must now carry the role's method (dacli 202).
func TestBriefCarriesTheRolePrompt(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject(w, agentid.RootID, "Core", "core", "", ""); err != nil {
		t.Fatal(err)
	}
	tk, err := store.CreateTask(w, agentid.RootID, "core", "do the thing", store.TaskOpts{Accept: []string{"it works"}})
	if err != nil {
		t.Fatal(err)
	}

	const method = "Reproduce before you fix. A fix for a bug you never observed is a guess."
	if err := store.CreateRole(w, agentid.RootID, team.Role{
		Name:    "fixer",
		Summary: "implements one task and lands it",
		Kind:    "implementer",
	}); err != nil {
		t.Fatal(err)
	}
	// Write the role's body — the part that carries HOW it works.
	rolePath := w.RolePath("fixer")
	d, err := mdstore.ReadFile(rolePath)
	if err != nil {
		t.Fatal(err)
	}
	d.SetSection("fixer", method+"\n")
	if err := mdstore.WriteFile(rolePath, d); err != nil {
		t.Fatal(err)
	}

	roles, err := store.LoadRoles(w)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, r := range roles {
		if r.Name == "fixer" && strings.Contains(r.Prompt, "Reproduce before you fix") {
			found = true
		}
	}
	if !found {
		t.Fatal("parseRole must keep the role file's body as Role.Prompt")
	}

	// Unroled brief: no role section.
	b, err := brief.Assemble(w, tk.Slug, brief.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(render(b), method) {
		t.Error("an unroled brief must not carry a role prompt")
	}

	// Roled brief: the method leads.
	b2, err := brief.Assemble(w, tk.Slug, brief.Options{Role: "fixer"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(render(b2), method) {
		t.Errorf("a roled brief must carry the role's standing instructions; got:\n%s", render(b2))
	}
}

func render(b *brief.Brief) string {
	var sb strings.Builder
	for _, s := range b.Sections {
		sb.WriteString("## " + s.Title + "\n" + s.Content + "\n")
	}
	return sb.String()
}
