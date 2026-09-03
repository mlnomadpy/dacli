package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/clikit"
)

func TestUnknownCommandGuidanceIsRegistryDerivedAndSafe(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		suggestions []string
		next        []string
		family      string
	}{
		{name: "missing task verb", args: []string{"task", "54"}, suggestions: []string{"dacli task show 54"}, next: []string{"review the suggested read-only command; it was not executed"}, family: "task"},
		{name: "ambiguous bare show", args: []string{"show", "54"}, suggestions: []string{"dacli project show 54", "dacli task show 54"}, next: []string{"choose the object family explicitly; no suggestion was executed"}},
		{name: "misspelled family", args: []string{"taks", "list"}, suggestions: []string{"dacli task list"}, next: []string{"review the suggested command; it was not executed"}},
		{name: "misspelled leaf", args: []string{"task", "cliam", "54"}, suggestions: []string{"dacli task claim 54"}, next: []string{"review the suggested command; it was not executed"}, family: "task"},
		{name: "invalid reference remains a suggestion", args: []string{"task", "not-a-real-ref"}, suggestions: []string{"dacli task show not-a-real-ref"}, next: []string{"review the suggested read-only command; it was not executed"}, family: "task"},
		{name: "invalid leaf", args: []string{"events", "status"}, next: []string{"dacli events --help"}, family: "events"},
		{name: "far typo", args: []string{"zzzzzz"}, next: []string{"dacli help", "dacli help --all"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := commandGuidanceFor(tt.args)
			if !reflect.DeepEqual(got.suggestions, tt.suggestions) || !reflect.DeepEqual(got.nextActions, tt.next) || got.family != tt.family {
				t.Fatalf("guidance = %#v", got)
			}
		})
	}

	for _, suggestion := range commandGuidanceFor([]string{"show", "54"}).suggestions {
		path := strings.TrimPrefix(suggestion, "dacli ")
		cmd, _ := match(strings.Fields(path))
		if cmd == nil {
			t.Fatalf("suggestion %q is absent from command registry", suggestion)
		}
		if cmd.Mutates && !mutationInvocationIsReadOnly(cmd, []string{"54"}) {
			t.Fatalf("ambiguous read-only intent suggested mutation %q", suggestion)
		}
	}
}

func TestUnknownCommandJSONIncludesStableGuidanceWithoutExecuting(t *testing.T) {
	err := unknownCommandError([]string{"task", "54"})
	if exitCode(err) != 2 {
		t.Fatalf("exit = %d, want 2", exitCode(err))
	}
	var stderr bytes.Buffer
	emitError(&Ctx{Stderr: &stderr, JSON: true}, err)
	var got clikit.ErrorDetails
	if decodeErr := json.Unmarshal(stderr.Bytes(), &got); decodeErr != nil {
		t.Fatalf("JSON error: %v\n%s", decodeErr, stderr.String())
	}
	if !reflect.DeepEqual(got.Suggestions, []string{"dacli task show 54"}) || len(got.NextActions) != 1 {
		t.Fatalf("details = %#v", got)
	}
}

func TestKnownFamilyFailurePrintsFamilyBeforeTopLevelHelp(t *testing.T) {
	readEnd, writeEnd, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	original := os.Stderr
	os.Stderr = writeEnd
	t.Cleanup(func() { os.Stderr = original })
	if code := Main([]string{"events", "status"}); code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
	if err := writeEnd.Close(); err != nil {
		t.Fatal(err)
	}
	raw, err := io.ReadAll(readEnd)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	familyAt := strings.Index(text, "events tail")
	topAt := strings.Index(text, "Primary bounded workflow")
	if familyAt < 0 || topAt < 0 || familyAt > topAt {
		t.Fatalf("family help must precede top-level help:\n%s", text)
	}
}

func TestSuggestionsNeverExecuteMutatingCommands(t *testing.T) {
	called := false
	original := commands
	commands = []Command{{Path: "task claim", Mutates: true, Brief: "claim", Run: func(*Ctx, []string) error { called = true; return nil }}}
	t.Cleanup(func() { commands = original })

	_ = unknownCommandError([]string{"task", "cliam", "54"})
	if called {
		t.Fatal("constructing a suggestion executed its command")
	}
}
