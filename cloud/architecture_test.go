package cloud_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

func TestCloudBoundaryDoesNotImportLocalExecutionState(t *testing.T) {
	forbidden := []string{"/internal/store", "/internal/workspace", "/internal/features/execution"}
	err := filepath.WalkDir(".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imported := range file.Imports {
			for _, suffix := range forbidden {
				if strings.Contains(imported.Path.Value, suffix) {
					t.Errorf("%s imports forbidden local boundary %s", path, imported.Path.Value)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
