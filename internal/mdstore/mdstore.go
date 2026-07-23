// Package mdstore reads and writes the markdown-with-frontmatter files that
// make up a workspace.
//
// Two invariants this package must never violate:
//
//  1. Round-tripping preserves unknown frontmatter keys and full-line
//     comments. A file written by a newer dacli, a third-party tool, or a
//     human must survive being read and rewritten by this build with nothing
//     dropped.
//
//  2. Writes are atomic. Everything goes to a temp file in the same directory
//     followed by a rename, so a crash mid-write leaves the old file intact
//     rather than a truncated one. Half a task file is worse than none.
//
// The frontmatter dialect is deliberately narrow — top-level `key: value`
// scalars, inline lists `[a, b]`, inline maps `{k: v}`, and indented blocks
// preserved verbatim. The format spec only uses those; a full YAML parser is
// a dependency and an attack surface this module does not need.
package mdstore

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Entry is one frontmatter line (or block). Key == "" means a full-line
// comment or blank line, preserved verbatim in Raw.
type Entry struct {
	Key   string
	Value string // scalar value as written (may carry a trailing comment)
	Block string // for `key:` followed by indented lines: those lines, verbatim
	Raw   string // for Key == "": the exact line
}

// Front is ordered frontmatter. Order is preserved so rewrites produce
// minimal diffs — these files live in git and get reviewed by humans.
type Front struct {
	entries []Entry
}

// Get returns the cleaned scalar value: surrounding quotes removed, trailing
// ` # comment` stripped. Returns false for absent keys and block entries.
func (f *Front) Get(k string) (string, bool) {
	for _, e := range f.entries {
		if e.Key == k && e.Block == "" {
			return clean(e.Value), true
		}
	}
	return "", false
}

// GetBlock returns the raw indented block under a key, if any.
func (f *Front) GetBlock(k string) (string, bool) {
	for _, e := range f.entries {
		if e.Key == k && e.Block != "" {
			return e.Block, true
		}
	}
	return "", false
}

// Set adds or replaces a scalar value, preserving position on replace.
func (f *Front) Set(k, v string) {
	for i, e := range f.entries {
		if e.Key == k {
			f.entries[i] = Entry{Key: k, Value: v}
			return
		}
	}
	f.entries = append(f.entries, Entry{Key: k, Value: v})
}

// SetBlock adds or replaces an indented-block value (e.g. the github:
// mapping). Block lines must arrive already indented; they render verbatim.
func (f *Front) SetBlock(k, block string) {
	for i, e := range f.entries {
		if e.Key == k {
			f.entries[i] = Entry{Key: k, Block: block}
			return
		}
	}
	f.entries = append(f.entries, Entry{Key: k, Block: block})
}

// Delete removes a keyed entry, if present.
func (f *Front) Delete(k string) {
	for i, e := range f.entries {
		if e.Key == k {
			f.entries = append(f.entries[:i], f.entries[i+1:]...)
			return
		}
	}
}

// Keys returns the keyed entries in order.
func (f *Front) Keys() []string {
	var out []string
	for _, e := range f.entries {
		if e.Key != "" {
			out = append(out, e.Key)
		}
	}
	return out
}

// GetText returns a key's human-readable text: a scalar's cleaned value, or
// a literal/folded block's dedented content joined with newlines.
func (f *Front) GetText(k string) (string, bool) {
	for _, e := range f.entries {
		if e.Key != k {
			continue
		}
		if e.Block == "" {
			return clean(e.Value), true
		}
		var lines []string
		for _, l := range strings.Split(e.Block, "\n") {
			lines = append(lines, strings.TrimSpace(l))
		}
		return strings.TrimSpace(strings.Join(lines, "\n")), true
	}
	return "", false
}

// GetList parses an inline list value: `[a, b, "c d"]`.
func (f *Front) GetList(k string) []string {
	v, ok := f.Get(k)
	if !ok || !strings.HasPrefix(v, "[") || !strings.HasSuffix(v, "]") {
		return nil
	}
	return splitTop(v[1 : len(v)-1])
}

