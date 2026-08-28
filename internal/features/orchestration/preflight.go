package orchestration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/team"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// cyclePreflightRunner is deliberately narrower than runner. Production and
// dry-run runners implement it; old test runners need not grow synthetic
// preflight behavior just to exercise unrelated cycle phases.
type cyclePreflightRunner interface {
	runPreflight(label string, args ...string) (string, error)
}

type cyclePreflightResult struct {
	SchemaVersion int                   `json:"schema_version"`
	Project       string                `json:"project"`
	Cycle         int                   `json:"cycle"`
	Verdict       string                `json:"verdict"`
	GeneratedAt   time.Time             `json:"generated_at"`
	Phases        []cyclePreflightPhase `json:"phases"`
}

type cyclePreflightPhase struct {
	Phase       string                    `json:"phase"`
	Task        string                    `json:"task,omitempty"`
	Role        string                    `json:"role,omitempty"`
	Runtime     string                    `json:"runtime,omitempty"`
	Model       string                    `json:"model,omitempty"`
	Grant       string                    `json:"grant,omitempty"`
	Claims      []string                  `json:"claims,omitempty"`
	Capacity    *team.TaskCapacityVerdict `json:"capacity,omitempty"`
	Verdict     string                    `json:"verdict"`
	Evidence    string                    `json:"evidence,omitempty"`
	Remediation string                    `json:"remediation,omitempty"`
}

func cyclePreflightFile(w *workspace.Workspace, project string) string {
	return filepath.Join(w.Root, workspace.Dir, "loop", project+"-preflight.json")
}

func (d *driver) emitCyclePreflight(result cyclePreflightResult) {
	raw, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return
	}
	d.logf("cycle preflight JSON: %s", string(raw))
	if d.cfg.dryRun {
		return
	}
	path := cyclePreflightFile(d.w, d.cfg.project)
	if os.MkdirAll(filepath.Dir(path), 0o755) == nil {
		_ = writeStateFile(path, string(append(raw, '\n')))
	}
}

