package cli

// End-to-end arcs (dacli 237).
//
// The unit suites prove each command in isolation and `TestDogfoodLoop` proves
// one short session. Neither proves the ARCS a user actually runs, and every
// expensive bug in this codebase has lived in an integration seam rather than
// in a function: a task closed without the work existing, a record that lost
// its frontmatter, a flag silently dropped between two commands. A seam is only
// visible when the commands run in sequence, against the same workspace, with
// the same identity resolution the binary uses.
//
// So these tests drive the REAL command table through `run` (dogfood_test.go),
// which dispatches via match() exactly as Main does. Everything runs offline in
// t.TempDir(): no network, no `gh`, no spawned agent process, no sleeps.
//
// Assertions are on BEHAVIOR and exit codes wherever possible — a task's status
// after a sequence, a file on disk, a dependency edge in the store — so a
// reworded message does not break the suite. Where output text is matched, it
// is a stable substring and the comment says why that substring is load-bearing.

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/mlnomadpy/dacli/internal/agentid"
)

// --- helpers -----------------------------------------------------------

// e2eStdout runs a command capturing stdout SEPARATELY from stderr. `run`
// returns them combined, which is right for most assertions but wrong for the
// one command whose stdout is a value rather than a message: `agent spawn`
// prints the token to stdout alone precisely so `TOK=$(dacli agent spawn)`
// works, and a test that reads the combined stream would capture the human
// banner along with it.
func e2eStdout(t *testing.T, dir string, wantCode int, args ...string) string {
	t.Helper()
	var out, errb bytes.Buffer
	ctx := &Ctx{Stdout: &out, Stderr: &errb, Cwd: dir}
	cmd, rest := match(args)
	if cmd == nil {
		t.Fatalf("no such command: %v", args)
	}
	err := cmd.Run(ctx, rest)
	if got := exitCode(err); got != wantCode {
		t.Fatalf("%v: exit %d, want %d (err: %v)\nstdout: %s\nstderr: %s",
			args, got, wantCode, err, out.String(), errb.String())
	}
	return out.String()
}

// e2eSpawnRO mints a read-only child and returns its token, which the caller
// binds with t.Setenv(agentid.EnvVar, tok). Binding through the environment is
// not a shortcut: it is exactly how a real child receives its identity, so the
// whole grant path below is the production one and not a test-only
// impersonation.
func e2eSpawnRO(t *testing.T, dir, role string) string {
	t.Helper()
	tok := strings.TrimSpace(e2eStdout(t, dir, 0, "agent", "spawn", "--grant", "ro", "--role", role))
	if tok == "" {
		t.Fatal("agent spawn printed no token on stdout")
	}
	return tok
}

// e2eGitRepo is a temp dir that is a real git repository. Adoption reads a
// working tree, and several downstream commands assume a repo, so the arc that
// needs one gets one — skipped rather than failed where git is absent, the same
// contract orchestration's loopEnv uses.
func e2eGitRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "e2e@example.invalid"},
		{"config", "user.name", "e2e"},
	} {
		c := exec.Command("git", args...)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return dir
}

func e2eWriteFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// e2eGreenfield is the shared setup for the arcs that start from nothing: one
// `dacli new` with an explicit stack, so no arc depends on what the temp
// directory happens to contain.
func e2eGreenfield(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run(t, dir, 0, "new", "Ledger Service",
		"--goal", "Record every balance transfer through one auditable write path",
		"--slug", "ledger", "--stack", "go", "--template", "standard")
	return dir
}

// --- arc 1: greenfield --------------------------------------------------

