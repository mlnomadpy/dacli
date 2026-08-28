// Package commandresult carries typed command facts across dacli's subprocess
// feature-slice boundary without turning user-facing output into an API.
package commandresult

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const envKey = "DACLI_COMMAND_RESULT"

type Spawn struct {
	RunID string `json:"run_id"`
}

type Integration struct {
	Merged int `json:"merged"`
	Open   int `json:"open"`
}

// Flush serializes result when the parent process requested a command result.
func Flush(result any) error {
	path := os.Getenv(envKey)
	if path == "" || result == nil {
		return nil
	}
	b, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("encode command result: %w", err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		return fmt.Errorf("write command result: %w", err)
	}
	return nil
}

// Capture runs cmd while receiving its typed result through a private file.
// Output is returned unchanged for display and diagnostics.
func Capture(cmd *exec.Cmd, target any) ([]byte, error) {
	f, err := os.CreateTemp("", "dacli-command-result-*")
	if err != nil {
		return nil, err
	}
	path := f.Name()
	if err := f.Close(); err != nil {
		return nil, err
	}
	defer func() { _ = os.Remove(path) }()
	cmd.Env = append(os.Environ(), envKey+"="+path)
	out, runErr := Run(cmd, RunOptions{
		Operation:     filepath.Base(cmd.Path),
		WorkspaceRoot: cmd.Dir,
	})
	b, readErr := os.ReadFile(path)
	if len(b) > 0 {
		if err := json.Unmarshal(b, target); err != nil {
			return out, fmt.Errorf("decode command result: %w", err)
		}
	} else if runErr == nil {
		return out, fmt.Errorf("command returned no structured result")
	}
	if readErr != nil {
		return out, fmt.Errorf("read command result: %w", readErr)
	}
	return out, runErr
}
