package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/commandresult"
)

const GitHubRequiredCheckPolicySchema = "github-required-check-policy/v1"

type RequiredCheckSource struct {
	Kind              string `json:"kind"`
	RulesetID         int64  `json:"ruleset_id,omitempty"`
	RulesetSourceType string `json:"ruleset_source_type,omitempty"`
	RulesetSource     string `json:"ruleset_source,omitempty"`
}

type RequiredCheckRequirement struct {
	Name    string                `json:"name"`
	Sources []RequiredCheckSource `json:"sources"`
}

// GitHubRequiredCheckPolicy is the exact effective check policy observed for
// one target branch. Requirements retain every source so a configured check
// cannot conceal a stricter repository or organization rule (issue #1004).
type GitHubRequiredCheckPolicy struct {
	Schema       string                     `json:"schema"`
	Repository   string                     `json:"repository"`
	Branch       string                     `json:"branch"`
	ObservedAt   time.Time                  `json:"observed_at"`
	State        string                     `json:"state"` // observed or unobservable
	Requirements []RequiredCheckRequirement `json:"requirements"`
	Error        string                     `json:"error,omitempty"`
	Override     bool                       `json:"override,omitempty"`
}

func (p GitHubRequiredCheckPolicy) Names() []string {
	out := make([]string, 0, len(p.Requirements))
	for _, requirement := range p.Requirements {
		out = append(out, requirement.Name)
	}
	return out
}

var ObserveGitHubRequiredCheckPolicy = observeGitHubRequiredCheckPolicy

func observeGitHubRequiredCheckPolicy(root, repo, branch string, configured []string, now time.Time) (GitHubRequiredCheckPolicy, error) {
	policy := newRequiredCheckPolicy(repo, branch, configured, now)
	if strings.TrimSpace(repo) == "" || strings.TrimSpace(branch) == "" {
		policy.State, policy.Error = "unobservable", "GitHub check policy requires repository and target branch"
		return policy, errors.New(policy.Error)
	}
	run := func(operation string, args ...string) ([]byte, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		base := []string{"api", "-H", "X-GitHub-Api-Version: 2022-11-28"}
		cmd := exec.CommandContext(ctx, "gh", append(base, args...)...)
		cmd.Dir = root
		return commandresult.Output(cmd, commandresult.RunOptions{Operation: operation, WorkspaceRoot: root, TimedOut: func() bool { return ctx.Err() == context.DeadlineExceeded }})
	}
	branchPath := url.PathEscape(branch)
	legacyRaw, legacyErr := run("observe legacy GitHub required checks", "repos/"+repo+"/branches/"+branchPath+"/protection/required_status_checks")
	if legacyErr != nil && !githubBranchNotProtected(legacyErr) {
		policy.State, policy.Error = "unobservable", legacyErr.Error()
		return policy, fmt.Errorf("observe legacy required checks for %s/%s: %w", repo, branch, legacyErr)
	}
	rulesRaw, err := run("observe applicable GitHub ruleset checks", "--paginate", "--slurp", "repos/"+repo+"/rules/branches/"+branchPath+"?per_page=100")
	if err != nil {
		policy.State, policy.Error = "unobservable", err.Error()
		return policy, fmt.Errorf("observe applicable rulesets for %s/%s: %w", repo, branch, err)
	}
	if err := mergeGitHubRequiredChecks(&policy, legacyRaw, rulesRaw); err != nil {
		policy.State, policy.Error = "unobservable", err.Error()
		return policy, err
	}
	policy.State = "observed"
	return policy, nil
}

func newRequiredCheckPolicy(repo, branch string, configured []string, now time.Time) GitHubRequiredCheckPolicy {
	p := GitHubRequiredCheckPolicy{Schema: GitHubRequiredCheckPolicySchema, Repository: repo, Branch: branch, ObservedAt: now.UTC()}
	for _, name := range configured {
		addRequiredCheck(&p, name, RequiredCheckSource{Kind: "configured"})
	}
	return p
}

func mergeGitHubRequiredChecks(policy *GitHubRequiredCheckPolicy, legacyRaw, rulesRaw []byte) error {
	if len(strings.TrimSpace(string(legacyRaw))) > 0 {
		var legacy struct {
			Contexts []string `json:"contexts"`
			Checks   []struct {
				Context string `json:"context"`
			} `json:"checks"`
		}
		if err := json.Unmarshal(legacyRaw, &legacy); err != nil {
			return fmt.Errorf("decode legacy branch protection required checks: %w", err)
		}
		for _, name := range legacy.Contexts {
			addRequiredCheck(policy, name, RequiredCheckSource{Kind: "legacy_branch_protection"})
		}
		for _, check := range legacy.Checks {
			addRequiredCheck(policy, check.Context, RequiredCheckSource{Kind: "legacy_branch_protection"})
		}
	}
	type branchRule struct {
		Type       string `json:"type"`
		Parameters struct {
			RequiredStatusChecks []struct {
				Context string `json:"context"`
			} `json:"required_status_checks"`
		} `json:"parameters"`
		RulesetID         int64  `json:"ruleset_id"`
		RulesetSourceType string `json:"ruleset_source_type"`
		RulesetSource     string `json:"ruleset_source"`
	}
	var pages [][]branchRule
	if err := json.Unmarshal(rulesRaw, &pages); err != nil {
		var rules []branchRule
		if directErr := json.Unmarshal(rulesRaw, &rules); directErr != nil {
			return fmt.Errorf("decode applicable GitHub ruleset checks: %w", err)
		}
		pages = [][]branchRule{rules}
	}
	for _, rules := range pages {
		for _, rule := range rules {
			if rule.Type != "required_status_checks" {
				continue
			}
			source := RequiredCheckSource{Kind: "ruleset", RulesetID: rule.RulesetID, RulesetSourceType: rule.RulesetSourceType, RulesetSource: rule.RulesetSource}
			for _, check := range rule.Parameters.RequiredStatusChecks {
				addRequiredCheck(policy, check.Context, source)
			}
		}
	}
	sort.Slice(policy.Requirements, func(i, j int) bool { return policy.Requirements[i].Name < policy.Requirements[j].Name })
	for i := range policy.Requirements {
		sort.Slice(policy.Requirements[i].Sources, func(a, b int) bool {
			x, y := policy.Requirements[i].Sources[a], policy.Requirements[i].Sources[b]
			return fmt.Sprintf("%s/%020d/%s/%s", x.Kind, x.RulesetID, x.RulesetSourceType, x.RulesetSource) < fmt.Sprintf("%s/%020d/%s/%s", y.Kind, y.RulesetID, y.RulesetSourceType, y.RulesetSource)
		})
	}
	return nil
}

func addRequiredCheck(policy *GitHubRequiredCheckPolicy, name string, source RequiredCheckSource) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	for i := range policy.Requirements {
		if policy.Requirements[i].Name != name {
			continue
		}
		for _, existing := range policy.Requirements[i].Sources {
			if existing == source {
				return
			}
		}
		policy.Requirements[i].Sources = append(policy.Requirements[i].Sources, source)
		return
	}
	policy.Requirements = append(policy.Requirements, RequiredCheckRequirement{Name: name, Sources: []RequiredCheckSource{source}})
}

func githubBranchNotProtected(err error) bool {
	diagnostic, ok := commandresult.AsDiagnostic(err)
	if !ok {
		return false
	}
	text := strings.ToLower(diagnostic.StdoutTail + "\n" + diagnostic.StderrTail)
	return strings.Contains(text, "branch not protected")
}
