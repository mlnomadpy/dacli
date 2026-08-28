package store

import (
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/commandresult"
)

// semanticCmdTimeout bounds the external semantic backend so a hung or slow
// scorer can never wedge a `task add`. The dedup check is a courtesy guard, not
// a correctness gate — if the backend does not answer in time we fall back to
// the lexical score rather than block the operator.
const semanticCmdTimeout = 10 * time.Second

// envSemanticBackend returns a SemanticScorer that delegates to the external
// command in $DACLI_SEMANTIC_CMD, or nil when the variable is unset. This is
// dacli's only seam for real semantic scoring: dacli ships no embedding model —
// bundling one would break the zero-dependency property (dacli task 249) — so
// an operator who wants paraphrase detection points this at their own scorer.
//
// Contract: the command is run via `sh -c <cmd> dacli-semantic <a> <b>`, so the
// two titles arrive as "$1" and "$2" (never interpolated into the shell, which
// keeps a title full of quotes or $() from being executed). It must print a
// single float in [0,1] on stdout — the meaning-similarity of the two titles,
// on the same scale as the lexical Jaccard index. Anything else (non-zero exit,
// timeout, unparseable output) is treated as "no opinion" (ok=false) so the
// pair falls back to the lexical score.
func envSemanticBackend() SemanticScorer {
	cmd := strings.TrimSpace(os.Getenv("DACLI_SEMANTIC_CMD"))
	if cmd == "" {
		return nil
	}
	return func(a, b string) (float64, bool) {
		return runSemanticCmd(cmd, a, b)
	}
}

// runSemanticCmd executes the operator's scorer against one title pair and
// parses its verdict. It is deliberately forgiving: any failure yields
// (0, false) so a broken backend degrades to lexical-only rather than crashing
// `task add`.
func runSemanticCmd(cmd, a, b string) (float64, bool) {
	score, ok, _ := runSemanticCmdChecked(cmd, a, b)
	return score, ok
}

// runSemanticCmdChecked exposes a governed command failure to diagnostic-aware
// callers while runSemanticCmd keeps semantic scoring optional. A missing,
// timed-out, or broken scorer is still "no opinion", never a task-add failure.
func runSemanticCmdChecked(cmd, a, b string) (float64, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), semanticCmdTimeout)
	defer cancel()
	// The titles ride as positional args ($1, $2), NOT spliced into the shell
	// string, so an operator's command stays a template and a hostile title
	// cannot inject shell.
	c := exec.CommandContext(ctx, "sh", "-c", cmd, "dacli-semantic", a, b)
	out, err := commandresult.Output(c, commandresult.RunOptions{
		Operation: "score semantic similarity",
		TimedOut:  func() bool { return ctx.Err() == context.DeadlineExceeded },
	})
	if err != nil {
		return 0, false, err
	}
	score, perr := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if perr != nil || score < 0 || score > 1 {
		return 0, false, nil
	}
	return score, true, nil
}
