package commandresult

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"syscall"
)

const diagnosticTailBytes = 4096

// Diagnostic is the stable, bounded account of an external command failure.
// It deliberately excludes argv and environment: both commonly carry tokens.
type Diagnostic struct {
	Kind       string `json:"kind"`
	Operation  string `json:"operation"`
	Executable string `json:"executable"`
	ExitCode   *int   `json:"exit_code,omitempty"`
	Signal     string `json:"signal,omitempty"`
	Timeout    bool   `json:"timeout"`
	StdoutTail string `json:"stdout_tail,omitempty"`
	StderrTail string `json:"stderr_tail,omitempty"`
	CwdScope   string `json:"cwd_scope"`
	Retryable  bool   `json:"retryable"`
	NextAction string `json:"next_action"`
}

// ExternalError retains the process error as its cause while carrying facts
// callers can inspect without parsing presentation text.
type ExternalError struct {
	Diagnostic Diagnostic
	cause      error
}

func (e *ExternalError) Error() string {
	detail := actionableLine(e.Diagnostic.StderrTail)
	if detail == "" {
		detail = actionableLine(e.Diagnostic.StdoutTail)
	}
	if detail == "" {
		detail = "external command failed"
	}
	state := "command failure"
	if e.Diagnostic.Timeout {
		state = "timeout"
	} else if e.Diagnostic.Signal != "" {
		state = "signal " + e.Diagnostic.Signal
	} else if e.Diagnostic.ExitCode != nil {
		state = fmt.Sprintf("exit %d", *e.Diagnostic.ExitCode)
	}
	return fmt.Sprintf("%s: %s (%s; next: %s)", e.Diagnostic.Operation, detail, state, e.Diagnostic.NextAction)
}

func (e *ExternalError) Unwrap() error { return e.cause }

// AsDiagnostic returns a copy of the typed diagnostic retained anywhere in an
// error chain. Wrapping with %w therefore preserves both the cause and facts.
func AsDiagnostic(err error) (Diagnostic, bool) {
	var external *ExternalError
	if !errors.As(err, &external) {
		return Diagnostic{}, false
	}
	return external.Diagnostic, true
}

// RunOptions describes only safe command metadata. Operation should be a
// verb-level label ("git status"), never an unsanitized command line.
type RunOptions struct {
	Operation     string
	WorkspaceRoot string
	TimedOut      func() bool
}

// Run executes cmd and turns a failure into an ExternalError. Successful output
// remains compatible with CombinedOutput callers; failed output is bounded and
// sanitized because legacy callers may interpolate it. stdout and stderr stay
// distinct in the typed failure.
func Run(cmd *exec.Cmd, opts RunOptions) ([]byte, error) {
	stdout, stderr, err := output(cmd)
	out := append(append([]byte(nil), stdout...), stderr...)
	if err == nil {
		return out, nil
	}
	timedOut := opts.TimedOut != nil && opts.TimedOut()
	// Legacy wrappers often interpolate the returned output alongside err. Make
	// that compatibility surface obey the same bounded disclosure policy too;
	// otherwise a typed safe error could sit next to a raw leaked token.
	safeOut := boundedTail(Redact(string(out), opts.WorkspaceRoot))
	return []byte(safeOut), NewExternalError(cmd, opts, stdout, stderr, err, timedOut)
}

// Output is the typed-diagnostic equivalent of exec.Cmd.Output: successful
// callers receive stdout alone, while a failure retains separately bounded
// and redacted stdout/stderr facts. Probe wrappers use this instead of Run
// because stderr is evidence, never part of the value they parse (issue #876).
func Output(cmd *exec.Cmd, opts RunOptions) ([]byte, error) {
	stdout, stderr, err := output(cmd)
	if err == nil {
		return stdout, nil
	}
	timedOut := opts.TimedOut != nil && opts.TimedOut()
	return nil, NewExternalError(cmd, opts, stdout, stderr, err, timedOut)
}

