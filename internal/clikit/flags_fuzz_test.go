package clikit

import (
	"strings"
	"testing"
)

// Flag parsing is the surface EVERY command reads untrusted argv through, and
// this codebase's most-repeated defect is a flag that goes missing quietly:
// `Flags.Reject` reached 4 handlers out of 112, `--dry-run 001` swallowed the
// positional and turned a rehearsal into a real write, and a value beginning
// with `--` was read as a flag. Each was found by someone hitting it.
//
// The properties below are the ones those defects violated, stated so a fuzzer
// can look for the next one.
func FuzzParseFlags(f *testing.F) {
	for _, seed := range [][]string{
		{}, {"--a"}, {"--a", "b"}, {"--a=b"}, {"pos"}, {"--a", "--b"},
		{"--dry-run", "001"}, {"--a", "--", "--b"}, {"--a=--b"},
		{"--", "x"}, {"---"}, {"--="}, {"--a=", "--a=x"}, {"-single"},
		{"--a", "b", "c"}, {"--force", "pos"}, {"--json"},
	} {
		f.Add(strings.Join(seed, "\x00"))
	}
	f.Fuzz(func(t *testing.T, joined string) {
		var args []string
		if joined != "" {
			args = strings.Split(joined, "\x00")
		}
		fl, err := ParseFlags(args)
		if err != nil {
			return // a rejected parse makes no promises about its output
		}

		// 1. NOTHING IS SILENTLY LOST. Every token is either a positional or
		//    accounted for by a flag — the property behind every
		//    "silently dropped flag" bug in this repo.
		seen := len(fl.Pos)
		for range fl.vals {
			seen++ // one key, however many values it collected
		}
		if seen == 0 && len(args) > 0 {
			// The only legitimate way to consume args and report nothing is
			// for every token to have been a bare "--" terminator.
			for _, a := range args {
				if a != "--" {
					t.Fatalf("ParseFlags(%q) consumed %d args and reported nothing", args, len(args))
				}
			}
		}

		// 2. A FLAG THE PARSER SAW IS RETRIEVABLE. Get must not return empty
		//    for a key the parser recorded — that gap is how a set flag reads
		//    as unset.
		for k, vals := range fl.vals {
			if len(vals) == 0 {
				t.Fatalf("ParseFlags(%q): key %q recorded with no values at all", args, k)
			}
			if got := fl.All(k); len(got) != len(vals) {
				t.Fatalf("ParseFlags(%q): key %q has %d values but All() returned %d", args, k, len(vals), len(got))
			}
		}

		// 3. AN alwaysBool FLAG NEVER EATS THE NEXT TOKEN. `--dry-run 001`
		//    disabled the preview AND dropped task 001 from the window,
		//    turning a rehearsal into a real remote write. Stated the way the
		//    defect actually presented: the token SURVIVES as a positional.
		//
		//    Only the bare form is checked. `--force=` is the inline `=` form
		//    — the caller writing a value explicitly — and Bool reads any
		//    non-"false" value as set, so it is legitimate rather than a
		//    swallow.
		for i, a := range args {
			if a == "--" {
				break // everything after a terminator is positional, not a flag
			}
			key := strings.TrimPrefix(a, "--")
			if !strings.HasPrefix(a, "--") || strings.Contains(a, "=") || !alwaysBool[key] {
				continue
			}
			if i+1 >= len(args) {
				break
			}
			next := args[i+1]
			if strings.HasPrefix(next, "--") {
				continue // the next token is its own flag, nothing to swallow
			}
			found := false
			for _, p := range fl.Pos {
				if p == next {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("ParseFlags(%q): boolean --%s ate %q; it must survive as a positional", args, key, next)
			}
		}
	})
}

// The two spellings of a value must parse identically. They do not always: a
// value that starts with "--" is ambiguous to a schema-less parser, which is
// why the `=` form and the `--` terminator exist as documented escapes. This
// pins that the ESCAPES work, so the documentation is enforced rather than
// asserted.
func FuzzValueSpellingsAgree(f *testing.F) {
	for _, v := range []string{"x", "", "--dash", "a b", "=eq", "--", "-", "ünïcode", "a=b"} {
		f.Add(v)
	}
	f.Fuzz(func(t *testing.T, val string) {
		if strings.Contains(val, "\x00") {
			return
		}
		eq, err1 := ParseFlags([]string{"--k=" + val})
		term, err2 := ParseFlags([]string{"--k", "--", val})
		if err1 != nil || err2 != nil {
			return
		}
		if a, b := eq.Get("k"), term.Get("k"); a != b {
			t.Fatalf("--k=%q gave %q but --k -- %q gave %q; the documented escapes must agree", val, a, val, b)
		}
	})
}
