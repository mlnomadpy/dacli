package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeGH puts a scripted `gh` first on PATH: creates are logged with bodies
// saved per issue, marker search greps those bodies, and visibility flips
// via a state file. Zero network — and the log is the assertion surface.
func fakeGH(t *testing.T, dir string) (stateDir string) {
	t.Helper()
	stateDir = filepath.Join(dir, "ghstate")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	binDir := filepath.Join(dir, "ghbin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	script := `#!/bin/sh
S="$GH_STATE"
grab() { # grab <flag> "$@" -> value after flag
  want="$1"; shift; prev=""
  for a in "$@"; do [ "$prev" = "$want" ] && { echo "$a"; return; }; prev="$a"; done
}
case "$1 $2" in
"auth status") echo "Logged in to github.com"; exit 0;;
"repo view")
  vis=PRIVATE; [ -f "$S/public" ] && vis=PUBLIC
  echo "{\"nameWithOwner\":\"me/demo\",\"visibility\":\"$vis\"}";;
"issue create")
  n=$(( $(cat "$S/n" 2>/dev/null || echo 0) + 1 )); echo "$n" > "$S/n"
  grab --body "$@" > "$S/issue_$n.body"
  echo "create $n" >> "$S/log"
  echo "https://github.com/me/demo/issues/$n";;
"issue list")
  q=$(grab --search "$@")
  out="["; sep=""
  for f in "$S"/issue_*.body; do
    [ -f "$f" ] || continue
    if grep -qF "$q" "$f"; then
      num=${f##*issue_}; num=${num%.body}
      out="$out$sep{\"number\":$num}"; sep=","
    fi
  done
  echo "$out]";;
"issue close")
  echo "close $3" >> "$S/log";;
*) echo "fake gh: unhandled $*" >&2; exit 1;;
esac
`
	if err := os.WriteFile(filepath.Join(binDir, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GH_STATE", stateDir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return stateDir
}

func countLines(t *testing.T, path, prefix string) int {
	raw, _ := os.ReadFile(path)
	n := 0
	for _, l := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(l, prefix) {
			n++
		}
	}
	return n
}

// Acceptance 1: an interrupted sync re-run converges with ZERO duplicate
// issues — the characteristic failure of naive syncers.
func TestGithubPushIdempotent(t *testing.T) {
	dir := t.TempDir()
	state := fakeGH(t, dir)
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "P", "--slug", "p", "--goal", "g")
	run(t, dir, 0, "task", "add", "Audit the writers", "--project", "p", "--accept", "a")
	run(t, dir, 0, "task", "add", "Ship the shim", "--project", "p", "--accept", "b")

	run(t, dir, 0, "github", "doctor")
	run(t, dir, 0, "github", "link", "p")

	out := run(t, dir, 0, "github", "push", "p")
	if !strings.Contains(out, "2 created, 0 adopted") {
		t.Fatalf("first push wrong:\n%s", out)
	}
	// Plain re-run: mappings hit, nothing created.
	out = run(t, dir, 0, "github", "push", "p")
	if !strings.Contains(out, "0 created, 0 adopted-by-marker, 2 unchanged") {
		t.Errorf("second push not a no-op:\n%s", out)
	}

	// The interruption: the remote issue exists but the local mapping write
	// never landed. Strip the mapping and push again — the marker search
	// must ADOPT, never duplicate.
	w, _, err := openWorkspace(&Ctx{Cwd: dir, Stdout: os.Stdout, Stderr: os.Stderr})
	if err != nil {
		t.Fatal(err)
	}
	_ = w
	tk := findTaskDoc(t, dir, "001")
	tk.Doc.Front.Delete("github")
	if err := saveTask(tk); err != nil {
		t.Fatal(err)
	}

	out = run(t, dir, 0, "github", "push", "p")
	if !strings.Contains(out, "0 created, 1 adopted-by-marker") {
		t.Errorf("crash recovery did not adopt:\n%s", out)
	}
	if got := countLines(t, filepath.Join(state, "log"), "create"); got != 2 {
		t.Errorf("%d creates across three pushes for two tasks — duplicates!", got)
	}

	// Done tasks get their issues closed on push.
	run(t, dir, 0, "task", "claim", "002")
	run(t, dir, 0, "task", "check", "002", "--all")
	run(t, dir, 0, "task", "done", "002")
	run(t, dir, 0, "github", "push", "p")
	if got := countLines(t, filepath.Join(state, "log"), "close"); got != 1 {
		t.Errorf("done task not closed (%d closes)", got)
	}
}

// Acceptance 2: a public repo requires RECORDED per-project consent, checked
// live at every push — a repo flipped public after linking re-trips the gate.
func TestGithubPublicDisclosureGate(t *testing.T) {
	dir := t.TempDir()
	state := fakeGH(t, dir)
	run(t, dir, 0, "init", "--name", "x")
	run(t, dir, 0, "project", "add", "P", "--slug", "p", "--goal", "g")
	run(t, dir, 0, "task", "add", "T one", "--project", "p", "--accept", "a")

	// Public from the start: link refuses without the flag.
	if err := os.WriteFile(filepath.Join(state, "public"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	refusal := run(t, dir, 3, "github", "link", "p")
	if !strings.Contains(refusal, "disclosure event") || !strings.Contains(refusal, "--allow-public") {
		t.Fatalf("public link refusal wrong:\n%s", refusal)
	}
	run(t, dir, 0, "github", "link", "p", "--allow-public")
	run(t, dir, 0, "github", "push", "p")

	// A second project on the same repo has NO recorded consent: per-project
	// means per-project.
	run(t, dir, 0, "project", "add", "Q", "--slug", "q", "--goal", "g")
	run(t, dir, 3, "github", "link", "q")

	// Private at link time, flipped public later: push re-trips the gate.
	if err := os.Remove(filepath.Join(state, "public")); err != nil {
		t.Fatal(err)
	}
	run(t, dir, 0, "github", "link", "q")
	if err := os.WriteFile(filepath.Join(state, "public"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	refusal = run(t, dir, 3, "github", "push", "q")
	if !strings.Contains(refusal, "no recorded consent") {
		t.Errorf("visibility flip not re-gated:\n%s", refusal)
	}
}
