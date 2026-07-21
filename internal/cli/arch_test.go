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