func output(cmd *exec.Cmd) (stdoutBytes, stderrBytes []byte, err error) {
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

// NewExternalError classifies a failed command that was executed by a wrapper
// with custom stream handling.
func NewExternalError(cmd *exec.Cmd, opts RunOptions, stdout, stderr []byte, cause error, timedOut bool) error {
	executable := filepath.Base(cmd.Path)
	operation := strings.TrimSpace(Redact(opts.Operation, opts.WorkspaceRoot))
	if operation == "" {
		operation = executable
	}
	d := Diagnostic{
		Kind:       "command_failure",
		Operation:  operation,
		Executable: executable,
		Timeout:    timedOut,
		StdoutTail: boundedTail(Redact(string(stdout), opts.WorkspaceRoot)),
		StderrTail: boundedTail(Redact(string(stderr), opts.WorkspaceRoot)),
		CwdScope:   cwdScope(cmd.Dir, opts.WorkspaceRoot),
	}
	if timedOut {
		d.Kind, d.Retryable = "timeout", true
		d.NextAction = "check the external service or child process, then retry"
	} else {
		var exit interface{ ExitCode() int }
		if errors.As(cause, &exit) {
			code := exit.ExitCode()
			d.ExitCode = &code
			var processExit *exec.ExitError
			if errors.As(cause, &processExit) {
				status, ok := processExit.Sys().(syscall.WaitStatus)
				if ok && status.Signaled() {
					d.Kind = "signal"
					d.Signal = strings.ToUpper(status.Signal().String())
					d.Retryable = true
					d.NextAction = "inspect why the process was terminated, then retry if safe"
				}
			}
		}
	}
	if d.NextAction == "" {
		combined := strings.ToLower(d.StdoutTail + "\n" + d.StderrTail)
		switch {
		case isMissingExecutable(cause):
			d.Kind = "missing_executable"
			d.NextAction = "install the executable or correct PATH"
		case strings.Contains(combined, "index.lock") || strings.Contains(combined, "another git process"):
			d.Kind, d.Retryable = "contention", true
			d.NextAction = "wait for the other process; remove a stale lock only after confirming no process owns it"
		case strings.Contains(combined, "authentication") || strings.Contains(combined, "not logged") || strings.Contains(combined, "bad credentials"):
			d.Kind = "authentication"
			d.NextAction = "authenticate the executable, then retry"
		default:
			d.NextAction = "inspect the retained stderr/stdout tail and correct the command condition"
		}
	}
	return &ExternalError{Diagnostic: d, cause: cause}
}

type recordedExitError struct{ code int }

func (e recordedExitError) Error() string { return fmt.Sprintf("exit status %d", e.code) }
func (e recordedExitError) ExitCode() int { return e.code }

// NewRecordedExitError reconstructs a typed failure from a durable exit marker
// after the original process is no longer available to inspect. Its cause still
// exposes ExitCode(), while the shared diagnostic policy classifies and redacts
// the retained output exactly like a live command failure.
func NewRecordedExitError(cmd *exec.Cmd, opts RunOptions, stdout, stderr []byte, exitCode int) error {
	return NewExternalError(cmd, opts, stdout, stderr, recordedExitError{code: exitCode}, false)
}

func isMissingExecutable(err error) bool {
	var execErr *exec.Error
	var pathErr *os.PathError
	return errors.As(err, &execErr) || (errors.As(err, &pathErr) && errors.Is(pathErr, os.ErrNotExist))
}

func boundedTail(s string) string {
	s = strings.TrimSpace(s)
	if len(s) <= diagnosticTailBytes {
		return s
	}
	return "[truncated] " + s[len(s)-diagnosticTailBytes:]
}

func actionableLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if line := strings.TrimSpace(lines[i]); line != "" {
			return line
		}
	}
	return ""
}

func cwdScope(cwd, root string) string {
	if cwd == "" || root == "" {
		return "<undisclosed>"
	}
	absCWD, cwdErr := filepath.Abs(cwd)
	absRoot, rootErr := filepath.Abs(root)
	if cwdErr != nil || rootErr != nil {
		return "<undisclosed>"
	}
	rel, err := filepath.Rel(absRoot, absCWD)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "<outside-workspace>"
	}
	if rel == "." {
		return "."
	}
	return "<workspace>/" + filepath.ToSlash(rel)
}

var (
	unixPathPattern = regexp.MustCompile(`(^|[ =(:])(/[[:alnum:]_.~@%+,:=-]+(?:/[[:alnum:]_.~@%+,:=-]+)*)`)
	urlSecret       = regexp.MustCompile(`(?i)(https?://)[^/@\s]+@`)
	tokenPattern    = regexp.MustCompile(`(?i)\b(?:gh[pousr]_[A-Za-z0-9_]{8,}|github_pat_[A-Za-z0-9_]{8,}|bearer\s+[A-Za-z0-9._~+/=-]{8,})\b`)
)

// Redact applies the shared disclosure policy used by subprocess wrappers.
// Secret-valued environment variables, recognizable tokens, URL credentials,
// and absolute paths outside workspaceRoot never survive it.
func Redact(value, workspaceRoot string) string {
	redacted := value
	for _, entry := range os.Environ() {
		name, secret, ok := strings.Cut(entry, "=")
		if !ok || len(secret) < 4 || !secretName(name) {
			continue
		}
		redacted = strings.ReplaceAll(redacted, secret, "<redacted>")
	}
	redacted = tokenPattern.ReplaceAllString(redacted, "<redacted>")
	redacted = urlSecret.ReplaceAllString(redacted, "${1}<redacted>@")
	if runtime.GOOS != "windows" {
		redacted = unixPathPattern.ReplaceAllStringFunc(redacted, func(match string) string {
			prefix := ""
			path := match
			if match[0] != '/' {
				prefix, path = match[:1], match[1:]
			}
			return prefix + disclosedPath(path, workspaceRoot)
		})
	}
	return redacted
}

func secretName(name string) bool {
	name = strings.ToUpper(name)
	for _, marker := range []string{"TOKEN", "SECRET", "PASSWORD", "PASSWD", "API_KEY", "APIKEY", "CREDENTIAL", "AUTHORIZATION"} {
		if strings.Contains(name, marker) {
			return true
		}
	}
	return false
}

func disclosedPath(path, root string) string {
	if root == "" {
		return "<outside-workspace>"
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "<outside-workspace>"
	}
	rel, err := filepath.Rel(absRoot, filepath.Clean(path))
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "<outside-workspace>"
	}
	if rel == "." {
		return "<workspace>"
	}
	return "<workspace>/" + filepath.ToSlash(rel)
}
