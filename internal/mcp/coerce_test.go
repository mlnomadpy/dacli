package mcp

import (
	"encoding/json"
	"testing"
)

// The MCP argument coercers are the boundary where an agent's untyped JSON
// becomes a typed Go value, and they were the least-covered code in the
// package. A wrong coercion here does not fail — it yields a zero value, which
// is the silent-wrong-answer class this project treats as its most expensive
// (dacli 361).
func TestBoolCoercionAcceptsTheSpellingsClientsActuallySend(t *testing.T) {
	for _, tc := range []struct {
		name string
		val  any
		want bool
	}{
		{"real bool true", true, true},
		{"real bool false", false, false},
		// Several MCP clients stringify scalars. b() used to accept only a
		// real bool, so these silently read FALSE — and b() gates dry_run.
		{"string true", "true", true},
		{"string false", "false", false},
		{"string True", "True", true},
		{"string 1", "1", true},
		{"string 0", "0", false},
		{"padded", " true ", true},
		// Genuinely uninterpretable still yields false, which is the safe
		// direction for every flag b() currently gates except dry_run.
		{"nonsense", "yes please", false},
		{"number", float64(1), false},
		{"absent", nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := map[string]any{}
			if tc.val != nil {
				args["k"] = tc.val
			}
			if got := b(args, "k"); got != tc.want {
				t.Errorf("b(%#v) = %v, want %v", tc.val, got, tc.want)
			}
		})
	}
}

// dry_run is the one flag where a false-by-accident is UNSAFE: the caller asked
// for a rehearsal and got a real mutation. Named separately so the reason
// survives a refactor of the table above.
func TestDryRunSurvivesAStringifyingClient(t *testing.T) {
	var args map[string]any
	// Exactly what a client that stringifies scalars puts on the wire.
	if err := json.Unmarshal([]byte(`{"dry_run":"true"}`), &args); err != nil {
		t.Fatal(err)
	}
	if !b(args, "dry_run") {
		t.Fatal("dry_run sent as a string read as FALSE — the caller asked for a rehearsal and would get a real mutation")
	}
}

func TestIntCoercionAcceptsEveryNumericShape(t *testing.T) {
	for _, tc := range []struct {
		name string
		val  any
		want int
	}{
		{"json number", float64(3), 3}, // what JSON actually yields
		{"go int", 3, 3},               // what a Go caller hands over
		{"go int64", int64(3), 3},      //
		{"numeric string", "3", 3},     // a stringifying client
		{"negative", float64(-2), -2},  //
		{"truncates", float64(3.9), 3}, // documented, not accidental
		{"unparseable", "three", 0},    //
		{"absent", nil, 0},             //
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := map[string]any{}
			if tc.val != nil {
				args["k"] = tc.val
			}
			if got := i(args, "k"); got != tc.want {
				t.Errorf("i(%#v) = %d, want %d", tc.val, got, tc.want)
			}
		})
	}
}

// list drops non-strings rather than failing, so a mixed array silently
// shortens. Pinned so the behaviour is a decision rather than an accident.
func TestListKeepsOnlyStringsAndSaysSoByShape(t *testing.T) {
	args := map[string]any{"k": []any{"a", float64(2), "b", nil, true, "c"}}
	got := list(args, "k")
	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Errorf("list = %#v; want the three strings, non-strings dropped", got)
	}
	if l := list(map[string]any{}, "k"); len(l) != 0 {
		t.Errorf("an absent key must yield an empty list, got %#v", l)
	}
	if l := list(map[string]any{"k": "not an array"}, "k"); len(l) != 0 {
		t.Errorf("a non-array must yield an empty list, got %#v", l)
	}
}