// new → seeded backlog → next → context → stage, plus the CI workflow on disk.
// This is the path a first-time user takes, and each step consumes what the
// previous one wrote: the backlog's dependency edges decide what `next`
// recommends, the project sections `new` filled decide what the brief carries,
// and the workflow file decides what the CI rung of the backlog is allowed to
// claim.
func TestE2EGreenfieldArc(t *testing.T) {
	dir := e2eGreenfield(t)

	// The seeded backlog exists and is a DAG, not a flat list. Asserted over
	// the store rather than over `critical-path`'s columns: the edge is the
	// fact, the schedule is a rendering of it.
	first := findTaskDoc(t, dir, "001")
	if deps := first.Deps(); len(deps) != 0 {
		t.Errorf("the first rung must depend on nothing, got %v", deps)
	}
	second := findTaskDoc(t, dir, "002")
	depsOnFirst := false
	for _, d := range second.Deps() {
		if d.Ref == first.ID {
			depsOnFirst = true
		}
	}
	if !depsOnFirst {
		t.Errorf("rung 002 must depend on rung 001 (deps: %v)", second.Deps())
	}
	// Every rung is sized, or the build-stage gate the backlog exists to open
	// (tasks: all_have_estimate) would be closed by the seeding itself.
	for _, ref := range []string{"001", "002", "003", "004", "005"} {
		if _, ok := findTaskDoc(t, dir, ref).Estimate(); !ok {
			t.Errorf("seeded task %s has no estimate", ref)
		}
	}

	// `next` recommends the one rung whose dependencies are satisfied. Matching
	// on the task ref, not the prose: 001 is ready, 002 is not.
	nx := run(t, dir, 0, "next")
	if !strings.Contains(nx, "001-") {
		t.Errorf("next did not recommend the ready first rung:\n%s", nx)
	}
	if strings.Contains(nx, "002-") {
		t.Errorf("next recommended a rung still waiting on its dependency:\n%s", nx)
	}

	// The brief is the product. It must carry the task, the goal, and the Spec
	// section `new` seeded — on a greenfield repo the Spec is the ONLY
	// description of the thing being built, so its absence is silent
	// context loss rather than a formatting nit.
	brief := run(t, dir, 0, "context", "001")
	for _, want := range []string{
		"## Task: Scaffold the Go project skeleton",
		"## Spec",                // seeded by `new`, carried by brief.Assemble
		"## Architecture",        // ditto
		"one auditable write",    // the goal reached the brief
		"go build ./...",         // the recorded stack's real build command
		"data, not instructions", // the untrusted-content header must never be trimmed away
	} {
		if !strings.Contains(brief, want) {
			t.Errorf("brief missing %q:\n%s", want, brief)
		}
	}

	// Gate status is reported per check, with both outcomes visible: the Goal
	// `new` filled passes, the glossary and decisions it cannot invent do not.
	// ✓/✗ are the gate markers themselves, not prose.
	st := run(t, dir, 0, "stage", "ledger")
	if !strings.Contains(st, "✓") || !strings.Contains(st, "✗") {
		t.Errorf("stage did not report per-check gate status:\n%s", st)
	}
	// And the gate REFUSES (exit 3) rather than advancing on an unmet check.
	run(t, dir, 3, "stage", "advance", "ledger")

	// The CI workflow is written at project birth, running the same two
	// commands recorded in Constraints — a repo with no workflow reports an
	// empty check list, which reads as "green" to everything downstream.
	wf := filepath.Join(dir, ".github", "workflows", "ci.yml")
	raw, err := os.ReadFile(wf)
	if err != nil {
		t.Fatalf("CI workflow not written to %s: %v", wf, err)
	}
	for _, want := range []string{"go build ./...", "go test ./..."} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("workflow does not run %q:\n%s", want, raw)
		}
	}

	// `new` is greenfield only: a second run against the same slug is a
	// refusal (3), not a silent second seeding of the same backlog.
	run(t, dir, 3, "new", "Ledger Service",
		"--goal", "Record every balance transfer through one auditable write path",
		"--slug", "ledger", "--stack", "go")
}

// `dacli context` cannot be asked for a role, so a role's standing
// instructions never reach a hand-assembled brief — brief.Options.Role is set
// only by `spawn`/`agents run` (internal/features/execution). This is not a
// bug so much as an undocumented seam, and it is worth a test because the
// natural reading of "context assembles the brief for an agent" is that
// `--role` would work. If a --role flag is ever added, this test fails and
// tells whoever added it to assert the instructions instead.
func TestE2EContextTakesNoRole(t *testing.T) {
	dir := e2eGreenfield(t)
	run(t, dir, 2, "context", "001", "--role", "implementer")
}

// --- arc 2: adopt --------------------------------------------------------