// SetList encodes v as an inline list and stores it under k — the exact
// inverse of GetList: any element containing a comma, bracket, brace, quote,
// or leading/trailing whitespace is quoted so it round-trips losslessly
// through splitTop/clean.
func (f *Front) SetList(k string, v []string) {
	quoted := make([]string, len(v))
	for i, elem := range v {
		quoted[i] = quoteListElem(elem)
	}
	f.Set(k, "["+strings.Join(quoted, ", ")+"]")
}

// quoteListElem wraps an inline-list element in quotes when it contains a
// character splitTop/clean treat as significant (a comma would otherwise
// re-split the element into extra list entries; brackets, braces, or a `#`
// would otherwise be misread as structure or a comment), or when leading/
// trailing whitespace would otherwise be trimmed on read-back.
func quoteListElem(s string) string {
	if !strings.ContainsAny(s, ",[]{}#\"'") && s == strings.TrimSpace(s) {
		return s
	}
	if strings.Contains(s, "\"") && !strings.Contains(s, "'") {
		return "'" + s + "'"
	}
	return "\"" + s + "\""
}

// GetMap parses an inline map value: `{a: 1, b: two}`.
func (f *Front) GetMap(k string) map[string]string {
	v, ok := f.Get(k)
	if !ok || !strings.HasPrefix(v, "{") || !strings.HasSuffix(v, "}") {
		return nil
	}
	out := map[string]string{}
	for _, part := range splitTop(v[1 : len(v)-1]) {
		if i := strings.Index(part, ":"); i >= 0 {
			out[strings.TrimSpace(part[:i])] = strings.TrimSpace(part[i+1:])
		}
	}
	return out
}

// clean strips a trailing ` # comment` (outside quotes) and surrounding quotes.
func clean(v string) string {
	v = strings.TrimSpace(v)
	inQ := byte(0)
	for i := 0; i < len(v); i++ {
		c := v[i]
		switch {
		case inQ != 0:
			if c == inQ {
				inQ = 0
			}
		case c == '"' || c == '\'':
			inQ = c
		case c == '#' && i > 0 && (v[i-1] == ' ' || v[i-1] == '\t'):
			v = strings.TrimSpace(v[:i])
			i = len(v)
		}
	}
	if len(v) >= 2 && (v[0] == '"' && v[len(v)-1] == '"' || v[0] == '\'' && v[len(v)-1] == '\'') {
		v = v[1 : len(v)-1]
	}
	return v
}

// splitTop splits on commas not nested inside quotes/brackets/braces.
func splitTop(s string) []string {
	var out []string
	depth := 0
	inQ := byte(0)
	start := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case inQ != 0:
			if c == inQ {
				inQ = 0
			}
		case c == '"' || c == '\'':
			inQ = c
		case c == '[' || c == '{':
			depth++
		case c == ']' || c == '}':
			depth--
		case c == ',' && depth == 0:
			if p := strings.TrimSpace(s[start:i]); p != "" {
				out = append(out, clean(p))
			}
			start = i + 1
		}
	}
	if p := strings.TrimSpace(s[start:]); p != "" {
		out = append(out, clean(p))
	}
	return out
}

// Section is a markdown heading and the content under it, up to the next
// heading at any level. Level 0 with empty Title is body text before the
// first heading.
type Section struct {
	Level   int
	Title   string
	Content string
}

