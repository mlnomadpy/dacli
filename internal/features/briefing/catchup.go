package briefing

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/clikit"
	"github.com/mlnomadpy/dacli/internal/eventlog"
	"github.com/mlnomadpy/dacli/internal/model"
	"github.com/mlnomadpy/dacli/internal/ulid"
)

// cmdCatchup answers "what have my siblings recorded since my brief was
// written?".
//
// A brief is assembled once, at spawn. In a wave of four agents that means the
// fourth is reading a snapshot taken before the first three found anything, so
// it re-derives what a sibling already filed ten minutes ago and files the
// duplicate (task 274). Re-assembling a full brief mid-run is the obvious fix
// and the wrong one: it costs a full task-tree walk plus the whole prompt
// budget, per check, to surface a handful of lines.
//
// This reads only the append-only log — the one place a sibling's finding
// lands the instant it is written, with no owner sync required — and filters
// by time. Cheap enough for an agent to run between steps, which is the only
// way it actually gets run.
func cmdCatchup(ctx *clikit.Ctx, args []string) error {
	w, _, err := clikit.OpenWorkspace(ctx)
	if err != nil {
		return err
	}
	f, _ := clikit.ParseFlags(args)
	if err := f.Reject("since", "project", "about", "mine"); err != nil {
		return err
	}

	since, err := sinceCutoff(f.Get("since"))
	if err != nil {
		return err
	}

	events, err := eventlog.List(w, eventlog.Query{})
	if err != nil {
		return err
	}

	// Findings and decisions are what a sibling records that changes another
	// agent's plan. Claims matter too: they say a task is already taken.
	interesting := map[model.EventKind]bool{
		model.EventFinding: true, model.EventClaim: true, model.EventBlock: true,
	}

	self := strings.TrimSpace(f.Get("mine"))
	about := strings.TrimSpace(f.Get("about"))

	var lines []string
	for _, e := range events {
		if !interesting[e.Kind] {
			continue
		}
		if !since.IsZero() && eventTime(e).Before(since) {
			continue
		}
		// An agent's own filings are not news to it. Excluding them is what
		// makes the output short enough to be worth reading.
		if self != "" && e.Actor == self {
			continue
		}
		if about != "" && e.About != about {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s  %-8s %-14s %s", clikit.Short(e.ID, 10), e.Kind, e.Actor, firstLine(e.Body)))
	}
	sort.Strings(lines)

	if len(lines) == 0 {
		fmt.Fprintln(ctx.Stdout, "nothing new from your siblings since that point — carry on")
		return nil
	}
	fmt.Fprintf(ctx.Stdout, "%d thing(s) your siblings recorded since then:\n", len(lines))
	for _, l := range lines {
		fmt.Fprintln(ctx.Stdout, l)
	}
	fmt.Fprintln(ctx.Stdout, "\n(read one in full with `dacli events tail`, or the task it is about with `dacli task show <ref>`)")
	return nil
}

// sinceCutoff parses --since as a duration back from now ("20m", "2h"). Empty
// means the whole log, which is the honest default for an agent that does not
// know when its brief was cut.
func sinceCutoff(v string) (time.Time, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return time.Time{}, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return time.Time{}, clikit.Usagef("--since must be a duration like 20m or 2h, got %q", v)
	}
	return time.Now().Add(-d), nil
}

// eventTime recovers an event's creation time from its ULID prefix, which is
// millisecond-precision and already lexically ordered — no extra file read.
func eventTime(e *eventlog.Event) time.Time {
	if t, ok := ulid.Time(e.ID); ok {
		return t
	}
	return time.Time{}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return clikit.Short(strings.TrimSpace(s), 90)
}