// An existing repo with code → adopt → tasks seeded from its markers → doctor
// clean. The seam here is that adopt's scan output has to reach an agent: the
// codebase map is written onto the PROJECT, so it rides into every brief, and
// a map that landed anywhere else would be invisible without re-walking the
// tree.
func TestE2EAdoptArc(t *testing.T) {
	dir := e2eGitRepo(t)
	e2eWriteFile(t, dir, "README.md", "# Widget Service\n\nMakes widgets.\n")
	e2eWriteFile(t, dir, "src/writer.go",
		"package src\n\n// TODO: route the flush path through the service layer\nfunc Flush() {}\n")
	e2eWriteFile(t, dir, "src/reader.go", "package src\n\nfunc Read() {}\n")

	// --project is explicit: without it the slug is derived from the directory
	// name, which under t.TempDir() is a bare number.
	out := run(t, dir, 0, "adopt", "--project", "widget", "--todos")
	if !strings.Contains(out, "codebase map") {
		t.Errorf("adopt did not report a codebase map:\n%s", out)
	}

	// The marker became a real task, not a line in a report.
	tasks := run(t, dir, 0, "task", "list", "--project", "widget")
	if !strings.Contains(tasks, "flush path") {
		t.Errorf("TODO marker was not seeded as a task:\n%s", tasks)
	}

	// And the scan reaches the brief for that task — the whole point of writing
	// the map onto the project.
	brief := run(t, dir, 0, "context", "001")
	for _, want := range []string{"## Codebase map", "Go"} {
		if !strings.Contains(brief, want) {
			t.Errorf("brief missing %q after adopt:\n%s", want, brief)
		}
	}

	// A freshly adopted workspace is healthy: doctor exists to distinguish a
	// clean tree from a damaged one, so a false positive here would train
	// operators to ignore it.
	if d := run(t, dir, 0, "doctor"); !strings.Contains(d, "no anti-patterns") {
		t.Errorf("doctor flagged a freshly adopted repo:\n%s", d)
	}
}

// --- arc 3: task lifecycle ----------------------------------------------

// add → estimate → critical-path → team assign → claim → check → accept
// --verify → list. Every step reads what the previous one wrote; the sizing
// step in particular is what makes the two scheduling commands work at all,
// and before `task estimate` existed a backlog filed without one was
// permanently unschedulable.
func TestE2ETaskLifecycleArc(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "lifecycle")
	run(t, dir, 0, "project", "add", "Lifecycle", "--slug", "lc",
		"--goal", "Prove a task survives the whole add-to-accepted arc")
	run(t, dir, 0, "role", "add", "junior", "--kind", "implementer",
		"--model", "haiku", "--max-points", "3", "--grant", "rw")
	run(t, dir, 0, "role", "add", "senior", "--kind", "implementer",
		"--model", "opus", "--grant", "rw")

	run(t, dir, 0, "task", "add", "Extract the balance writer into a service",
		"--project", "lc", "--priority", "must",
		"--accept", "One write path into balances",
		"--accept", "A test fails when that path breaks")

	// Routing REFUSES an unsized task (3) rather than guessing a role: an
	// estimate is the input capacity routing runs on.
	run(t, dir, 3, "team", "assign", "001")

	run(t, dir, 0, "task", "estimate", "001", "--estimate", "2,5,14")
	if _, ok := findTaskDoc(t, dir, "001").Estimate(); !ok {
		t.Fatal("task estimate did not persist a three-point estimate")
	}

	// Now it schedules...
	cp := run(t, dir, 0, "critical-path", "--project", "lc")
	if !strings.Contains(cp, "001-") {
		t.Errorf("critical-path did not schedule the sized task:\n%s", cp)
	}

	// ...and routes. Te 6.0 exceeds junior's 3-point cap, so the only capable
	// implementer is senior. Matching the role NAME is the behavior under test:
	// picking the capped role would be the bug.
	as := run(t, dir, 0, "team", "assign", "001")
	if !strings.Contains(as, "senior") {
		t.Errorf("assign did not route to the only role whose capacity covers Te 6.0:\n%s", as)
	}
	if strings.Contains(as, "→ junior") {
		t.Errorf("assign routed to a role whose cap is below the task's Te:\n%s", as)
	}

	run(t, dir, 0, "task", "claim", "001")
	if got := findTaskDoc(t, dir, "001").Status; string(got) != "active" {
		t.Errorf("claim left status %q, want active", got)
	}

	run(t, dir, 0, "task", "check", "001", "--all")

	// accept --verify closes it AND records what certified the close. `sh -c`
	// is the verify runner, so this half is unix-only.
	if runtime.GOOS == "windows" {
		t.Skip("accept --verify shells out through sh -c")
	}
	run(t, dir, 0, "accept", "001", "--verify", "exit 0")

	closed := findTaskDoc(t, dir, "001")
	if string(closed.Status) != "done" {
		t.Fatalf("accept did not close the task (status %q)", closed.Status)
	}
	// The evidence line is the anti-"closed without work existing" guard: a
	// close whose evidence is absent used to be indistinguishable from a
	// verified one. "verified by" is the stable half of that record.
	logSec, _ := closed.Doc.Section("Log")
	if !strings.Contains(logSec.Content, "verified by") {
		t.Errorf("accept --verify recorded no verification evidence:\n%s", logSec.Content)
	}

	// `task list --status` must actually filter. A dropped flag here lists
	// EVERY task, which reads as a correct answer to a question nobody asked —
	// exactly the silently-dropped-flag class this suite exists to catch.
	run(t, dir, 0, "task", "add", "A second, still-open task", "--project", "lc",
		"--accept", "it is open")
	doneOnly := run(t, dir, 0, "task", "list", "--status", "done")
	if !strings.Contains(doneOnly, "001-") {
		t.Errorf("--status done omitted the closed task:\n%s", doneOnly)
	}
	if strings.Contains(doneOnly, "002-") {
		t.Errorf("--status done was dropped: an open task appeared:\n%s", doneOnly)
	}
}

