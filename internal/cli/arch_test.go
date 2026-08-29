package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The feature-sliced design's load-bearing rule: SLICES NEVER IMPORT EACH
// OTHER. Shared behavior belongs in clikit or the entity/engine layers;
// a feature→feature import is coupling that will calcify. This test is the
// rule's enforcement — without it, the isolation is a comment.
func TestFeatureSlicesAreIsolated(t *testing.T) {
	featuresDir := filepath.Join("..", "features")
	slices, err := os.ReadDir(featuresDir)
	if err != nil {
		t.Fatalf("features dir: %v", err)
	}
	importRe := regexp.MustCompile(`"github\.com/mlnomadpy/dacli/internal/features/([a-z]+)`)

	for _, slice := range slices {
		if !slice.IsDir() {
			continue
		}
		files, _ := os.ReadDir(filepath.Join(featuresDir, slice.Name()))
		for _, file := range files {
			if !strings.HasSuffix(file.Name(), ".go") {
				continue
			}
			raw, err := os.ReadFile(filepath.Join(featuresDir, slice.Name(), file.Name()))
			if err != nil {
				t.Fatal(err)
			}
			for _, m := range importRe.FindAllStringSubmatch(string(raw), -1) {
				if m[1] != slice.Name() {
					t.Errorf("slice %s imports slice %s (%s) — shared behavior belongs in clikit or an entity package",
						slice.Name(), m[1], file.Name())
				}
			}
		}
	}
}

// The app layer owns aggregation, not behavior: cli must not reach past the
// kernel into entities directly (the executor and mcp serve are the two
// sanctioned exceptions, and they need only clikit + mcp).
func TestAppLayerStaysThin(t *testing.T) {
	raw, err := os.ReadFile("cli.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{
		"internal/store\"", "internal/eventlog\"", "internal/brief\"", "internal/spm\"",
	} {
		if strings.Contains(string(raw), forbidden) {
			t.Errorf("cli.go imports %s — feature logic is leaking back into the app layer", forbidden)
		}
	}
}

// The three oldest feature slices accumulated unrelated lifecycle concerns in
// single coordinator files. Keep the extracted boundaries load-bearing: a
// future change must extend the focused component instead of silently growing
// the old collision hotspot back past the reviewed ceiling (issue #901).
func TestLargeFeatureCoordinatorsStayDecomposed(t *testing.T) {
	featuresDir := filepath.Join("..", "features")
	components := map[string][]string{
		"execution":     {"runtime_adapters.go", "provider_runtime.go", "observability.go", "lifecycle.go", "behavioral_preflight.go"},
		"orchestration": {"preflight.go", "scheduling.go", "delivery_tail.go", "recovery.go", "phase_journal.go"},
		"ghmirror":      {"transport.go", "adoption.go", "publication.go", "project.go"},
	}
	coordinators := map[string]int{
		"execution/execution.go":         2250,
		"orchestration/orchestration.go": 2500,
		"ghmirror/ghmirror.go":           1100,
	}
	for rel, ceiling := range coordinators {
		raw, err := os.ReadFile(filepath.Join(featuresDir, rel))
		if err != nil {
			t.Fatal(err)
		}
		if lines := strings.Count(string(raw), "\n") + 1; lines > ceiling {
			t.Errorf("%s grew to %d lines (ceiling %d); put the responsibility in its focused component", rel, lines, ceiling)
		}
	}
	for slice, files := range components {
		for _, file := range files {
			if _, err := os.Stat(filepath.Join(featuresDir, slice, file)); err != nil {
				t.Errorf("%s component %s is missing: %v", slice, file, err)
			}
		}
	}
}
