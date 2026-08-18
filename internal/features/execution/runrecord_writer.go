package execution

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// runRecord is the single write boundary for execution audit artifacts. Brief,
// invocation, process identity and terminal outcome are critical: callers must
// stop when they cannot be atomically replaced. Enrichment is best-effort, but
// its failure is itself appended to diagnostics.txt so `runs show` cannot hide
// that the durable record is incomplete (issue #688).
type runRecord struct {
	dir    string
	warn   io.Writer
	rename func(string, string) error
}

func openRunRecord(dir string, warn io.Writer) runRecord {
	return runRecord{dir: dir, warn: warn, rename: os.Rename}
}

func (r runRecord) critical(name, content string) error {
	if err := r.replace(name, content); err != nil {
		return fmt.Errorf("record critical run artifact %s: %w", name, err)
	}
	return nil
}

func (r runRecord) bestEffort(name, content string) {
	if err := r.replace(name, content); err != nil {
		msg := fmt.Sprintf("%s: could not record optional %s: %v\n", time.Now().UTC().Format(time.RFC3339), name, err)
		if derr := r.appendDiagnostic(msg); derr != nil && r.warn != nil {
			fmt.Fprintf(r.warn, "warning: %s (diagnostic also failed: %v)\n", strings.TrimSpace(msg), derr)
		}
	}
}

func (r runRecord) replace(name, content string) error {
	tmp, err := os.CreateTemp(r.dir, "."+filepath.Base(name)+"-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	ok := false
	defer func() {
		_ = tmp.Close()
		if !ok {
			_ = os.Remove(tmpName)
		}
	}()
	if err := tmp.Chmod(0o644); err != nil {
		return err
	}
	if _, err := io.WriteString(tmp, content); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := r.rename(tmpName, filepath.Join(r.dir, name)); err != nil {
		return err
	}
	ok = true
	return nil
}

func (r runRecord) appendDiagnostic(msg string) error {
	path := filepath.Join(r.dir, "diagnostics.txt")
	old, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return r.replace("diagnostics.txt", string(old)+msg)
}