// A failing verification must leave the task OPEN. Reported operationally
// (exit 1), because the check ran and the work did not pass it — that is a
// result, not a policy refusal.
func TestE2EFailedVerificationDoesNotClose(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("accept --verify shells out through sh -c")
	}
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "v")
	run(t, dir, 0, "project", "add", "V", "--slug", "v", "--goal", "Prove a failed check keeps a task open")
	run(t, dir, 0, "task", "add", "Work whose check fails", "--project", "v", "--accept", "it works")

	run(t, dir, 1, "accept", "001", "--verify", "exit 7")
	if got := findTaskDoc(t, dir, "001").Status; string(got) == "done" {
		t.Fatal("a task closed despite its verification failing")
	}
}

// accept WITHOUT --verify still closes — the common operator flow — but the
// record must say so. An unverified close that looks identical to a verified
// one is how "done" became an unverified assertion.
func TestE2EAcceptWithoutVerifyRecordsItUnverified(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "u")
	run(t, dir, 0, "project", "add", "U", "--slug", "u", "--goal", "Prove an unverified close is labelled as one")
	run(t, dir, 0, "task", "add", "Work nobody checked", "--project", "u", "--accept", "it works")

	run(t, dir, 0, "accept", "001")

	closed := findTaskDoc(t, dir, "001")
	if string(closed.Status) != "done" {
		t.Fatalf("accept did not close the task (status %q)", closed.Status)
	}
	logSec, _ := closed.Doc.Section("Log")
	// "WITHOUT verification" is load-bearing text: it is the record that
	// distinguishes the two closes, so its wording IS the contract.
	if !strings.Contains(logSec.Content, "WITHOUT verification") {
		t.Errorf("an unverified close was not labelled as unverified:\n%s", logSec.Content)
	}
	if strings.Contains(logSec.Content, "verified by") {
		t.Errorf("an unverified close claims evidence:\n%s", logSec.Content)
	}

	// --require-verify makes the unverified close impossible rather than
	// merely visible: a refusal (3), never a retry.
	run(t, dir, 0, "task", "add", "Second piece of work", "--project", "u", "--accept", "it works")
	run(t, dir, 3, "accept", "002", "--require-verify")
	if got := findTaskDoc(t, dir, "002").Status; string(got) == "done" {
		t.Fatal("--require-verify closed a task with no verification command")
	}
}

// --- arc 4: ownership and grants ----------------------------------------

