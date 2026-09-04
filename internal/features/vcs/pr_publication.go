package vcs

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/gitx"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

var (
	observeRemoteBranch = func(root, branch string) (string, error) {
		out, err := gitx.RunNetwork(root, "ls-remote", "--heads", "origin", "refs/heads/"+branch)
		if err != nil {
			return "", err
		}
		fields := strings.Fields(out)
		if len(fields) == 0 {
			return "", nil
		}
		if len(fields) != 2 || fields[1] != "refs/heads/"+branch {
			return "", fmt.Errorf("origin returned an invalid exact-ref response for %s", branch)
		}
		return fields[0], nil
	}
	publishTaskBranch = gitx.PushSync
)

func localBranchOID(root, branch string) (string, error) {
	oid, err := gitx.Run(root, "rev-parse", "--verify", "refs/heads/"+branch+"^{commit}")
	if err != nil {
		return "", fmt.Errorf("canonical task branch %s does not exist; create and commit it with `dacli worktree add --task <ref>` before opening a PR: %w", branch, err)
	}
	return strings.TrimSpace(oid), nil
}

func requireTaskCommits(root, branch, base string) error {
	baseRef := "refs/heads/" + base
	if !gitCommitExists(root, baseRef) {
		baseRef = "refs/remotes/origin/" + base
		if !gitCommitExists(root, baseRef) {
			// A configured GitHub base need not exist in a shallow/local clone.
			// GitHub validates that base during PR creation; the no-change refusal
			// remains exact whenever either local representation is observable.
			return nil
		}
	}
	out, err := gitx.Run(root, "rev-list", "--count", baseRef+"..refs/heads/"+branch)
	if err != nil {
		return fmt.Errorf("compare canonical branch %s with landing base %s: %w", branch, base, err)
	}
	n, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil {
		return fmt.Errorf("git returned an invalid commit count for %s: %q", branch, out)
	}
	if n == 0 {
		return clikit.Refusedf("canonical branch %s has no commits beyond %s; commit the task result before opening a PR", branch, base)
	}
	return nil
}

func gitCommitExists(root, ref string) bool {
	_, err := gitx.Run(root, "rev-parse", "--verify", ref+"^{commit}")
	return err == nil
}

func previewPRPublication(w *workspace.Workspace, t *store.Task, base string) (store.PRPublication, error) {
	branch := BranchFor(t)
	if !gitx.BranchExists(w.Root, branch) {
		return store.PRPublication{Branch: branch}, nil
	}
	oid, err := localBranchOID(w.Root, branch)
	if err != nil {
		return store.PRPublication{}, err
	}
	if err := requireTaskCommits(w.Root, branch, base); err != nil {
		return store.PRPublication{}, err
	}
	return store.PRPublication{Branch: branch, LocalOID: oid}, nil
}

func observePRPublication(w *workspace.Workspace, t *store.Task, base string) (store.PRPublication, error) {
	branch := BranchFor(t)
	oid, err := localBranchOID(w.Root, branch)
	if err != nil {
		return store.PRPublication{}, err
	}
	if err := requireTaskCommits(w.Root, branch, base); err != nil {
		return store.PRPublication{}, err
	}
	return store.PRPublication{
		Schema: store.PRPublicationSchema, TaskID: t.ID, Branch: branch,
		Base: base, LocalOID: oid, Stage: "observed", UpdatedAt: time.Now().UTC(),
	}, nil
}

func publishCanonicalTaskBranch(w *workspace.Workspace, t *store.Task, base string) (store.PRPublication, error) {
	cp, err := observePRPublication(w, t, base)
	if err != nil {
		return cp, err
	}
	if err := store.SavePRPublication(w, cp); err != nil {
		return cp, fmt.Errorf("checkpoint canonical branch observation: %w", err)
	}
	remoteOID, err := observeRemoteBranch(w.Root, cp.Branch)
	if err != nil {
		return cp, fmt.Errorf("observe origin/%s before publication: %w", cp.Branch, err)
	}
	if remoteOID != "" && remoteOID != cp.LocalOID {
		return cp, clikit.Refusedf("origin/%s is at %s but the canonical task branch is at %s; resolve the divergent branch, then rerun `dacli push --task %s` before `dacli pr --task %s`", cp.Branch, remoteOID, cp.LocalOID, t.ID, t.ID)
	}
	if remoteOID == "" {
		if _, err := publishTaskBranch(w.Root, cp.Branch); err != nil {
			if errors.Is(err, gitx.ErrLeaseRequired) {
				return cp, clikit.Refusedf("origin/%s diverges from the canonical task branch; inspect and resolve the branch history, then rerun `dacli push --task %s` before `dacli pr --task %s`", cp.Branch, t.ID, t.ID)
			}
			return cp, fmt.Errorf("publish canonical branch %s: %w", cp.Branch, err)
		}
		remoteOID, err = observeRemoteBranch(w.Root, cp.Branch)
		if err != nil {
			return cp, fmt.Errorf("verify origin/%s after publication: %w", cp.Branch, err)
		}
	}
	if remoteOID != cp.LocalOID {
		return cp, clikit.Refusedf("origin/%s is at %s instead of the observed canonical commit %s; rerun `dacli push --task %s` after resolving the branch", cp.Branch, orMissing(remoteOID), cp.LocalOID, t.ID)
	}
	cp.RemoteOID = remoteOID
	cp.Stage = "pushed"
	cp.UpdatedAt = time.Now().UTC()
	if err := store.SavePRPublication(w, cp); err != nil {
		return cp, fmt.Errorf("checkpoint canonical branch publication: %w", err)
	}
	return cp, nil
}

func recordPRPublication(w *workspace.Workspace, cp store.PRPublication, url string) error {
	cp.PRURL = strings.TrimSpace(url)
	cp.Stage = "pr-recorded"
	cp.UpdatedAt = time.Now().UTC()
	if err := store.SavePRPublication(w, cp); err != nil {
		return fmt.Errorf("checkpoint PR publication: %w", err)
	}
	return nil
}

func orMissing(s string) string {
	if strings.TrimSpace(s) == "" {
		return "<missing>"
	}
	return strings.TrimSpace(s)
}
