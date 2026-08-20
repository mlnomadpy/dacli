package store

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mlnomadpy/dacli/internal/spm"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// DependencyChange is the durable payload shared by the direct command and
// propose-to-sync path. References are accepted in every form FindTask
// supports, then stored as stable task IDs so later ambiguity cannot retarget
// an edge.
type DependencyChange struct {
	Add    []string `json:"add,omitempty"`
	Remove []string `json:"remove,omitempty"`
}

func EncodeDependencyChange(change DependencyChange) (string, error) {
	b, err := json.Marshal(change)
	return string(b), err
}

func DecodeDependencyChange(body string) (DependencyChange, error) {
	var change DependencyChange
	if err := json.Unmarshal([]byte(body), &change); err != nil {
		return change, fmt.Errorf("invalid dependency event: %w", err)
	}
	if len(change.Add) == 0 && len(change.Remove) == 0 {
		return change, fmt.Errorf("dependency change has no additions or removals")
	}
	return change, nil
}

// ApplyDependencyChange validates the complete resulting graph before
// touching frontmatter. Callers serialize the target with WithTask; sync
// already holds that lock. This all-before-write ordering is what makes a
// missing ref, ambiguity, self-edge, bad type, or cycle leave no partial edit.
func ApplyDependencyChange(w *workspace.Workspace, target *Task, change DependencyChange) error {
	return applyDependencyChange(w, target, change, true)
}

// ValidateDependencyChange runs the identical resolver and graph guard without
// changing the target. Proposal writers use it so sync is never handed an
// event that was invalid at creation time.
func ValidateDependencyChange(w *workspace.Workspace, target *Task, change DependencyChange) error {
	return applyDependencyChange(w, target, change, false)
}

func applyDependencyChange(w *workspace.Workspace, target *Task, change DependencyChange, write bool) error {
	all, err := ListTasks(w, "", "")
	if err != nil {
		return err
	}
	idx := NewTaskIndex(all)
	byID := make(map[string]*Task, len(all))
	for _, task := range all {
		byID[task.ID] = task
	}
	byID[target.ID] = target

	type edge struct{ ref, typ string }
	parse := func(raw string) (edge, error) {
		ref, typ := raw, "FS"
		if i := strings.LastIndex(raw, ":"); i > 0 {
			ref, typ = raw[:i], strings.ToUpper(raw[i+1:])
		}
		switch typ {
		case "FS", "SS", "FF", "SF":
		default:
			return edge{}, fmt.Errorf("dependency %q has unsupported type %q (want FS, SS, FF, or SF)", raw, typ)
		}
		dep, err := idx.Find(ref)
		if err != nil {
			return edge{}, err
		}
		if dep.ID == target.ID {
			return edge{}, fmt.Errorf("task %03d-%s cannot depend on itself", target.Seq, target.Slug)
		}
		return edge{dep.ID, typ}, nil
	}

	result := make([]edge, 0, len(target.Deps())+len(change.Add))
	for _, dep := range target.Deps() {
		resolved, rerr := parse(dep.Ref + ":" + dep.Type)
		if rerr != nil {
			return fmt.Errorf("existing dependency %q: %w", dep.Ref, rerr)
		}
		result = append(result, resolved)
	}
	for _, raw := range change.Remove {
		remove, rerr := parse(raw)
		if rerr != nil {
			return fmt.Errorf("remove %q: %w", raw, rerr)
		}
		kept := result[:0]
		for _, current := range result {
			if current != remove {
				kept = append(kept, current)
			}
		}
		result = kept
	}
	for _, raw := range change.Add {
		add, aerr := parse(raw)
		if aerr != nil {
			return fmt.Errorf("add %q: %w", raw, aerr)
		}
		seen := false
		for _, current := range result {
			seen = seen || current == add
		}
		if !seen {
			result = append(result, add)
		}
	}

	nodes := make([]spm.Node, 0, len(byID))
	edges := make([]spm.Edge, 0)
	for id, task := range byID {
		nodes = append(nodes, spm.Node{ID: id})
		deps := task.Deps()
		if id == target.ID {
			deps = nil
			for _, dep := range result {
				deps = append(deps, Dep{Ref: dep.ref, Type: dep.typ})
			}
		}
		for _, dep := range deps {
			typ := strings.ToUpper(dep.Type)
			switch typ {
			case "FS", "SS", "FF", "SF":
			default:
				return fmt.Errorf("task %03d-%s dependency %q has unsupported type %q", task.Seq, task.Slug, dep.Ref, dep.Type)
			}
			resolved, rerr := idx.Find(dep.Ref)
			if rerr != nil {
				return fmt.Errorf("task %03d-%s dependency %q: %w", task.Seq, task.Slug, dep.Ref, rerr)
			}
			edges = append(edges, spm.Edge{From: resolved.ID, To: id, Type: spm.DepType(typ)})
		}
	}
	if _, err := spm.ComputeCPM(nodes, edges); err != nil {
		return err
	}

	encoded := make([]string, 0, len(result))
	for _, dep := range result {
		value := dep.ref
		if dep.typ != "FS" {
			value += ":" + dep.typ
		}
		encoded = append(encoded, value)
	}
	if !write {
		return nil
	}
	target.Doc.Front.SetList("depends_on", encoded)
	return SaveTask(target)
}