// The cooperative model's core contract, driven end to end with a real spawned
// identity bound through DACLI_AGENT:
//
//	a read-only child READS and FILES findings freely;
//	its MUTATIONS become proposals, never writes;
//	the root's `sync` applies what it owns;
//	`accept --force` reconciles a task orphaned by the (now finished) child.
//
// Nothing here is simulated: the token comes from `agent spawn`, identity is
// resolved from the process environment, and every refusal is the production
// path an out-of-process child would take.
func TestE2EOwnershipGrantArc(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "grants")
	run(t, dir, 0, "project", "add", "Grants", "--slug", "g",
		"--goal", "Prove the cooperative grant model holds across a whole session")
	run(t, dir, 0, "task", "add", "Audit the write paths into balances",
		"--project", "g", "--priority", "must", "--estimate", "2,5,14",
		"--accept", "Writers of balances listed with file:line")

	tok := e2eSpawnRO(t, dir, "auditor")
	t.Setenv(agentid.EnvVar, tok)

	// Identity resolves to the child, attenuated to ro.
	who := run(t, dir, 0, "whoami")
	if !strings.Contains(who, "ro") || strings.Contains(who, agentid.RootID) {
		t.Fatalf("the spawned identity did not bind read-only:\n%s", who)
	}

	// A read-only agent is not mute: it reads...
	if b := run(t, dir, 0, "context", "001"); !strings.Contains(b, "## Task:") {
		t.Errorf("a read-only agent could not read a brief:\n%s", b)
	}
	// ...and files findings, which are visible to every reader immediately
	// (they land as pending events, and briefs fold pending events in).
	run(t, dir, 0, "note", "add", "finding", "Batch job writes balances directly",
		"--project", "g", "--about", "001", "--severity", "major",
		"--body", "cron/settle_batch.go:112 bypasses the service layer entirely.")
	if b := run(t, dir, 0, "context", "001"); !strings.Contains(b, "bypasses the service layer") {
		t.Errorf("a finding filed by a read-only agent did not reach the next brief:\n%s", b)
	}

	// Its mutations become PROPOSALS, not writes. Claim and accept are exit 0
	// (the proposal was recorded successfully); the box-check, which has no
	// proposal path, is a refusal (3).
	run(t, dir, 0, "task", "claim", "001")
	run(t, dir, 3, "task", "check", "001", "--all")
	run(t, dir, 0, "accept", "001")

	// Nothing the child did rewrote the task.
	beforeSync := findTaskDoc(t, dir, "001")
	if beforeSync.Owner() == "" || beforeSync.Owner() != agentid.RootID {
		t.Errorf("a read-only claim rewrote ownership before sync (owner %q)", beforeSync.Owner())
	}
	for _, box := range beforeSync.Acceptance() {
		if box.Done {
			t.Error("a read-only agent checked an acceptance box")
		}
	}

	// Back to root: sync applies the events it owns.
	_ = os.Unsetenv(agentid.EnvVar)
	sy := run(t, dir, 0, "sync")
	if !strings.Contains(sy, "applied") {
		t.Errorf("sync applied nothing:\n%s", sy)
	}
	synced := findTaskDoc(t, dir, "001")
	if synced.Owner() == agentid.RootID || synced.Owner() == "" {
		t.Fatalf("sync did not transfer ownership to the claiming child (owner %q)", synced.Owner())
	}
	if string(synced.Status) != "active" {
		t.Errorf("sync did not activate the claimed task (status %q)", synced.Status)
	}

	// Root is now NOT the owner, so a plain accept proposes rather than
	// closing — peer concurrency safety, preserved even for root.
	run(t, dir, 0, "accept", "001")
	if got := findTaskDoc(t, dir, "001").Status; string(got) == "done" {
		t.Fatal("root closed a task it does not own without --force")
	}

	// --force is the explicit operator override: the child has finished and
	// will never sync again, so without it the backlog orphan-locks. Doctor
	// names that condition before the override is used.
	if d := run(t, dir, 0, "doctor"); !strings.Contains(d, "orphaned-task") {
		t.Errorf("doctor did not flag the task orphaned behind a finished agent:\n%s", d)
	}
	run(t, dir, 0, "accept", "001", "--force")

	closed := findTaskDoc(t, dir, "001")
	if string(closed.Status) != "done" {
		t.Fatalf("accept --force did not reconcile the orphaned task (status %q)", closed.Status)
	}
	if closed.Owner() != agentid.RootID {
		t.Errorf("accept --force did not adopt the task (owner %q)", closed.Owner())
	}
	for _, box := range closed.Acceptance() {
		if !box.Done {
			t.Errorf("accept --force closed the task with an unchecked box: %s", box.Text)
		}
	}
}

