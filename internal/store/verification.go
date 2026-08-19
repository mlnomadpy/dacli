package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/gitx"
)

const verificationSection = "Verification Evidence"

// VerificationEvidence is the machine-readable provenance of one command
// verification. Legacy is populated only for pre-schema rendered Log lines;
// their unavailable provenance fields deliberately remain zero-valued.
type VerificationEvidence struct {
	Command      string `json:"command"`
	ExitCode     int    `json:"exit_code"`
	DurationMS   int64  `json:"duration_ms"`
	ArtifactHash string `json:"artifact_hash"`
	Verifier     string `json:"verifier"`
	Branch       string `json:"branch"`
	CommitSHA    string `json:"commit_sha"`
	Legacy       string `json:"legacy,omitempty"`
}

// RunVerification captures the command's combined-output digest and execution
// provenance even when it fails. Callers decide whether failed evidence should
// be persisted alongside a task transition.
func RunVerification(dir, verifier, command string) (VerificationEvidence, []byte, error) {
	ev := VerificationEvidence{Command: command, Verifier: verifier, ExitCode: 0}
	if gitx.Available() {
		if out, err := gitx.Run(dir, "branch", "--show-current"); err == nil {
			ev.Branch = out
		}
		if out, err := gitx.Run(dir, "rev-parse", "HEAD"); err == nil {
			ev.CommitSHA = strings.TrimSpace(out)
		}
	}
	started := time.Now()
	c := exec.Command("sh", "-c", command)
	c.Dir = dir
	out, err := c.CombinedOutput()
	ev.DurationMS = time.Since(started).Milliseconds()
	sum := sha256.Sum256(out)
	ev.ArtifactHash = "sha256:" + hex.EncodeToString(sum[:])
	if err != nil {
		ev.ExitCode = 1
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			ev.ExitCode = ee.ExitCode()
		}
	}
	return ev, out, err
}

// ValidateCommandVerification enforces the provenance fields whose absence
// makes it impossible to attribute or bind command evidence to an artifact.
func ValidateCommandVerification(ev VerificationEvidence) error {
	var missing []string
	if strings.TrimSpace(ev.Command) == "" {
		missing = append(missing, "command")
	}
	if strings.TrimSpace(ev.ArtifactHash) == "" {
		missing = append(missing, "artifact hash")
	}
	if strings.TrimSpace(ev.Verifier) == "" {
		missing = append(missing, "verifier identity")
	}
	if len(missing) > 0 {
		return fmt.Errorf("verification evidence missing %s", strings.Join(missing, ", "))
	}
	return nil
}

// AppendVerificationEvidence adds one JSON record without replacing legacy
// task content. JSON-lines keeps appends and mechanical migration simple.
func AppendVerificationEvidence(t *Task, ev VerificationEvidence) error {
	if err := ValidateCommandVerification(ev); err != nil {
		return err
	}
	b, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	sec, _ := t.Doc.Section(verificationSection)
	t.Doc.SetSection(verificationSection, sec.Content+string(b)+"\n")
	return nil
}

var legacyVerificationLine = regexp.MustCompile(`(?m)^- .* verified by .+$`)

// VerificationEvidenceRecords reads structured records and also surfaces old
// string-only evidence as Legacy records. It never guesses fields that the old
// rendered sentence did not reliably encode.
func VerificationEvidenceRecords(t *Task) []VerificationEvidence {
	var records []VerificationEvidence
	if sec, ok := t.Doc.Section(verificationSection); ok {
		for _, line := range strings.Split(sec.Content, "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			var ev VerificationEvidence
			if json.Unmarshal([]byte(line), &ev) == nil {
				records = append(records, ev)
			}
		}
	}
	if len(records) > 0 {
		return records
	}
	if log, ok := t.Doc.Section("Log"); ok {
		for _, line := range legacyVerificationLine.FindAllString(log.Content, -1) {
			records = append(records, VerificationEvidence{Legacy: strings.TrimSpace(line)})
		}
	}
	return records
}

// AcceptanceRequiresCommandVerification recognizes the task format's explicit
// inline-code convention: a criterion containing a backticked command asks the
// checker to run and provenance that command, not merely assert its outcome.
func AcceptanceRequiresCommandVerification(t *Task, criterion int) bool {
	boxes := t.Acceptance()
	if criterion < 1 || criterion > len(boxes) {
		return false
	}
	text := boxes[criterion-1].Text
	start := strings.IndexByte(text, '`')
	return start >= 0 && strings.IndexByte(text[start+1:], '`') >= 0
}
