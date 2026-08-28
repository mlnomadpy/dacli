package store

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/workspace"
)

func dependencyWorkspace(t *testing.T) *workspace.Workspace {
	t.Helper()
	w, err := workspace.Init(t.TempDir(), "dependency-test")
	if err != nil {
		t.Fatal(err)
	}
	for _, slug := range []string{"p", "q"} {
		if _, err := CreateProject(w, "a-root", strings.ToUpper(slug), slug, "goal", ""); err != nil {
			t.Fatal(err)
		}
	}
	return w
}

func TestDependencyChangeAddsRemovesTypesAndReplaysIdempotently(t *testing.T) {
	w := dependencyWorkspace(t)
	base, _ := CreateTask(w, "a-root", "p", "base", TaskOpts{})
	target, _ := CreateTask(w, "a-root", "p", "target", TaskOpts{})
	change := DependencyChange{Add: []string{base.Slug + ":SS"}}
	if err := ApplyDependencyChange(w, target, change); err != nil {
		t.Fatal(err)
	}
	if err := ApplyDependencyChange(w, target, change); err != nil {
		t.Fatalf("idempotent replay: %v", err)
	}
	got, _ := FindTask(w, target.ID)
	if want := []string{base.ID + ":SS"}; !reflect.DeepEqual(got.Doc.Front.GetList("depends_on"), want) {
		t.Fatalf("depends_on = %#v, want %#v", got.Doc.Front.GetList("depends_on"), want)
	}
	if err := ApplyDependencyChange(w, got, DependencyChange{Remove: []string{base.ID + ":SS"}}); err != nil {
		t.Fatal(err)
	}
	got, _ = FindTask(w, target.ID)
	if deps := got.Doc.Front.GetList("depends_on"); len(deps) != 0 {
		t.Fatalf("remove left dependencies: %#v", deps)
	}
}

func TestDependencyChangeAcceptsProjectQualifiedReference(t *testing.T) {
	w := dependencyWorkspace(t)
	dep, _ := CreateTask(w, "a-root", "q", "shared slug", TaskOpts{})
	_, _ = CreateTask(w, "a-root", "p", "shared slug", TaskOpts{})
	target, _ := CreateTask(w, "a-root", "p", "target", TaskOpts{})
	if err := ApplyDependencyChange(w, target, DependencyChange{Add: []string{"q/shared-slug:FF"}}); err != nil {
		t.Fatal(err)
	}
	got, _ := FindTask(w, target.ID)
	if deps := got.Doc.Front.GetList("depends_on"); len(deps) != 1 || deps[0] != dep.ID+":FF" {
		t.Fatalf("qualified dependency = %#v, want %s:FF", deps, dep.ID)
	}
}

func TestDependencyChangeIgnoresUnrelatedAmbiguousLegacyRefs(t *testing.T) {
	w := dependencyWorkspace(t)
	pBase, _ := CreateTask(w, "a-root", "p", "p base", TaskOpts{})
	pTarget, _ := CreateTask(w, "a-root", "p", "p target", TaskOpts{})
	qBase, _ := CreateTask(w, "a-root", "q", "q base", TaskOpts{})
	qLegacy, _ := CreateTask(w, "a-root", "q", "q legacy", TaskOpts{})
	qLegacy.Doc.Front.SetList("depends_on", []string{"001"})
	if err := SaveTask(qLegacy); err != nil {
		t.Fatal(err)
	}

	if err := ApplyDependencyChange(w, pTarget, DependencyChange{Add: []string{"p/001:FS"}}); err != nil {
		t.Fatalf("project-qualified edit was blocked by unrelated legacy ref: %v", err)
	}
	got, err := FindTask(w, pTarget.ID)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{pBase.ID}; !reflect.DeepEqual(got.Doc.Front.GetList("depends_on"), want) {
		t.Fatalf("depends_on = %#v, want %#v", got.Doc.Front.GetList("depends_on"), want)
	}

	frontier := ReadyFrontier([]*Task{pBase, got, qBase, qLegacy})
	for _, problem := range frontier.Problems {
		if problem.Task.ID == qLegacy.ID {
			t.Fatalf("own-project legacy ref should resolve to q/001, got %v", problem)
		}
	}
}

func TestDependencyChangeValidatesReachableLegacyRefs(t *testing.T) {
	w := dependencyWorkspace(t)
	target, _ := CreateTask(w, "a-root", "p", "target", TaskOpts{})
	broken, _ := CreateTask(w, "a-root", "p", "broken", TaskOpts{})
	broken.Doc.Front.SetList("depends_on", []string{"missing"})
	if err := SaveTask(broken); err != nil {
		t.Fatal(err)
	}

	err := ApplyDependencyChange(w, target, DependencyChange{Add: []string{broken.ID}})
	if err == nil || !strings.Contains(err.Error(), "task 002-broken dependency \"missing\"") {
		t.Fatalf("reachable missing dependency error = %v, want scoped task diagnostic", err)
	}
}

