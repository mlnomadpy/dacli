// Package spm implements the software product management frameworks that
// dacli encodes natively. See docs/SPM.md for the mapping and for which
// frameworks deliberately do not port to agent work.
package spm

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Category is one of the eleven categories of ambiguous language.
type Category string

const (
	CatIndirect    Category = "indirect-language"
	CatVague       Category = "vague-words"
	CatPersuasion  Category = "persuasion-words"
	CatCompletion  Category = "completion-words"
	CatQualifier   Category = "qualifiers"
	CatComparative Category = "comparatives"
	CatQuantity    Category = "quantities"
	CatPronoun     Category = "pronouns"
	CatPositional  Category = "positional-words"
	CatTemporal    Category = "temporal-words"
	CatJoining     Category = "joining-words"
)

// Severity reuses the review-technique classification, so an ambiguity
// finding and a code-review finding rank on the same scale.
type Severity string

const (
	// SevMajor means the fix is not obvious and requires exploration —
	// somebody has to go find out what was actually meant.
	SevMajor Severity = "major"
	// SevModerate means the fix is straightforward but needs review.
	SevModerate Severity = "moderate"
	// SevMinor means the fix is obvious or unnecessary.
	SevMinor Severity = "minor"
)

var severityRank = map[Severity]int{SevMajor: 0, SevModerate: 1, SevMinor: 2}

// Finding is one flagged term.
type Finding struct {
	Category Category
	Severity Severity
	Term     string
	Line     int
	Col      int
	Fix      string
}

func (f Finding) String() string {
	return fmt.Sprintf("%d:%d %s [%s] %q — %s", f.Line, f.Col, f.Severity, f.Category, f.Term, f.Fix)
}

type rule struct {
	cat   Category
	sev   Severity
	fix   string
	terms []string
	re    *regexp.Regexp
}

// The rule set. Severities are assigned by how expensive the ambiguity is to
// resolve later, not by how often it occurs: a vague verb costs a whole
// re-implementation, a stray pronoun usually costs nothing.
var rules = []*rule{
	{
		cat: CatVague, sev: SevMajor,
		fix:   "replace with a specific action verb and a defined noun; add the noun to the glossary",
		terms: []string{"process", "processed", "processing", "handle", "handled", "handling", "manage", "managed", "support", "supported", "item", "items", "entity", "entities", "thing", "things", "stuff", "appropriate", "appropriately", "properly", "correctly", "robust", "efficient", "efficiently", "user-friendly", "seamless", "seamlessly", "flexible", "optimize", "optimized", "improve", "improved", "enhance", "enhanced", "clean up", "refactor as needed", "as needed", "as appropriate", "if necessary", "where applicable"},
	},
	{
		cat: CatComparative, sev: SevMajor,
		fix:   "state the exact attribute and the reference point being compared against",
		terms: []string{"bigger", "smaller", "faster", "fastest", "slower", "slowest", "better", "best", "worse", "worst", "higher", "lower", "quicker", "cheaper", "stronger", "same as", "similar to", "comparable to", "on par with", "more efficient", "less complex"},
	},
	{
		cat: CatCompletion, sev: SevMajor,
		fix:   "complete the list explicitly; an open-ended list has no acceptance criterion",
		terms: []string{"and so on", "etc", "and more", "among others", "including but not limited to", "such as", "or similar", "and others", "the like"},
	},
	{
		cat: CatQuantity, sev: SevModerate,
		fix:   "replace with a specific number or a minimum threshold",
		terms: []string{"a few", "some", "most", "many", "several", "various", "multiple", "numerous", "minimal", "sufficient", "adequate", "reasonable", "a number of", "plenty"},
	},
	{
		cat: CatQualifier, sev: SevModerate,
		fix:   "specify the exact scope the qualifier ranges over",
		terms: []string{"all", "every", "only", "none", "never", "always", "any", "each", "entire", "whole"},
	},
	{
		cat: CatIndirect, sev: SevModerate,
		fix:   "specify the exact condition; use \"must\" for what is always true",
		terms: []string{"should", "could", "may", "might", "sometimes", "usually", "typically", "generally", "ought", "preferably", "ideally", "possibly"},
	},
	{
		cat: CatTemporal, sev: SevModerate,
		fix:   "state one-time vs. recurring, and the exact timing",
		terms: []string{"current", "currently", "latest", "recent", "recently", "soon", "eventually", "periodically", "regularly", "frequently", "in time", "later", "for now", "at some point"},
	},
	{
		cat: CatPositional, sev: SevModerate,
		fix:   "specify the exact relationship between the two things being ordered",
		terms: []string{"after", "before", "following", "preceding", "last", "prior", "previous", "subsequent", "above", "below"},
	},
	{
		cat: CatPersuasion, sev: SevMinor,
		fix:   "delete; persuasion words carry no requirement content",
		terms: []string{"clearly", "obviously", "certainly", "simply", "of course", "naturally", "evidently", "undoubtedly", "needless to say", "it is well known"},
	},
	{
		cat: CatPronoun, sev: SevMinor,
		fix:   "replace with the specific noun it refers to",
		terms: []string{"it", "they", "them", "this", "that", "these", "those", "its", "their"},
	},
	{
		cat: CatJoining, sev: SevMinor,
		fix:   "\"and\" often means two requirements — split them; \"or\" needs the both-true case defined",
		terms: []string{"and/or", "and", "or", "both"},
	},
}