// Attenuation is monotonic: a read-only agent's whole subtree is read-only,
// so it cannot spawn itself an rw child and launder its way to a write.
func TestE2EReadOnlyCannotWidenItsGrant(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "atten")
	tok := e2eSpawnRO(t, dir, "auditor")
	t.Setenv(agentid.EnvVar, tok)
	run(t, dir, 3, "agent", "spawn", "--grant", "rw")
}

// --- arc 5: knowledge ----------------------------------------------------

// A decision recorded once must reach every later brief in that project, with
// the rejected alternative and the reason attached. That is the single
// highest-value guarantee in the tool: without it, agent N+1 re-litigates the
// choice agent N already made and paid for.
func TestE2EDecisionReachesALaterBrief(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "knowledge")
	run(t, dir, 0, "project", "add", "Ledger", "--slug", "ledger",
		"--goal", "One write path into balances, reconciliation-clean")
	run(t, dir, 0, "task", "add", "Audit the write paths into balances",
		"--project", "ledger", "--accept", "Writers listed with file:line")

	// A decision with no rejection is refused: it cannot be safely revisited,
	// which is the only thing a recorded decision is for.
	run(t, dir, 1, "note", "add", "decision", "Half a decision", "--project", "ledger")

	run(t, dir, 0, "note", "add", "decision", "Ledger writes stay synchronous",
		"--project", "ledger",
		"--rejected", "async queue with eventual reconciliation",
		"--because", "reconciliation cost exceeds the latency win")

	// The task the decision must reach is created AFTER it — the whole point
	// is that later work inherits earlier reasoning without being told.
	run(t, dir, 0, "task", "add", "Add the reconciliation report",
		"--project", "ledger", "--accept", "Report lists every unmatched entry")

	brief := run(t, dir, 0, "context", "002")
	for _, want := range []string{
		"Ledger writes stay synchronous",              // the choice
		"Rejected: async queue",                       // the road not taken
		"reconciliation cost exceeds the latency win", // why
	} {
		if !strings.Contains(brief, want) {
			t.Errorf("decision did not reach a later brief — missing %q:\n%s", want, brief)
		}
	}

	// Scope is per project: a decision filed under one project must not leak
	// into another's brief, or every brief eventually carries every decision.
	run(t, dir, 0, "project", "add", "Unrelated", "--slug", "other", "--goal", "A different piece of work entirely")
	run(t, dir, 0, "task", "add", "Something else entirely", "--project", "other", "--accept", "it works")
	// Referenced by slug, not by "003": task seq is per PROJECT, so the second
	// project's first task is also 001 (see the ambiguity test below).
	if b := run(t, dir, 0, "context", "something-else-entirely"); strings.Contains(b, "async queue") {
		t.Errorf("a decision leaked across the project boundary:\n%s", b)
	}
}

// Task sequence numbers are per PROJECT, but every ref resolves across the
// WHOLE workspace — so the moment a second project has a task, the "001" that
// `task add`, `next` and the seeded next-steps block all print is ambiguous.
// The resolution is at least safe: it refuses rather than guessing.
//
// SUSPECTED SEAM, not fixed here (dacli 237): the refusal arrives as a GENERIC
// error, exit 1, not a usage error (2) or a refusal (3). An agent branching on
// the exit-code contract cannot tell "your ref was ambiguous, disambiguate it"
// from "something broke", so the correct response — re-issue with --project or
// the slug — is exactly the retry the contract tells it not to attempt on a 3
// and gives it no signal to attempt on a 1. Documented as the ACTUAL behavior.
func TestE2ECrossProjectRefIsAmbiguousNotGuessed(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "refs")
	run(t, dir, 0, "project", "add", "Alpha", "--slug", "alpha", "--goal", "The first of two projects in one workspace")
	run(t, dir, 0, "project", "add", "Beta", "--slug", "beta", "--goal", "The second of two projects in one workspace")
	run(t, dir, 0, "task", "add", "Work filed under alpha", "--project", "alpha", "--accept", "it works")
	run(t, dir, 0, "task", "add", "Work filed under beta", "--project", "beta", "--accept", "it works")

	// Both tasks are 001. The ref must NOT silently resolve to one of them.
	out := run(t, dir, 1, "context", "001")
	if !strings.Contains(out, "ambiguous") {
		t.Errorf("an ambiguous ref must say so rather than guess:\n%s", out)
	}
	// Both candidates are named, so the caller can disambiguate without a
	// second command.
	for _, want := range []string{"alpha/001", "beta/001"} {
		if !strings.Contains(out, want) {
			t.Errorf("the ambiguity error does not name %q:\n%s", want, out)
		}
	}
	// The slug form is the working disambiguator, and it still resolves.
	run(t, dir, 0, "context", "work-filed-under-alpha")
}