func TestReadyFrontierScopesAmbiguousLegacyRefToOwningTask(t *testing.T) {
	w := dependencyWorkspace(t)
	first, _ := CreateTask(w, "a-root", "q", "duplicate", TaskOpts{})
	second, _ := CreateTask(w, "a-root", "q", "duplicate", TaskOpts{})
	broken, _ := CreateTask(w, "a-root", "q", "broken", TaskOpts{})
	broken.Doc.Front.SetList("depends_on", []string{"duplicate"})
	if err := SaveTask(broken); err != nil {
		t.Fatal(err)
	}

	frontier := ReadyFrontier([]*Task{first, second, broken})
	if len(frontier.Problems) != 1 || frontier.Problems[0].Task.ID != broken.ID || frontier.Problems[0].Ref != "duplicate" {
		t.Fatalf("problems = %#v, want diagnostic scoped to q/%03d-broken", frontier.Problems, broken.Seq)
	}
}

func TestDependencyChangeValidationFailuresDoNotWrite(t *testing.T) {
	w := dependencyWorkspace(t)
	a, _ := CreateTask(w, "a-root", "p", "same", TaskOpts{})
	_, _ = CreateTask(w, "a-root", "q", "same", TaskOpts{})
	b, _ := CreateTask(w, "a-root", "p", "second", TaskOpts{DependsOn: []string{a.ID}})
	before, _ := os.ReadFile(a.Path)

	for name, raw := range map[string]string{
		"ambiguous": "same",
		"missing":   "does-not-exist",
		"self":      a.ID,
		"bad type":  b.ID + ":XY",
		"cycle":     b.ID,
	} {
		t.Run(name, func(t *testing.T) {
			if err := ApplyDependencyChange(w, a, DependencyChange{Add: []string{raw}}); err == nil {
				t.Fatal("invalid dependency change succeeded")
			}
			after, _ := os.ReadFile(a.Path)
			if !reflect.DeepEqual(after, before) {
				t.Fatal("validation failure partially rewrote the task")
			}
		})
	}
}

func TestDependencyChangePreservesAdoptedTaskGitHubMapping(t *testing.T) {
	w := dependencyWorkspace(t)
	dep, _ := CreateTask(w, "a-root", "p", "dependency", TaskOpts{})
	adopted, _ := CreateTask(w, "a-root", "p", "adopted issue", TaskOpts{})
	adopted.Doc.Front.Set("github", "{issue: 717, repo: mlnomadpy/dacli}")
	if err := SaveTask(adopted); err != nil {
		t.Fatal(err)
	}
	before, _ := adopted.Doc.Front.Get("github")
	if err := ApplyDependencyChange(w, adopted, DependencyChange{Add: []string{dep.ID + ":SF"}}); err != nil {
		t.Fatal(err)
	}
	got, _ := FindTask(w, adopted.ID)
	after, _ := got.Doc.Front.Get("github")
	if after != before {
		t.Fatalf("github mapping changed: %q -> %q", before, after)
	}
}

func TestDependencyChangeMigratesStoredLegacyFinishStartAliases(t *testing.T) {
	for _, alias := range []string{"blocks", "hard", "HARD"} {
		t.Run(alias, func(t *testing.T) {
			w := dependencyWorkspace(t)
			base, _ := CreateTask(w, "a-root", "p", "base", TaskOpts{})
			extra, _ := CreateTask(w, "a-root", "p", "extra", TaskOpts{})
			target, _ := CreateTask(w, "a-root", "p", "target", TaskOpts{DependsOn: []string{base.ID}})
			target.Doc.Front.SetList("depends_on", []string{base.ID + ":" + alias})
			if err := SaveTask(target); err != nil {
				t.Fatal(err)
			}

			if err := ApplyDependencyChange(w, target, DependencyChange{Add: []string{extra.ID + ":SS"}}); err != nil {
				t.Fatalf("edit alongside stored :%s dependency: %v", alias, err)
			}
			got, err := FindTask(w, target.ID)
			if err != nil {
				t.Fatal(err)
			}
			if want := []string{base.ID, extra.ID + ":SS"}; !reflect.DeepEqual(got.Doc.Front.GetList("depends_on"), want) {
				t.Fatalf("dependency edit did not migrate :%s: got %#v, want %#v", alias, got.Doc.Front.GetList("depends_on"), want)
			}
		})
	}
}
