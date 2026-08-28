// Package verifyroute resolves declarative verification rules against a
// concrete change surface. It deliberately accepts argv, never a shell command
// string: repository paths and arguments remain data instead of executable
// syntax (issue #860).
package verifyroute

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

type EnvironmentPolicy struct {
	Inherit []string          `json:"inherit,omitempty"`
	Set     map[string]string `json:"set,omitempty"`
}

type Rule struct {
	ID             string            `json:"id"`
	Include        []string          `json:"include"`
	Exclude        []string          `json:"exclude,omitempty"`
	Cwd            string            `json:"cwd"`
	Argv           []string          `json:"argv"`
	Environment    EnvironmentPolicy `json:"environment,omitempty"`
	Gate           string            `json:"gate"`
	Fanout         bool              `json:"fanout"`
	Required       bool              `json:"required,omitempty"`
	ContractGroups []string          `json:"contract_groups,omitempty"`
}

type ContractGroup struct {
	ID      string   `json:"id"`
	Include []string `json:"include"`
	Exclude []string `json:"exclude,omitempty"`
	Rules   []string `json:"rules"`
}

type Command struct {
	RuleID      string            `json:"rule_id"`
	Cwd         string            `json:"cwd"`
	Argv        []string          `json:"argv"`
	Environment EnvironmentPolicy `json:"environment,omitempty"`
	Gate        string            `json:"gate"`
	Fanout      bool              `json:"fanout"`
}

type Result struct {
	RuleID string   `json:"rule_id"`
	Cwd    string   `json:"cwd"`
	Argv   []string `json:"argv"`
	Output []byte   `json:"-"`
}

// Validate rejects ambiguous policy before any verifier is started.
func Validate(root string, rules []Rule, groups []ContractGroup) error {
	ids := make(map[string]Rule, len(rules))
	for i, rule := range rules {
		where := fmt.Sprintf("verification rule %d", i+1)
		if strings.TrimSpace(rule.ID) == "" {
			return fmt.Errorf("%s has an empty id", where)
		}
		if previous, ok := ids[rule.ID]; ok {
			return fmt.Errorf("verification rule id %q conflicts: cwd/argv %q %q and %q %q", rule.ID, previous.Cwd, previous.Argv, rule.Cwd, rule.Argv)
		}
		ids[rule.ID] = rule
		if len(rule.Include) == 0 {
			return fmt.Errorf("verification rule %q has no path include matcher", rule.ID)
		}
		if len(rule.Argv) == 0 || strings.TrimSpace(rule.Argv[0]) == "" {
			return fmt.Errorf("verification rule %q has empty argv; configure an executable and arguments as separate JSON values", rule.ID)
		}
		if strings.TrimSpace(rule.Gate) == "" {
			return fmt.Errorf("verification rule %q has no gate kind", rule.ID)
		}
		if root != "" {
			if err := validateCwd(root, rule.Cwd); err != nil {
				return fmt.Errorf("verification rule %q: %w", rule.ID, err)
			}
		}
		for _, name := range append(slices.Clone(rule.Environment.Inherit), mapKeys(rule.Environment.Set)...) {
			if !validEnvName(name) {
				return fmt.Errorf("verification rule %q has invalid environment name %q", rule.ID, name)
			}
		}
	}
	groupIDs := map[string]bool{}
	for _, group := range groups {
		if group.ID == "" || groupIDs[group.ID] {
			return fmt.Errorf("verification contract group id %q is empty or duplicated", group.ID)
		}
		groupIDs[group.ID] = true
		if len(group.Include) == 0 {
			return fmt.Errorf("verification contract group %q needs include matchers", group.ID)
		}
		for _, id := range group.Rules {
			if _, ok := ids[id]; !ok {
				return fmt.Errorf("verification contract group %q references unknown rule %q", group.ID, id)
			}
		}
	}
	for _, rule := range rules {
		for _, group := range rule.ContractGroups {
			if !groupIDs[group] {
				return fmt.Errorf("verification rule %q references unknown contract group %q", rule.ID, group)
			}
		}
	}
	for _, group := range groups {
		dependents := len(group.Rules)
		for _, rule := range rules {
			if slices.Contains(rule.ContractGroups, group.ID) {
				dependents++
			}
		}
		if dependents == 0 {
			return fmt.Errorf("verification contract group %q has no dependent rules", group.ID)
		}
	}
	return nil
}