// preflightCycle resolves the complete worker boundary before the first
// implementation spawn. It intentionally starts with facts available without
// launching a coding agent: exact task capacity and claims, then reviewer and
// implementation runtime contracts through the read-only preflight command.
// Later #867 slices add landing/check and verification-directory probes to the
// same versioned result rather than inventing another preflight format.
func (d *driver) preflightCycle(ready []*store.Task) error {
	result := cyclePreflightResult{SchemaVersion: 1, Project: d.cfg.project, Cycle: d.gov.Cycle() + 1, Verdict: "pass", GeneratedAt: d.now().UTC()}
	wave := selectClaimCompatibleWave(d.w.Root, ready, d.cfg.width)
	implementationRoles := map[string]bool{}
	fallbackRole := d.buildRole()

	for _, task := range wave.Tasks {
		roleName := fallbackRole
		role, found := store.LoadRole(d.w, roleName)
		if !found && d.cfg.implRoleExplicit {
			phase := cyclePreflightPhase{Phase: "implementation", Task: task.ID, Role: roleName, Verdict: "refuse", Evidence: "configured role is missing", Remediation: "configure the implementation role before starting the loop"}
			result.Phases = append(result.Phases, phase)
			result.Verdict = "refuse"
			d.emitCyclePreflight(result)
			return clikit.Refusedf("cycle preflight: implementation role %s does not exist; configure it before starting the loop", roleName)
		}
		if !found {
			result.Phases = append(result.Phases, cyclePreflightPhase{Phase: "implementation", Task: task.ID, Role: roleName, Verdict: "unobserved", Evidence: "fallback role is not declared; launch retains legacy resolution"})
			implementationRoles[roleName] = true
			continue
		}
		te := 0.0
		sized := false
		if estimate, ok := task.Estimate(); ok {
			te, sized = estimate.Expected(), true
		}
		if !d.cfg.implRoleExplicit && found {
			all, _ := store.LoadRoles(d.w)
			var candidates []team.Role
			for _, candidate := range all {
				if strings.EqualFold(candidate.Kind, role.Kind) && d.roleAllowedByHarness(candidate) {
					candidates = append(candidates, candidate)
				}
			}
			if selected, ok := team.CheapestCapableForTitled(candidates, role.Kind, te, store.ClaimHints(d.w.Root, task), task.Title, orchestrationTaskBody(task)); ok {
				role = selected
			}
		}
		capacity := team.TaskCapacity(role, te, sized)
		claims := store.ClaimHints(d.w.Root, task)
		phase := cyclePreflightPhase{Phase: "implementation", Task: task.ID, Role: role.Name, Runtime: role.Runtime, Model: role.ModelID(), Grant: "rw", Claims: claims, Capacity: &capacity, Verdict: "pass", Evidence: "capacity and claims resolved before worker launch"}
		if !capacity.Fits && d.cfg.implRoleExplicit {
			phase.Verdict = "refuse"
			phase.Evidence = capacity.Reason
			phase.Remediation = "assign a role whose capacity covers the task, estimate or decompose the task"
			result.Phases = append(result.Phases, phase)
			result.Verdict = "refuse"
			d.emitCyclePreflight(result)
			return clikit.Refusedf("cycle preflight: task %03d-%s cannot use explicit role %s: %s; no worker was started", task.Seq, task.Slug, role.Name, capacity.Reason)
		}
		if !capacity.Fits {
			phase.Verdict = "unobserved"
			phase.Evidence = capacity.Reason + "; automatic sizing/routing remains a later phase"
		}
		result.Phases = append(result.Phases, phase)
		implementationRoles[role.Name] = true
	}

	// Runtime probes are subcommands, never coding-agent launches. Reviewer is
	// checked first so an impossible review seat cannot spend implementation
	// tokens and then fail at the end of every cycle.
	roles := append([]string{d.cfg.reviewRole}, sortedKeys(implementationRoles)...)
	grants := append([]string{"ro"}, repeat("rw", len(roles)-1)...)
	for i, roleName := range roles {
		role, found := store.LoadRole(d.w, roleName)
		phaseName := "implementation-runtime"
		if i == 0 {
			phaseName = "reviewer-runtime"
		}
		phase := cyclePreflightPhase{Phase: phaseName, Role: roleName, Grant: grants[i], Verdict: "pass"}
		if found {
			phase.Runtime, phase.Model = role.Runtime, role.ModelID()
			if !d.roleAllowedByHarness(role) {
				phase.Verdict = "refuse"
				phase.Evidence = "role runtime is outside the pinned harness policy"
				phase.Remediation = "choose a role on the selected harness or explicitly configure hybrid mode"
				result.Phases = append(result.Phases, phase)
				result.Verdict = "refuse"
				d.emitCyclePreflight(result)
				return clikit.Refusedf("cycle preflight: %s role %s is outside harness policy %s:%s; no worker was started", phaseName, roleName, d.cfg.harnessMode, strings.Join(d.cfg.allowedHarnesses, ","))
			}
		}
		if pr, ok := d.run.(cyclePreflightRunner); ok {
			out, err := pr.runPreflight(phaseName, "preflight", "--role", roleName, "--grant", grants[i])
			phase.Evidence = strings.TrimSpace(out)
			if err != nil {
				phase.Verdict = "refuse"
				phase.Remediation = "fix the declared runtime contract or select a compatible role; policy refusals are not retried"
				result.Phases = append(result.Phases, phase)
				result.Verdict = "refuse"
				d.emitCyclePreflight(result)
				return fmt.Errorf("cycle preflight %s for role %s: %w", phaseName, roleName, err)
			}
		} else {
			phase.Verdict = "unobserved"
			phase.Evidence = "runner does not expose preflight (test or embedding boundary)"
		}
		result.Phases = append(result.Phases, phase)
	}
	d.emitCyclePreflight(result)
	return nil
}

func sortedKeys(values map[string]bool) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	slices.Sort(out)
	return out
}

func repeat(value string, n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = value
	}
	return out
}