// s and need are the required-argument gate: a missing or wrong-typed string
// must be REFUSED by name, not defaulted to "".
func TestNeedRefusesAMissingOrWrongTypedArgument(t *testing.T) {
	ok := map[string]any{"task": "001", "project": "p"}
	if err := need(ok, "task", "project"); err != nil {
		t.Errorf("a complete argument set must pass: %v", err)
	}
	for _, args := range []map[string]any{
		{"task": "001"},                 // missing
		{"task": "001", "project": ""},  // empty
		{"task": "001", "project": 42},  // wrong type reads as ""
		{"task": "001", "project": nil}, //
	} {
		err := need(args, "task", "project")
		if err == nil {
			t.Errorf("need(%#v) accepted an unusable argument set", args)
			continue
		}
		if !contains(err.Error(), "project") {
			t.Errorf("the refusal must name the missing argument: %v", err)
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}

// The refusal the coercers could not give. `dry_run` is the case that matters:
// a value the coercer cannot read yielded FALSE, so a caller asking for a
// rehearsal got a real mutation (dacli 361).
func TestANonsenseDryRunIsRefusedRatherThanReadAsFalse(t *testing.T) {
	tl, ok := toolByName("dacli_github_push")
	if !ok {
		// Fall back to any tool declaring a boolean, so this test cannot rot
		// into passing because a tool was renamed.
		for _, cand := range tools {
			props, _ := cand.schema["properties"].(map[string]any)
			for k, v := range props {
				if m, _ := v.(map[string]any); m["type"] == "boolean" {
					tl, ok = cand, true
					t.Logf("using tool %q boolean %q", cand.name, k)
					break
				}
			}
			if ok {
				break
			}
		}
	}
	if !ok {
		t.Fatal("no tool declares a boolean argument — this test would measure nothing")
	}

	boolKey := ""
	props, _ := tl.schema["properties"].(map[string]any)
	for k, v := range props {
		if m, _ := v.(map[string]any); m["type"] == "boolean" {
			boolKey = k
			break
		}
	}
	if boolKey == "" {
		t.Fatal("the chosen tool declares no boolean argument")
	}

	err := validateArgs(tl, map[string]any{boolKey: "yes please"})
	if err == nil {
		t.Fatalf("an uninterpretable %q was accepted and would coerce to false", boolKey)
	}
	if !contains(err.Error(), boolKey) {
		t.Errorf("the refusal must NAME the argument: %v", err)
	}

	// And the spellings a real client sends are still accepted, or the guard
	// has traded a silent wrong answer for a loud wrong refusal.
	for _, good := range []any{true, false, "true", "false", "1", "0"} {
		if err := validateArgs(tl, map[string]any{boolKey: good}); err != nil {
			t.Errorf("validateArgs rejected a legitimate %q value %#v: %v", boolKey, good, err)
		}
	}
}

// The refusal has to REACH the client, not just exist. call() is the dispatch
// point, and its result is what the MCP client sees.
func TestTheRefusalReachesTheClientAsAnError(t *testing.T) {
	var executed bool
	exec := func(argv []string, jsonMode bool) (string, string, int) {
		executed = true
		return "", "", 0
	}
	// check_task declares "n" as an integer. Fatal, not Skip, if it is gone:
	// a skipped test measures nothing, which is the failure this suite exists
	// to prevent.
	tl, ok := toolByName("check_task")
	if !ok {
		t.Fatal("check_task is not in the tool table — pick another integer-taking tool rather than skipping")
	}
	// An integer argument that is not a number in any spelling. `ref` is
	// supplied so the refusal cannot be the required-argument check instead.
	res := call(tl, map[string]any{"ref": "001", "n": "not a number"}, exec)
	if !res.IsError {
		t.Fatalf("a malformed argument did not surface as an error: %+v", res)
	}
	if executed {
		t.Error("the command RAN despite a malformed argument — validation must precede execution")
	}
	if len(res.Content) == 0 || !contains(res.Content[0].Text, `"n"`) {
		t.Errorf("the error must name the argument: %+v", res.Content)
	}

	// The same call with a well-formed n DOES execute, or the guard is just
	// refusing everything.
	executed = false
	if res := call(tl, map[string]any{"ref": "001", "n": float64(2)}, exec); res.IsError {
		t.Errorf("a well-formed call was refused: %+v", res.Content)
	}
	if !executed {
		t.Error("a well-formed call never reached the executor")
	}
}
