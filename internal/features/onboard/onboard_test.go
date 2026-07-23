package onboard

import (
	"os"
	"path/filepath"
	"testing"
)

// scanTodos must find only the real marker comment — a string literal that
// merely contains the word "TODO", and a []string{...} listing marker names,
// are not markers, and used to be matched as bare substrings.
func TestScanTodosIgnoresStringLiterals(t *testing.T) {
	dir := t.TempDir()
	src := "package fixture\n\n" +
		"// TODO: handle x\n\n" +
		"func f() {\n" +
		"	s := \"TODO in a string\"\n" +
		"	_ = s\n" +
		"	markers := []string{\"TODO\", \"FIXME\"}\n" +
		"	_ = markers\n" +
		"}\n"
	path := filepath.Join(dir, "fixture.go")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	var r scanResult
	scanTodos(path, "fixture.go", &r)

	if len(r.todos) != 1 {
		t.Fatalf("want exactly 1 marker, got %d: %+v", len(r.todos), r.todos)
	}
	td := r.todos[0]
	if td.marker != "TODO" || td.loc != "fixture.go:3" || td.text != "handle x" {
		t.Errorf("unexpected marker: %+v", td)
	}
}
