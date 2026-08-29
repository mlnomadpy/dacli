package workflows_test

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

var requiredGates = []string{
	"test-matrix",
	"lint",
	"clean-checkout",
	"release-snapshot",
	"cross-compile",
}

func TestRequiredTestCheckGatesEveryCIJob(t *testing.T) {
	workflow := readWorkflow(t)
	testJob := jobBlock(t, workflow, "test")

	needs := needsList(t, testJob)
	sort.Strings(needs)
	wantNeeds := append([]string(nil), requiredGates...)
	sort.Strings(wantNeeds)
	if strings.Join(needs, ",") != strings.Join(wantNeeds, ",") {
		t.Fatalf("test.needs = %v, want %v", needs, wantNeeds)
	}

	for _, gate := range requiredGates {
		assertion := `test "${{ needs.` + gate + `.result }}" = "success"`
		if !strings.Contains(testJob, assertion) {
			t.Errorf("test job does not require %s to equal success", gate)
		}
	}

	for _, gate := range requiredGates {
		for _, result := range []string{"failure", "cancelled"} {
			results := successfulResults()
			results[gate] = result
			if requiredCheckSucceeds(testJob, results) {
				t.Errorf("required test check succeeds when %s is %s", gate, result)
			}
		}
	}
}

func TestLintUsesPatchedGo125Toolchain(t *testing.T) {
	workflow := readWorkflow(t)
	lintJob := jobBlock(t, workflow, "lint")

	if !regexp.MustCompile(`(?m)^          go-version: "1\.25\.(1[3-9]|[2-9][0-9]+)"$`).MatchString(lintJob) {
		t.Fatal("lint job must use Go 1.25.13 or newer within the 1.25 line")
	}
}

func TestRoutineCIIsLinuxOnlyAndPRTriggered(t *testing.T) {
	workflow := readWorkflow(t)
	testJob := jobBlock(t, workflow, "test-matrix")

	if regexp.MustCompile(`(?m)^\s*runs-on:\s*macos`).MatchString(workflow) {
		t.Fatal("routine ci must not run a native macOS job")
	}
	if strings.Contains(testJob, "strategy:") || strings.Contains(testJob, "matrix.") {
		t.Fatal("single-platform routine ci must not retain a one-entry matrix")
	}
	if !strings.Contains(testJob, "runs-on: ubuntu-latest") {
		t.Fatal("routine test job must run on ubuntu-latest")
	}
	if regexp.MustCompile(`(?m)^  push:`).MatchString(workflow) {
		t.Fatal("routine ci must not duplicate pull-request verification on pushes")
	}
	if !strings.Contains(workflow, "  pull_request:\n") {
		t.Fatal("routine ci must run for pull requests")
	}
	if !strings.Contains(workflow, "  workflow_dispatch:\n") {
		t.Fatal("routine ci must retain workflow_dispatch recovery")
	}
	if !strings.Contains(workflow, "group: ci-${{ github.event.pull_request.number || github.ref }}") ||
		!strings.Contains(workflow, "cancel-in-progress: true") {
		t.Fatal("routine ci must cancel a superseded run for the same pull request or ref")
	}
}

func TestFuzzCampaignsRunOnlyInQualityWorkflow(t *testing.T) {
	routine := readWorkflow(t)
	quality := readNamedWorkflow(t, "quality.yml")

	for _, target := range fuzzTargets {
		if strings.Contains(routine, target) {
			t.Errorf("routine ci must not run fuzz target %q", target)
		}
		if !strings.Contains(quality, target) {
			t.Errorf("quality workflow is missing fuzz target %q", target)
		}
	}
	if !strings.Contains(quality, "  schedule:\n") || !strings.Contains(quality, "  workflow_dispatch:\n") {
		t.Fatal("quality workflow must support scheduled and manual fuzz campaigns")
	}
}

