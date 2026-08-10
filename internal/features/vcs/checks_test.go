package vcs

import (
	"errors"
	"testing"
)

// A red or pending CI run must never be mistaken for an outage.
//
// runGH uses CombinedOutput, so prChecksPass sees gh's checks TABLE, and
// isNetworkErr is a bare substring scan. A check named `integration-timeout`
// — or any failing row whose text contains "timeout"/"unreachable"/"eof" —
// classified the run as GitHub-unreachable, and the caller answers an outage
// by LOCAL-MERGING the branch into trunk. Failing CI was a path to landing
// unverified code.
func TestARedCheckTableIsNotAnOutage(t *testing.T) {
	// A real gh checks table for a failing run, with a check whose NAME
	// contains a network word.
	table := "integration-timeout\tfail\t1m30s\thttps://github.com/o/r/actions/runs/1\n" +
		"unit\tpass\t20s\thttps://github.com/o/r/actions/runs/2\n"
	if !reachedGitHub(table) {
		t.Fatal("a checks table proves gh reached GitHub — no network failure can produce one")
	}

	// Pending is the same: the request completed.
	if !reachedGitHub("build\tpending\t0s\thttps://x/1\n") {
		t.Error("a pending row still proves the request completed")
	}
	if !reachedGitHub("no checks reported on the 'main' branch") {
		t.Error("gh's no-checks answer is a reply, not an outage")
	}
}

// A genuine transport failure still reads as one — the guard must not swing
// so far that a real outage stops falling back to a local merge.
func TestARealTransportFailureStillReadsAsNetwork(t *testing.T) {
	for _, out := range []string{
		"dial tcp 140.82.121.5:443: i/o timeout",
		"could not resolve host: api.github.com",
		"error connecting to api.github.com: connection refused",
	} {
		if reachedGitHub(out) {
			t.Errorf("%q is a transport failure, not a checks report", out)
		}
		if !isNetworkErr(out) {
			t.Errorf("%q must still classify as a network error", out)
		}
	}
}

// Driving prChecksPass itself, not just its helper: a red run must be reported
// as a GATE RESULT (pass=false, netErr=false), because only netErr may fall
// through to a local merge.
func TestPrChecksPassReportsARedRunAsAGateNotAnOutage(t *testing.T) {
	orig := runGH
	t.Cleanup(func() { runGH = orig })
	runGH = func(_ string, _ ...string) (string, error) {
		// gh exits non-zero when a required check fails, printing the table.
		// The failing check's NAME carries a network word.
		return "integration-timeout\tfail\t1m30s\thttps://x/1\nunit\tpass\t20s\thttps://x/2\n",
			errors.New("exit status 1")
	}

	pass, absent, _, netErr := prChecksPass("/tmp", "dacli/001-x")
	if pass {
		t.Error("a failing check must not report pass")
	}
	if absent {
		t.Error("checks were reported, so they are not absent")
	}
	if netErr {
		t.Fatal("a red checks table is a GATE result, not an outage — netErr here is what local-merges unverified code to trunk")
	}
}

// And the outage path still works, or a real GitHub failure would strand every
// wave instead of landing locally.
func TestPrChecksPassStillDetectsARealOutage(t *testing.T) {
	orig := runGH
	t.Cleanup(func() { runGH = orig })
	runGH = func(_ string, _ ...string) (string, error) {
		return "dial tcp 140.82.121.5:443: i/o timeout", errors.New("exit status 1")
	}
	if _, _, _, netErr := prChecksPass("/tmp", "dacli/001-x"); !netErr {
		t.Error("a genuine transport failure must still classify as a network error")
	}
}

// The merge fallback carries the same hazard and the same consequence: gh
// REFUSING to merge (red checks, a conflict, branch protection) is an answer,
// and only an outage may fall through to a local merge.
func TestARefusedMergeIsNotAnOutage(t *testing.T) {
	for _, out := range []string{
		"Pull request #12 is not mergeable: the base branch policy prohibits the merge",
		"X Required status check \"e2e-timeout\" is failing",
		"merge conflict between base and head",
	} {
		if !reachedGitHub(out) {
			t.Errorf("gh answered about the pull request, so it reached GitHub: %q", out)
		}
	}
}
