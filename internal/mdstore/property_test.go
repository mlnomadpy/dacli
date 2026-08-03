package mdstore

import (
	"strings"
	"testing"
)

// FuzzFrontMatterRoundTrip asserts the single property the whole store rests
// on: a value written with Set and read back with Get is the value you wrote.
//
// This property is worth more than any number of example tests. Four real,
// separately-found bugs were all violations of this one line — CRLF input
// silently yielding empty frontmatter, a newline in a value making the file
// unparseable (and the task invisible), an inline " #" truncating a value, and
// surrounding quotes being stripped. Each was found by a human noticing, long
// after it shipped. The property finds them in seconds.
func FuzzFrontMatterRoundTrip(f *testing.F) {
	for _, seed := range []string{
		"plain", "with space", "fix bug #42", "a: colon", "  padded  ",
		"line1\nline2", "line1\r\nline2", "---", "\"quoted\"", "'single'",
		"[bracket]", "{brace}", "| pipe", "> gt", "unicode ⵟ ok", "",
		"trailing\\", "tab\there", "# leading hash", "-- dashes",
	} {
		f.Add("k", seed)
	}
	f.Fuzz(func(t *testing.T, key, val string) {
		// Keys are structural, not user data: restrict to what the format can
		// legally express, so the property is about VALUES.
		if key == "" || strings.ContainsAny(key, ":\r\n\t #\"'") || key != strings.TrimSpace(key) {
			t.Skip()
		}

		d := &Doc{}
		d.Front.Set(key, val)
		d.SetSection("Body", "content\n")

		reparsed, err := Parse(Render(d))
		if err != nil {
			t.Fatalf("a document written by Render must always re-parse; Set(%q, %q) produced:\n%s\nerror: %v",
				key, val, Render(d), err)
		}
		got, ok := reparsed.Front.Get(key)
		if !ok {
			t.Fatalf("key %q vanished after a round trip; value was %q", key, val)
		}
		// CR and LF cannot survive a line-oriented format literally; the
		// contract is that they round-trip through escaping. Everything else
		// must come back byte-for-byte.
		if got != val {
			t.Fatalf("round trip changed the value:\n  wrote %q\n  read  %q\n  rendered:\n%s", val, got, Render(d))
		}
	})
}

// FuzzParseNeverPanics: task files are hand-editable by design (the docs tell
// operators to edit them), so Parse is exposed to arbitrary text. It may
// return an error, but it must never panic — a crash in the parser takes down
// every command that touches the workspace.
func FuzzParseNeverPanics(f *testing.F) {
	for _, seed := range []string{
		"", "---", "---\n", "---\n---\n", "---\r\n---\r\n", "---\nk: v\n---\nbody",
		"---\nno-colon-line\n---\n", "---\n  indented: v\n---\n", "# heading only",
		"---\nk: |\n  block\n---\n", "```\nunclosed fence\n", "---\nk: v\n",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		d, err := Parse(raw)
		if err != nil {
			return // a refusal is a valid answer
		}
		// A successful parse must also survive being rendered and re-parsed:
		// anything else means Parse accepted something it cannot reproduce,
		// which is how a rewrite silently corrupts a file.
		if _, err := Parse(Render(d)); err != nil {
			t.Fatalf("Parse accepted input it cannot round-trip.\ninput:\n%q\nrendered:\n%q\nerror: %v",
				raw, Render(d), err)
		}
	})
}
