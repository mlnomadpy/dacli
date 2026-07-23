package store

import (
	"strings"

	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// DuplicateTitleThreshold is the TitleSimilarity score at or above which two
// task titles are treated as the same piece of work. Calibrated against the
// two near-duplicate pairs task 116 was filed over (0.75 and 0.50) — high
// enough that two tasks sharing only a handful of common domain words
// (loop, governor, token) do not collide.
const DuplicateTitleThreshold = 0.5

// titleStopWords are function words stripped before comparing titles — left
// in, two titles about unrelated work but both full of "the"/"to"/"and"
// would look artificially similar.
var titleStopWords = map[string]bool{
	"a": true, "an": true, "the": true, "to": true, "of": true, "in": true,
	"on": true, "for": true, "and": true, "or": true, "is": true, "are": true,
	"this": true, "that": true, "with": true, "from": true, "into": true,
	"per": true, "via": true, "at": true, "by": true, "as": true, "be": true,
	"so": true, "it": true, "its": true, "if": true, "when": true, "not": true,
	"no": true, "do": true, "does": true, "did": true, "all": true, "each": true,
}

// stemTitleWord crudely strips a common inflectional suffix so "review" and
// "reviewer", or "token" and "tokens", compare equal. It is not real
// morphology — just enough to catch the same-words-different-tense titles a
// review auditor tends to produce for work already on the backlog.
func stemTitleWord(w string) string {
	for _, suf := range []string{"ers", "ing", "ed", "er", "es", "s"} {
		if strings.HasSuffix(w, suf) && len(w)-len(suf) >= 3 {
			return w[:len(w)-len(suf)]
		}
	}
	return w
}

// titleTokens splits a title into a normalized, stopword-filtered, stemmed
// token set.
func titleTokens(title string) map[string]bool {
	toks := map[string]bool{}
	var b strings.Builder
	flush := func() {
		if b.Len() == 0 {
			return
		}
		w := b.String()
		b.Reset()
		if titleStopWords[w] {
			return
		}
		toks[stemTitleWord(w)] = true
	}
	for _, r := range strings.ToLower(title) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			flush()
		}
	}
	flush()
	return toks
}

// titleOverlap computes both the Jaccard similarity and the raw count of
// shared stemmed tokens between two titles. The raw count matters alongside
// the ratio: two short titles sharing only one generic word ("Feature A" vs
// "Feature B") can score deceptively high on Jaccard alone.
func titleOverlap(a, b string) (score float64, shared int) {
	ta, tb := titleTokens(a), titleTokens(b)
	if len(ta) == 0 || len(tb) == 0 {
		return 0, 0
	}
	inter := 0
	for w := range ta {
		if tb[w] {
			inter++
		}
	}
	union := len(ta) + len(tb) - inter
	if union == 0 {
		return 0, inter
	}
	return float64(inter) / float64(union), inter
}

// TitleSimilarity scores how alike two task titles are, as a Jaccard index
// (0..1) over normalized, stemmed word sets.
func TitleSimilarity(a, b string) float64 {
	score, _ := titleOverlap(a, b)
	return score
}

// minSharedTitleTokens is the floor on shared content words before two
// titles are even considered for the dedup check — without it, two short,
// otherwise-unrelated titles that happen to share one generic word can clear
// DuplicateTitleThreshold on Jaccard ratio alone.
const minSharedTitleTokens = 2

// FindNearDuplicateTask returns the open or active task in project whose
// title most resembles title, provided its TitleSimilarity score clears
// DuplicateTitleThreshold (and the two titles share enough real content, not
// just a single common word). It is the guard against a review auditor
// re-filing work the backlog already carries under slightly different
// wording (dacli task 116): nil, 0 means no existing task looks like the
// same work.
func FindNearDuplicateTask(w *workspace.Workspace, project, title string) (*Task, float64, error) {
	var best *Task
	var bestScore float64
	for _, st := range []model.Status{model.StatusOpen, model.StatusActive} {
		ts, err := ListTasks(w, project, st)
		if err != nil {
			return nil, 0, err
		}
		for _, t := range ts {
			score, shared := titleOverlap(title, t.Title)
			if shared < minSharedTitleTokens {
				continue
			}
			if score > bestScore {
				best, bestScore = t, score
			}
		}
	}
	if bestScore < DuplicateTitleThreshold {
		return nil, bestScore, nil
	}
	return best, bestScore, nil
}