func init() {
	for _, r := range rules {
		// Longest first so "more efficient" wins over "efficient".
		terms := append([]string(nil), r.terms...)
		sort.Slice(terms, func(i, j int) bool { return len(terms[i]) > len(terms[j]) })
		alts := make([]string, len(terms))
		for i, t := range terms {
			alts[i] = regexp.QuoteMeta(t)
		}
		r.re = regexp.MustCompile(`(?i)\b(?:` + strings.Join(alts, "|") + `)\b`)
	}
}

// Options tunes a scan.
type Options struct {
	// MinSeverity drops findings less severe than this. Defaults to
	// SevModerate: the minor categories (pronouns, joining words) are real
	// but noisy enough that surfacing them by default would train agents to
	// ignore the whole linter.
	MinSeverity Severity

	// Categories restricts the scan. Empty means all.
	Categories []Category
}

// Scan finds ambiguous language in text.
//
// Fenced and inline code spans are excluded. Flagging the word "process" in a
// snippet of Go would be noise, and a linter that cries wolf gets disabled.
func Scan(text string, opt Options) []Finding {
	if opt.MinSeverity == "" {
		opt.MinSeverity = SevModerate
	}
	maxRank := severityRank[opt.MinSeverity]

	allowed := map[Category]bool{}
	for _, c := range opt.Categories {
		allowed[c] = true
	}

	masked := maskCode(text)
	lineStarts := lineIndex(masked)

	var out []Finding
	for _, r := range rules {
		if len(allowed) > 0 && !allowed[r.cat] {
			continue
		}
		if severityRank[r.sev] > maxRank {
			continue
		}
		for _, loc := range r.re.FindAllStringIndex(masked, -1) {
			line, col := position(lineStarts, loc[0])
			out = append(out, Finding{
				Category: r.cat,
				Severity: r.sev,
				Term:     text[loc[0]:loc[1]],
				Line:     line,
				Col:      col,
				Fix:      r.fix,
			})
		}
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Line != out[j].Line {
			return out[i].Line < out[j].Line
		}
		if out[i].Col != out[j].Col {
			return out[i].Col < out[j].Col
		}
		return severityRank[out[i].Severity] < severityRank[out[j].Severity]
	})
	return out
}

// maskCode replaces fenced blocks and inline code with spaces, preserving
// every byte offset so positions still refer to the original text.
func maskCode(s string) string {
	b := []byte(s)
	n := len(b)
	blank := func(from, to int) {
		for i := from; i < to && i < n; i++ {
			if b[i] != '\n' {
				b[i] = ' '
			}
		}
	}

	// Fenced blocks: ``` ... ```
	for i := 0; i+2 < n; {
		if b[i] == '`' && b[i+1] == '`' && b[i+2] == '`' {
			end := bytes.Index(b[i+3:], []byte("```"))
			if end < 0 {
				blank(i, n)
				break
			}
			stop := i + 3 + end + 3
			blank(i, stop)
			i = stop
			continue
		}
		i++
	}

	// Inline code: `...` on a single line.
	for i := 0; i < n; i++ {
		if b[i] != '`' {
			continue
		}
		for j := i + 1; j < n && b[j] != '\n'; j++ {
			if b[j] == '`' {
				blank(i, j+1)
				i = j
				break
			}
		}
	}
	return string(b)
}

func lineIndex(s string) []int {
	starts := []int{0}
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			starts = append(starts, i+1)
		}
	}
	return starts
}

func position(starts []int, off int) (line, col int) {
	i := sort.Search(len(starts), func(i int) bool { return starts[i] > off }) - 1
	if i < 0 {
		i = 0
	}
	return i + 1, off - starts[i] + 1
}
