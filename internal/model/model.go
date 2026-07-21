// Package model defines the object types described in docs/FORMAT.md:
// the collaboration core (project, task, note, queue, agent) plus the
// satellite kinds later layers added (risk, role, shortcut, and — spec only —
// runtime and template). Events are the write mechanism, not a peer object.
//
// Every type here maps to exactly one markdown file on disk. Fields that the
// format spec calls "structural" (Goal, Constraints, Chose/Rejected/Because)
// are parsed out by heading; everything else stays in Body verbatim.
package model

import (
	"time"

	"github.com/mlnomadpy/dacli/internal/spm"
)

// Kind identifies which object a file holds. It is the `kind:` frontmatter key.
type Kind string

const (
	KindProject  Kind = "project"
	KindTask     Kind = "task"
	KindNote     Kind = "note"
	KindQueue    Kind = "queue"
	KindAgent    Kind = "agent"
	KindEvent    Kind = "event"
	KindRisk     Kind = "risk"
	KindRole     Kind = "role"
	KindShortcut Kind = "shortcut"
	KindRuntime  Kind = "runtime"
)

// Priority is MoSCoW (Dai Clegg). It exists to fight Cart Before the Horse,
// the most common agent planning failure: given a decomposed problem, agents
// reliably start with the tractable, interesting piece rather than the
// load-bearing one.
type Priority string

const (
	PriorityMust   Priority = "must"
	PriorityShould Priority = "should"
	PriorityCould  Priority = "could"
	// PriorityWont is "would like but won't get" — kept, never deleted. A
	// recorded out-of-scope decision stops the next agent re-proposing it.
	PriorityWont Priority = "wont"
)

// Rank orders priorities for scheduling; lower sorts first.
func (p Priority) Rank() int {
	switch p {
	case PriorityMust:
		return 0
	case PriorityShould:
		return 1
	case PriorityCould:
		return 2
	default:
		return 3
	}
}

// Dep is a typed task dependency. The type is recorded rather than reduced to
// a plain blocked_by because SS is what makes two tasks genuinely safe to run
// in parallel — a distinction that decides whether a parent agent may fan out.
type Dep struct {
	On   Ref         `yaml:"on"`
	Type spm.DepType `yaml:"type"` // FS | SS | FF | SF; defaults to FS
}

// Meta is the frontmatter every object carries.
//
// Extra holds keys this build does not understand. It exists so that
// round-tripping a file written by a newer dacli, a third-party tool, or a
// human never silently drops data. Writers must emit it back.
type Meta struct {
	ID        string            `yaml:"id"`
	Kind      Kind              `yaml:"kind"`
	Created   time.Time         `yaml:"created"`
	CreatedBy string            `yaml:"created_by"`
	Tags      []string          `yaml:"tags,omitempty"`
	Extra     map[string]string `yaml:"-"`
}

// Status is a task's state. It is derived from the containing folder, never
// from frontmatter: tasks/open/, tasks/active/, tasks/blocked/, tasks/done/.
// If a file's frontmatter ever disagrees with its folder, the folder wins.
type Status string

const (
	StatusOpen    Status = "open"
	StatusActive  Status = "active"
	StatusBlocked Status = "blocked"
	StatusDone    Status = "done"
)

// AllStatuses is the canonical order, used for folder creation and listing.
var AllStatuses = []Status{StatusOpen, StatusActive, StatusBlocked, StatusDone}

// Project is the root of a context tree: a goal plus the constraints bounding it.
type Project struct {
	Meta
	Slug string

	// Stage places the project on the Cone of Uncertainty, which sets how
	// wide every estimate inside it is reported.
	Stage spm.Stage

	// Structural sections, parsed by heading.
	Title           string
	Vision          string // Wiegers: the long-term why
	Goal            string // the near-term what
	Constraints     []string
	SuccessCriteria []Checkbox

	// OutOfScope is the scope-creep defense, and it earns its place by being
	// emitted into every context brief. Scope creep in an agent tree is not a
	// client asking for more — it is a child agent helpfully deciding to also
	// refactor the thing next door. Telling it the boundary up front is the
	// only intervention that happens before the tokens are spent.
	OutOfScope []string

	// Glossary gives the project's terms one definition every agent shares,
	// which is the direct counter to the vague-noun ambiguity category.
	Glossary map[string]string

	Body string // everything under non-structural headings
	Path string
}

// Task is a unit of work owned by exactly one agent.
type Task struct {
	Meta
	Seq     int // NNN in the filename; per-project, for sort order only
	Slug    string
	Status  Status // from the folder
	Project string // project slug

	Owner  string // agent id; the only agent that may rewrite this file
	Parent Ref    // optional parent task, forming the WBS

	// Priority is MoSCoW. `dacli next` never recommends a could while a must
	// is still open.
	Priority Priority

	// DependsOn carries typed dependencies, feeding the CPM scheduler.
	DependsOn []Dep

	// Estimate is three-point, never scalar. An agent asked for a number
	// produces a confident point value with no error bars; requiring
	// Pessimistic forces the unexamined risk to be stated.
	Estimate spm.ThreePoint

	// Traces links this task to the code and tests that satisfy it, which is
	// the traceability quality criterion and the only way to answer "which
	// requirement does this change serve".
	Traces []string

	Title string

	// SoThat is the value half of "As a [who], I want [what], so that [why]".
	// Its absence is what the INVEST Valuable check looks for.
	SoThat string

	Context string

	// Acceptance is the INVEST Testable criterion, and the single
	// highest-value lint in the tool: a subagent handed a task with no
	// acceptance criteria cannot know when to stop, so it stops too early or
	// burns its whole budget.
	Acceptance []Checkbox

	Log []string

	Body string
	Path string
}