// Resolve selects rules for changedPaths. Contract changes deliberately fan
// out to every declared dependent rule, while ordinary paths select only their
// matching subsystem. A required policy fails closed for unmatched non-docs.
func Resolve(root string, rules []Rule, groups []ContractGroup, changedPaths []string) ([]Command, error) {
	if err := Validate(root, rules, groups); err != nil {
		return nil, err
	}
	byID := make(map[string]Rule, len(rules))
	selected := map[string]bool{}
	matchedPath := map[string]bool{}
	for _, rule := range rules {
		byID[rule.ID] = rule
		for _, changed := range changedPaths {
			if matchesRule(rule.Include, rule.Exclude, changed) {
				selected[rule.ID], matchedPath[changed] = true, true
			}
		}
	}
	for _, group := range groups {
		triggered := false
		for _, changed := range changedPaths {
			if matchesRule(group.Include, group.Exclude, changed) {
				triggered, matchedPath[changed] = true, true
			}
		}
		if triggered {
			for _, id := range group.Rules {
				selected[id] = true
			}
			for _, rule := range rules {
				if slices.Contains(rule.ContractGroups, group.ID) {
					selected[rule.ID] = true
				}
			}
		}
	}
	requireCoverage := false
	for _, rule := range rules {
		requireCoverage = requireCoverage || rule.Required
	}
	if requireCoverage {
		var unmatched []string
		for _, changed := range changedPaths {
			if !matchedPath[changed] && !docsOnly(changed) {
				unmatched = append(unmatched, filepath.ToSlash(changed))
			}
		}
		if len(unmatched) > 0 {
			sort.Strings(unmatched)
			return nil, fmt.Errorf("required verification policy matches no rule for changed path(s) %s; add an include matcher or explicitly classify the paths as docs", strings.Join(unmatched, ", "))
		}
	}
	commands := make([]Command, 0, len(selected))
	for _, rule := range rules {
		if selected[rule.ID] {
			commands = append(commands, Command{RuleID: rule.ID, Cwd: rule.Cwd, Argv: slices.Clone(rule.Argv), Environment: rule.Environment, Gate: rule.Gate, Fanout: rule.Fanout})
		}
	}
	return commands, nil
}

// Execute runs an already-resolved plan directly. It never invokes a shell.
func Execute(ctx context.Context, root string, commands []Command) ([]Result, error) {
	results := make([]Result, 0, len(commands))
	for _, command := range commands {
		if len(command.Argv) == 0 {
			return results, fmt.Errorf("verification rule %q has empty argv", command.RuleID)
		}
		cwd, err := resolveCwd(root, command.Cwd)
		if err != nil {
			return results, fmt.Errorf("verification rule %q: %w", command.RuleID, err)
		}
		cmd := exec.CommandContext(ctx, command.Argv[0], command.Argv[1:]...)
		cmd.Dir = cwd
		cmd.Env = environment(command.Environment)
		out, err := cmd.CombinedOutput()
		results = append(results, Result{RuleID: command.RuleID, Cwd: command.Cwd, Argv: slices.Clone(command.Argv), Output: out})
		if err != nil {
			return results, fmt.Errorf("verification rule %q (%s in %s) failed: %w", command.RuleID, strings.Join(command.Argv, " "), command.Cwd, err)
		}
	}
	return results, nil
}

func matchesRule(include, exclude []string, changed string) bool {
	changed = strings.TrimPrefix(filepath.ToSlash(filepath.Clean(changed)), "./")
	matched := false
	for _, pattern := range include {
		if glob(pattern, changed) {
			matched = true
			break
		}
	}
	if !matched {
		return false
	}
	for _, pattern := range exclude {
		if glob(pattern, changed) {
			return false
		}
	}
	return true
}

func glob(pattern, name string) bool {
	pattern = strings.TrimPrefix(filepath.ToSlash(filepath.Clean(pattern)), "./")
	return matchSegments(strings.Split(pattern, "/"), strings.Split(name, "/"))
}

func matchSegments(pattern, name []string) bool {
	if len(pattern) == 0 {
		return len(name) == 0
	}
	if pattern[0] == "**" {
		return matchSegments(pattern[1:], name) || (len(name) > 0 && matchSegments(pattern, name[1:]))
	}
	if len(name) == 0 {
		return false
	}
	ok, err := path.Match(pattern[0], name[0])
	return err == nil && ok && matchSegments(pattern[1:], name[1:])
}

func docsOnly(name string) bool {
	clean := strings.ToLower(filepath.ToSlash(name))
	base := path.Base(clean)
	return strings.HasPrefix(clean, "docs/") || strings.HasPrefix(clean, ".github/") || base == "readme" || strings.HasPrefix(base, "readme.") || slices.Contains([]string{".md", ".mdx", ".rst", ".txt"}, path.Ext(base))
}

func validateCwd(root, cwd string) error {
	_, err := resolveCwd(root, cwd)
	return err
}

func resolveCwd(root, cwd string) (string, error) {
	if cwd == "" {
		cwd = "."
	}
	if filepath.IsAbs(cwd) {
		return "", fmt.Errorf("cwd %q must be repository-relative", cwd)
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	abs := filepath.Join(absRoot, filepath.FromSlash(cwd))
	rel, err := filepath.Rel(absRoot, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("cwd %q resolves outside repository root", cwd)
	}
	st, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("cwd %q is unavailable: %w", cwd, err)
	}
	if !st.IsDir() {
		return "", fmt.Errorf("cwd %q is not a directory", cwd)
	}
	return abs, nil
}

func environment(policy EnvironmentPolicy) []string {
	values := map[string]string{}
	for _, name := range policy.Inherit {
		if value, ok := os.LookupEnv(name); ok {
			values[name] = value
		}
	}
	for name, value := range policy.Set {
		values[name] = value
	}
	names := mapKeys(values)
	sort.Strings(names)
	out := make([]string, 0, len(names))
	for _, name := range names {
		out = append(out, name+"="+values[name])
	}
	return out
}

func validEnvName(name string) bool {
	if name == "" || strings.Contains(name, "=") {
		return false
	}
	for i, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_' || (i > 0 && r >= '0' && r <= '9') {
			continue
		}
		return false
	}
	return true
}

func mapKeys[V any](values map[string]V) []string {
	out := make([]string, 0, len(values))
	for key := range values {
		out = append(out, key)
	}
	return out
}
