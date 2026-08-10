package vcs

import "testing"

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
