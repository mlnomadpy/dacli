// Package brief assembles the context document handed to a subagent.
//
// This is the product. Everything else in dacli exists so that this function
// has something to slice.
package brief

import (
	"errors"

	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

// Options controls assembly.
type Options struct {
	// Budget is an approximate token ceiling. Zero means unlimited.
	Budget int

	// Depth bounds [[wikilink]] following when pulling in references.
	Depth int

	// JSON returns the sections structured instead of rendered, for the MCP
	// front end.
	JSON bool
}

// Part identifies a section of the brief. The order of these constants is the
// priority order, and trimming removes from the BOTTOM — so the highest-value
// content is never what gets cut.
type Part int

const (
	// PartTask is the task itself, in full. Never trimmed. If the task alone
	// exceeds the budget, assembly fails rather than silently truncating the
	// one thing the agent definitely needs.
	PartTask Part = iota

	// PartGoalChain is the project vision, goal, success criteria, and
	// ancestor task titles from root down. Establishes why the work exists.
	PartGoalChain

	// PartScope is the project's out-of-scope list. It sits this high because
	// it is small and because it is the only scope-creep intervention that
	// lands before the tokens are spent. Scope creep in an agent tree is a
	// child deciding to also fix the adjacent thing.
	PartScope

	// PartConstraints is project constraints plus every decision note in
	// scope, each with its Chose/Rejected/Because. This is the section that
	// stops a child from re-proposing an already-rejected option, and it is
	// why decision notes are first-class objects.
	PartConstraints

	// PartRisks is open rank-1 and rank-2 risks with their indicators. A risk
	// register helps an agent only in this form: here is what is likely to go
	// wrong, and here is what the early warning looks like.
	PartRisks

	// PartGlossary is the project term list — one definition per term that
	// every agent in the tree shares, countering vague nouns at the source.
	PartGlossary

	// PartFindings is findings from sibling and recently-completed tasks,
	// ranked by severity then recency. Stops a child from re-walking a dead
	// end a sibling already mapped.
	PartFindings

	// PartRefs is [[wikilink]] targets within Depth, excerpted.
	PartRefs

	// PartEvents is recent activity in this task's subtree.
	PartEvents
)

// Brief is an assembled context document.
type Brief struct {
	Subject  model.Ref
	Sections []Section
	Omitted  []Omission
}

type Section struct {
	Part    Part
	Title   string
	Content string
}

// Omission records something the budget forced out. Every trim is reported —
// inline in the markdown as an HTML comment, and here for the JSON path.
//
// A brief that looks complete but silently isn't is worse than one that admits
// its gaps: the agent cannot ask for what it does not know is missing.
type Omission struct {
	Part   Part
	Count  int
	Reason string
}

// Assemble builds the brief for ref.
//
// Reads fold in pending events, so a sibling's finding is visible here the
// instant it is appended — no sync required.
func Assemble(w *workspace.Workspace, ref model.Ref, opt Options) (*Brief, error) {
	// TODO, in priority order:
	//   1. resolve ref to a task (or project) and emit it whole
	//   2. walk parent links to the project; emit vision, goal, criteria
	//   3. emit the project's out-of-scope list
	//   4. collect decision notes reachable from the task and its ancestors
	//   5. collect open rank-1 and rank-2 risks with indicators
	//   6. emit the project glossary
	//   7. collect findings on siblings and recently-done tasks, ranked by
	//      severity then recency
	//   8. resolve [[wikilinks]] to Depth, excerpt
	//   9. tail recent events for the subtree
	// then trim from the bottom until EstimateTokens fits Budget, recording
	// an Omission for every drop.
	//
	// Miller's Law binds steps 4 and 5: cap constraints and risks at a
	// single-digit count and report the overflow. An agent handed 40
	// constraints drops most of them silently, exactly as a human would, so
	// emitting all of them is worse than admitting the cap.
	return nil, errors.New("not implemented")
}

// Render produces the markdown document.
func (b *Brief) Render() string { return "" }

// EstimateTokens approximates token count.
//
// chars/4 is a heuristic and is wrong per-model; see DESIGN.md § 11 open
// question 1. It is used rather than a real tokenizer because pulling one in
// costs a large dependency, and because every trim is announced anyway — the
// agent can see that the estimate bit, which is most of the value a precise
// count would provide.
func EstimateTokens(s string) int { return len(s) / 4 }