func TestCrossCompileUsesOneJobForAllReleaseTargets(t *testing.T) {
	workflow := readWorkflow(t)
	crossCompile := jobBlock(t, workflow, "cross-compile")

	if strings.Contains(crossCompile, "matrix:") || strings.Contains(crossCompile, "matrix.") {
		t.Fatal("cross-compile must not use a matrix")
	}
	for _, target := range crossCompileTargets {
		if !strings.Contains(crossCompile, target) {
			t.Errorf("cross-compile is missing target %q", target)
		}
	}
}

var fuzzTargets = []string{
	"FuzzParseFlags",
	"FuzzValueSpellingsAgree",
	"FuzzFrontMatterRoundTrip",
	"FuzzParseNeverPanics",
	"FuzzSafeSegmentNeverEscapes",
	"FuzzSafeRelPathNeverEscapes",
}

var crossCompileTargets = []string{
	"windows/amd64",
	"windows/arm64",
	"darwin/amd64",
	"darwin/arm64",
	"linux/amd64",
	"linux/arm64",
}

func TestReleaseRetainsNarrowNativeMacOSValidation(t *testing.T) {
	workflow := readNamedWorkflow(t, "release.yml")
	macOSJob := jobBlock(t, workflow, "macos-native")

	if !strings.Contains(macOSJob, "runs-on: macos-latest") {
		t.Fatal("release workflow must retain a native macOS validation job")
	}
	if !strings.Contains(macOSJob, "go test ./internal/procmon/ ./internal/features/execution/") {
		t.Fatal("native macOS validation must exercise process-sensitive packages")
	}
	if !strings.Contains(jobBlock(t, workflow, "goreleaser"), "needs: macos-native") {
		t.Fatal("release publication must wait for native macOS validation")
	}
}

func TestReleaseInstallsPinnedSyftBeforeGoReleaser(t *testing.T) {
	workflow := readNamedWorkflow(t, "release.yml")

	syft := regexp.MustCompile(`(?m)^      - uses: anchore/sbom-action/download-syft@v0\n        with:\n(?:          #[^\n]*\n)+          syft-version: "v[0-9]+\.[0-9]+\.[0-9]+"$`).FindStringIndex(workflow)
	if syft == nil {
		t.Fatal("release workflow must install a pinned Syft distribution")
	}
	goreleaser := strings.Index(workflow, "      - uses: goreleaser/goreleaser-action@v7\n")
	if goreleaser < 0 {
		t.Fatal("release workflow must run GoReleaser")
	}
	if syft[0] > goreleaser {
		t.Fatal("release workflow must install Syft before GoReleaser")
	}
}

func readWorkflow(t *testing.T) string {
	t.Helper()
	return readNamedWorkflow(t, "ci.yml")
}

func readNamedWorkflow(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("..", "..", ".github", "workflows", name)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

func jobBlock(t *testing.T, workflow, name string) string {
	t.Helper()
	start := strings.Index(workflow, "\n  "+name+":\n")
	if start < 0 {
		t.Fatalf("job %q not found", name)
	}
	block := workflow[start+1:]
	if end := regexp.MustCompile(`(?m)^  [a-zA-Z0-9_-]+:\s*$`).FindStringIndex(block[len("  "+name+":\n"):]); end != nil {
		block = block[:len("  "+name+":\n")+end[0]]
	}
	return block
}

func needsList(t *testing.T, job string) []string {
	t.Helper()
	re := regexp.MustCompile(`(?m)^    needs:\s*\[([^]]*)\]\s*$`)
	match := re.FindStringSubmatch(job)
	if match == nil {
		t.Fatal("test.needs must be an explicit list")
	}
	parts := strings.Split(match[1], ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func successfulResults() map[string]string {
	results := make(map[string]string, len(requiredGates))
	for _, gate := range requiredGates {
		results[gate] = "success"
	}
	return results
}

func requiredCheckSucceeds(job string, results map[string]string) bool {
	for gate, result := range results {
		assertion := `test "${{ needs.` + gate + `.result }}" = "success"`
		if strings.Contains(job, assertion) && result != "success" {
			return false
		}
	}
	return true
}