// --- arc 6: failures -----------------------------------------------------

// The exit-code contract (2 usage, 3 refusal, 4 not found, 1 everything else)
// must hold across a REAL sequence, not just per command in isolation. Agents
// branch on these without parsing stderr, and a 3 must never be retried — so a
// refusal that leaked out as a 1 would send a supervisor into a retry loop.
// The sequence also proves the workspace is not wedged by any of them: a
// working command still exits 0 afterwards.
func TestE2EExitCodesHoldAcrossASequence(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, 0, "init", "--name", "codes")
	run(t, dir, 0, "project", "add", "Codes", "--slug", "c",
		"--goal", "Prove the exit-code contract across a whole session")
	run(t, dir, 0, "task", "add", "Real work with real acceptance",
		"--project", "c", "--accept", "the work is done")

	// 2 — the caller's mistake: a missing required flag, and a typo'd one.
	run(t, dir, 2, "task", "add", "No project given")
	run(t, dir, 2, "task", "list", "--projct", "c")
	run(t, dir, 2, "context")

	// 4 — not found, distinct from a generic failure.
	run(t, dir, 4, "task", "show", "no-such-task")
	run(t, dir, 4, "context", "no-such-task")
	run(t, dir, 4, "accept", "no-such-task")

	// 3 — refused by policy. Unchecked acceptance is an ANSWER; the message
	// says so, and "do not retry" is the load-bearing half of it.
	refusal := run(t, dir, 3, "task", "done", "001")
	if !strings.Contains(refusal, "do not retry") {
		t.Errorf("the refusal does not tell a supervisor not to retry:\n%s", refusal)
	}
	if got := findTaskDoc(t, dir, "001").Status; string(got) == "done" {
		t.Fatal("task done closed a task with unmet acceptance")
	}
	// A destructive command refuses without its confirmation flag.
	run(t, dir, 3, "project", "rm", "c")

	// 0 — none of the above wedged anything, and the refused close still works
	// once the criterion is actually met.
	run(t, dir, 0, "task", "check", "001", "--all")
	run(t, dir, 0, "task", "done", "001")
	if got := findTaskDoc(t, dir, "001").Status; string(got) != "done" {
		t.Fatalf("task did not close after its acceptance was met (status %q)", got)
	}
	run(t, dir, 0, "status")
}

// A task file that lost its frontmatter still LISTS — status comes from the
// folder, seq and slug from the filename — so it appears as a hollow row and
// every list path carries on as if the workspace were healthy. doctor is the
// only thing that looks, so it must not call this clean. Driven here over a
// workspace built by the real onboarding arc rather than a hand-assembled one,
// because that is where the damage actually lands.
func TestE2ECorruptTaskFileIsReportedByDoctor(t *testing.T) {
	dir := e2eGreenfield(t)
	if d := run(t, dir, 0, "doctor"); !strings.Contains(d, "no anti-patterns") {
		t.Fatalf("a freshly seeded workspace should be clean:\n%s", d)
	}

	victim := findTaskDoc(t, dir, "003").Path
	if err := os.WriteFile(victim, []byte("this file lost its frontmatter\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	d := run(t, dir, 0, "doctor")
	if strings.Contains(d, "no anti-patterns") {
		t.Fatalf("doctor called a workspace with a destroyed task file healthy:\n%s", d)
	}
	// "corrupt-object" is the machine-facing pattern name doctor reports; the
	// detail line around it is free to be reworded.
	if !strings.Contains(d, "corrupt-object") {
		t.Errorf("doctor did not name the integrity failure:\n%s", d)
	}
}
