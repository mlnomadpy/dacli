// Package mdstore reads and writes the markdown-with-frontmatter files that
// make up a workspace.
//
// Two invariants this package must never violate:
//
//  1. Round-tripping preserves unknown frontmatter keys. A file written by a
//     newer dacli, a third-party tool, or a human must survive being read and
//     rewritten by this build with nothing dropped.
//
//  2. Writes are atomic. Everything goes to a temp file in the same directory
//     followed by a rename, so a crash mid-write leaves the old file intact
//     rather than a truncated one. Half a task file is worse than none.
package mdstore

import (
	"errors"
	"strings"
)

// Doc is a parsed markdown file: frontmatter as ordered key/value pairs, plus
// the body split into headed sections.
type Doc struct {
	Front    Front
	Sections []Section
	Raw      string
}

// Front is frontmatter. Order is preserved so rewrites produce minimal diffs —
// this matters because these files live in git and get reviewed by humans.
type Front struct {
	keys   []string
	values map[string]string
}

func (f *Front) Get(k string) (string, bool) {
	v, ok := f.values[k]
	return v, ok
}

func (f *Front) Set(k, v string) {
	if f.values == nil {
		f.values = map[string]string{}
	}
	if _, exists := f.values[k]; !exists {
		f.keys = append(f.keys, k)
	}
	f.values[k] = v
}

func (f *Front) Keys() []string { return append([]string(nil), f.keys...) }

// Section is a markdown heading and the content under it. Structural headings
// ("## Goal", "## Constraints", "## Rejected", ...) are located by Title.
type Section struct {
	Level   int
	Title   string
	Content string
}

// Section returns the first section with the given title, case-insensitively.
func (d *Doc) Section(title string) (Section, bool) {
	for _, s := range d.Sections {
		if strings.EqualFold(s.Title, title) {
			return s, true
		}
	}
	return Section{}, false
}

// Parse reads a document. The frontmatter subset here is deliberately narrow —
// scalars, inline lists, and quoted strings — because the format spec only
// uses those. A file with frontmatter this parser cannot handle is an error,
// never a silent partial parse.
func Parse(raw string) (*Doc, error) {
	// TODO: split on leading "---\n" ... "\n---\n"; parse flat YAML; split the
	// remainder into sections on ATX headings.
	return nil, errors.New("not implemented")
}

// Render serializes a Doc, emitting frontmatter keys in their original order
// and unknown keys verbatim.
func Render(d *Doc) (string, error) {
	return "", errors.New("not implemented")
}

// ReadFile parses the file at path.
func ReadFile(path string) (*Doc, error) {
	return nil, errors.New("not implemented")
}

// WriteFile renders d to path atomically (temp file in the same directory,
// then rename).
func WriteFile(path string, d *Doc) error {
	return errors.New("not implemented")
}

// Links extracts every [[wikilink]] target from s. Unresolved targets are
// valid and are returned like any other: in this format a dangling link marks
// something worth writing later, not an error.
func Links(s string) []string {
	var out []string
	for {
		i := strings.Index(s, "[[")
		if i < 0 {
			return out
		}
		j := strings.Index(s[i+2:], "]]")
		if j < 0 {
			return out
		}
		target := strings.TrimSpace(s[i+2 : i+2+j])
		if pipe := strings.Index(target, "|"); pipe >= 0 {
			target = strings.TrimSpace(target[:pipe]) // [[target|display]]
		}
		if target != "" {
			out = append(out, target)
		}
		s = s[i+2+j+2:]
	}
}