// NoteKind distinguishes the three note flavors. Decisions are the highest
// value item in any context brief: they are what stops a child agent from
// re-proposing an option the parent already rejected.
type NoteKind string

const (
	NoteDecision NoteKind = "decision"
	NoteFinding  NoteKind = "finding"
	NoteRef      NoteKind = "ref"
	// NoteMetric is a GQM chain. Basili's rule is enforced structurally: you
	// cannot state the metric without first stating the goal and the
	// question. An agent asked to "add some metrics" otherwise produces
	// whatever is easiest to count — the lines-of-code failure in a costume.
	NoteMetric NoteKind = "metric"
)

// Note is durable agent output.
type Note struct {
	Meta
	NoteKind NoteKind
	Project  string
	About    Ref // task or project this attaches to

	Title string

	// Structural, decisions only. A decision without Rejected is a decision
	// nobody can safely revisit; dacli lint flags it.
	Chose    string
	Rejected string
	Because  string

	// Severity applies to findings, using the review-technique classification.
	// It lets the context assembler rank findings by consequence rather than
	// only by recency.
	Severity spm.Severity

	// GQM chain, metrics only.
	MetricGoal     string
	MetricQuestion string
	MetricMetric   string

	Body string
	Path string
}

// Risk is an entry in the impact×likelihood matrix.
//
// Risks are emitted into context briefs, so a child working near a known risk
// is told both that it exists and what its indicator looks like.
type Risk struct {
	Meta
	Project string
	Slug    string

	Title      string
	Impact     Level
	Likelihood Level

	// Indicators are the observable signs that this risk is materializing.
	Indicators []string

	// Action is the mitigation plan. Required for rank 1 and 2; `dacli lint`
	// flags a rank-1 risk with no action.
	Action string

	Body string
	Path string
}

// Level is a coarse impact or likelihood band.
type Level string

const (
	LevelHigh   Level = "high"
	LevelMedium Level = "medium"
	LevelLow    Level = "low"
)

// Rank returns the risk ranking: 1 means mitigate immediately, 2 means make a
// plan, 3 means monitor only. Only 1 and 2 require an action plan.
func (r Risk) Rank() int {
	switch {
	case r.Impact == LevelHigh && r.Likelihood == LevelHigh:
		return 1
	case r.Impact == LevelHigh && r.Likelihood == LevelMedium:
		return 2
	case r.Impact == LevelLow || r.Likelihood == LevelLow:
		return 3
	default:
		return 2
	}
}

// Queue is an ordered list of steps plus a cursor. dacli never executes a
// step; it reports which one is next and records that the agent moved on.
type Queue struct {
	Meta
	Slug   string
	Title  string
	Steps  []string
	Cursor int // index of the next step, 0-based

	Path string
}

// Grant is a capability. Attenuation is monotonic: a child's grant may never
// exceed its parent's.
type Grant string

const (
	GrantRW Grant = "rw"
	GrantRO Grant = "ro"
)

// Exceeds reports whether g is strictly more permissive than parent.
func (g Grant) Exceeds(parent Grant) bool {
	return g == GrantRW && parent != GrantRW
}

// Agent is an identity in the tree. The token itself is never stored, only
// its hash; it is displayed once, at spawn.
type Agent struct {
	Meta
	Parent    Ref
	Grant     Grant
	Role      string
	TokenHash string

	Body string
	Path string
}

// EventKind enumerates the append-only write kinds. Any agent, including a
// read-only one, may append events; that is how a read-only agent reports
// results without being able to mutate objects it does not own.
type EventKind string

const (
	EventClaim         EventKind = "claim"
	EventRelease       EventKind = "release"
	EventFinding       EventKind = "finding"
	EventProposeStatus EventKind = "propose-status"
	EventComment       EventKind = "comment"
	EventBlock         EventKind = "block"

	// EventHelp is a typed help request: a specific question, addressed to
	// whichever role owns the path in question, that blocks the asking task.
	//
	// This is deliberately not a chat message. A conversation between agents
	// has no completion criterion and converges without adding information,
	// because agents are agreeable. A help request has exactly one thing that
	// ends it: an answer. See docs/TEAM.md § 3.
	EventHelp EventKind = "help"

	// EventAnswer resolves a help request. The answer is promoted to a
	// decision or finding note on sync, so it enters every future brief in
	// scope. The question is transient; the answer is permanent.
	EventAnswer EventKind = "answer"

	// EventRun records a shortcut invocation. Every command an agent runs
	// through dacli is attributed, which is what makes both anti-pattern
	// detection and shortcut promotion possible.
	EventRun EventKind = "run"
)

// Event is one immutable write. Filename is <ULID>-<agent>-<kind>.md, so the
// directory listing is the ordered log and two concurrent writers never
// contend for the same path.
type Event struct {
	Meta      // ID is a ULID
	EventKind EventKind
	About     Ref

	// Applied is the sole mutable field in the format, written only by the
	// owner of About during `dacli sync`.
	Applied bool

	Body string
	Path string
}

// Ref is a [[wikilink]] target. An unresolved Ref is valid: it marks
// something worth writing later rather than an error.
type Ref string

// Checkbox is a markdown task-list item.
type Checkbox struct {
	Text string
	Done bool
}