// Doc is a parsed markdown file.
type Doc struct {
	Front    Front
	Sections []Section
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

// SetSection replaces the content of the named section, or appends a new
// level-2 section if absent.
func (d *Doc) SetSection(title, content string) {
	for i, s := range d.Sections {
		if strings.EqualFold(s.Title, title) {
			d.Sections[i].Content = content
			return
		}
	}
	d.Sections = append(d.Sections, Section{Level: 2, Title: title, Content: content})
}

// Parse reads a document. Malformed frontmatter is an error, never a silent
// partial parse — a file half-understood is a file about to be corrupted on
// rewrite.
func Parse(raw string) (*Doc, error) {
	d := &Doc{}
	body := raw

	if strings.HasPrefix(raw, "---\n") {
		rest := raw[4:]
		end := strings.Index(rest, "\n---\n")
		var fm string
		switch {
		case end >= 0:
			fm, body = rest[:end], rest[end+5:]
		case strings.HasSuffix(rest, "\n---"):
			fm, body = rest[:len(rest)-4], ""
		default:
			return nil, fmt.Errorf("mdstore: unterminated frontmatter")
		}
		if err := parseFront(&d.Front, fm); err != nil {
			return nil, err
		}
	}

	d.Sections = parseSections(body)
	return d, nil
}

func parseFront(f *Front, fm string) error {
	lines := strings.Split(fm, "\n")
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Blank lines and full-line comments are preserved verbatim.
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			f.entries = append(f.entries, Entry{Raw: line})
			continue
		}
		if line[0] == ' ' || line[0] == '\t' {
			return fmt.Errorf("mdstore: unexpected indented frontmatter line %q", line)
		}
		colon := strings.Index(line, ":")
		if colon < 0 {
			return fmt.Errorf("mdstore: frontmatter line without key: %q", line)
		}
		key := strings.TrimSpace(line[:colon])
		val := strings.TrimSpace(line[colon+1:])

		// A bare `key:` (or one whose value is only a comment), or a YAML
		// literal/folded indicator (| |- > >-), followed by indented lines
		// is a block, preserved verbatim. The indicator case exists because
		// real native skills write `description: |` — found when the first
		// real library import parsed losslessly and read as nothing.
		isIndicator := val == "|" || val == "|-" || val == ">" || val == ">-"
		if val == "" || isIndicator || strings.HasPrefix(val, "#") {
			var block []string
			for i+1 < len(lines) && (strings.HasPrefix(lines[i+1], " ") || strings.HasPrefix(lines[i+1], "\t") || strings.TrimSpace(lines[i+1]) == "") {
				// Trailing blank lines belong to whatever follows, not the block.
				if strings.TrimSpace(lines[i+1]) == "" {
					allBlank := true
					for j := i + 1; j < len(lines); j++ {
						if strings.TrimSpace(lines[j]) != "" {
							allBlank = strings.HasPrefix(lines[j], " ") || strings.HasPrefix(lines[j], "\t")
							break
						}
					}
					if !allBlank {
						break
					}
				}
				i++
				block = append(block, lines[i])
			}
			if len(block) > 0 {
				f.entries = append(f.entries, Entry{Key: key, Value: val, Block: strings.Join(block, "\n")})
				continue
			}
		}
		f.entries = append(f.entries, Entry{Key: key, Value: val})
	}
	return nil
}

// parseSections splits on ATX headings, ignoring headings inside fenced code
// blocks — task bodies contain code, and a `# comment` inside a fence is not
// a heading.
func parseSections(body string) []Section {
	if body == "" {
		return nil
	}
	var out []Section
	cur := Section{Level: 0}
	var buf []string
	inFence := false

	flush := func() {
		// Every stored line is newline-terminated, so Content round-trips
		// byte-exactly: blank lines are real "" entries, and the file's final
		// newline is restored by the terminator on the last line.
		if len(buf) > 0 {
			cur.Content = strings.Join(buf, "\n") + "\n"
		} else {
			cur.Content = ""
		}
		if cur.Level != 0 || cur.Title != "" || strings.TrimSpace(cur.Content) != "" {
			out = append(out, cur)
		}
		buf = nil
	}

	lines := strings.Split(body, "\n")
	// The trailing "" from a final newline is an artifact of Split, not a
	// blank line; dropping it (and re-adding "\n" per line above) normalizes
	// files to newline-terminated, which is the only shape we ever write.
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence
		}
		if !inFence {
			if lvl, title, ok := heading(line); ok {
				flush()
				cur = Section{Level: lvl, Title: title}
				continue
			}
		}
		buf = append(buf, line)
	}
	flush()
	return out
}

func heading(line string) (int, string, bool) {
	i := 0
	for i < len(line) && line[i] == '#' {
		i++
	}
	if i == 0 || i > 6 || i >= len(line) || line[i] != ' ' {
		return 0, "", false
	}
	return i, strings.TrimSpace(line[i+1:]), true
}

// Render serializes a Doc: frontmatter entries in order (unknown keys and
// comments verbatim), then sections.
func Render(d *Doc) string {
	var b strings.Builder
	if len(d.Front.entries) > 0 {
		b.WriteString("---\n")
		for _, e := range d.Front.entries {
			switch {
			case e.Key == "":
				b.WriteString(e.Raw)
				b.WriteByte('\n')
			case e.Block != "":
				b.WriteString(e.Key)
				b.WriteByte(':')
				if e.Value != "" {
					b.WriteByte(' ')
					b.WriteString(e.Value)
				}
				b.WriteByte('\n')
				b.WriteString(e.Block)
				b.WriteByte('\n')
			default:
				b.WriteString(e.Key)
				b.WriteString(": ")
				b.WriteString(e.Value)
				b.WriteByte('\n')
			}
		}
		b.WriteString("---\n")
	}
	for _, s := range d.Sections {
		if s.Level > 0 {
			b.WriteString(strings.Repeat("#", s.Level))
			b.WriteByte(' ')
			b.WriteString(s.Title)
			b.WriteByte('\n')
		}
		// Content is newline-terminated by construction (parse) or by the
		// writer's convention (SetSection callers end with \n); normalize
		// rather than double-terminate.
		b.WriteString(s.Content)
		if s.Content != "" && !strings.HasSuffix(s.Content, "\n") {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// ReadFile parses the file at path.
func ReadFile(path string) (*Doc, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	d, err := Parse(string(raw))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return d, nil
}

// WriteFile renders d to path atomically: temp file in the same directory,
// fsync-free rename. The directory is created if needed.
func WriteFile(path string, d *Doc) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".dacli-tmp-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if _, err := tmp.WriteString(Render(d)); err != nil {
		tmp.Close()
		os.Remove(name)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return err
	}
	if err := os.Rename(name, path); err != nil {
		// A rename fault (cross-device link, EACCES, a dir replaced mid-write,
		// index lock) must not orphan the temp file in the object directory —
		// every workspace write funnels through here, so a transient fault
		// would otherwise litter the tree with .dacli-tmp-* files.
		os.Remove(name)
		return err
	}
	return nil
}

// Links extracts every [[wikilink]] target from s. Unresolved targets are
// valid: a dangling link marks something worth writing later, not an error.
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
			target = strings.TrimSpace(target[:pipe])
		}
		if target != "" {
			out = append(out, target)
		}
		s = s[i+2+j+2:]
	}
}

// Checkbox is a markdown task-list item.
type Checkbox struct {
	Text string
	Done bool
}

// Checkboxes extracts `- [ ]` / `- [x]` items from section content.
func Checkboxes(content string) []Checkbox {
	var out []Checkbox
	for _, line := range strings.Split(content, "\n") {
		t := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(t, "- [ ] "):
			out = append(out, Checkbox{Text: t[6:], Done: false})
		case strings.HasPrefix(t, "- [x] "), strings.HasPrefix(t, "- [X] "):
			out = append(out, Checkbox{Text: t[6:], Done: true})
		}
	}
	return out
}

// RenderCheckboxes produces the markdown for a checkbox list.
func RenderCheckboxes(boxes []Checkbox) string {
	var b strings.Builder
	for _, c := range boxes {
		if c.Done {
			b.WriteString("- [x] ")
		} else {
			b.WriteString("- [ ] ")
		}
		b.WriteString(c.Text)
		b.WriteByte('\n')
	}
	return b.String()
}

// Bullets extracts plain `- item` list entries (not checkboxes).
func Bullets(content string) []string {
	var out []string
	for _, line := range strings.Split(content, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "- ") && !strings.HasPrefix(t, "- [") {
			out = append(out, t[2:])
		}
	}
	return out
}
