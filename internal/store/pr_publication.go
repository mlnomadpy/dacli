package store

// PR branch publication is a restartable transaction. The checkpoint binds
// the task's canonical branch, exact local/remote object IDs, landing base,
// and eventual PR URL so a retry can re-observe instead of pushing whichever
// branch the operator happens to have checked out.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mlnomadpy/dacli/internal/mdstore"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

const PRPublicationSchema = "pr-publication/v1"

type PRPublication struct {
	Schema    string    `json:"schema"`
	TaskID    string    `json:"task_id"`
	Branch    string    `json:"branch"`
	Base      string    `json:"base"`
	LocalOID  string    `json:"local_oid"`
	RemoteOID string    `json:"remote_oid,omitempty"`
	PRURL     string    `json:"pr_url,omitempty"`
	Stage     string    `json:"stage"`
	UpdatedAt time.Time `json:"updated_at"`
}

func PRPublicationPath(w *workspace.Workspace, taskID string) string {
	return filepath.Join(w.RunsDir(), "pr-publications", taskID+".json")
}

func LoadPRPublication(w *workspace.Workspace, taskID string) (PRPublication, error) {
	var checkpoint PRPublication
	raw, err := os.ReadFile(PRPublicationPath(w, taskID))
	if err != nil {
		return checkpoint, err
	}
	if err := json.Unmarshal(raw, &checkpoint); err != nil {
		return checkpoint, err
	}
	if err := validatePRPublication(checkpoint); err != nil || checkpoint.TaskID != taskID {
		return PRPublication{}, fmt.Errorf("invalid %s checkpoint", PRPublicationSchema)
	}
	return checkpoint, nil
}

func SavePRPublication(w *workspace.Workspace, checkpoint PRPublication) error {
	if err := validatePRPublication(checkpoint); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(checkpoint, "", "  ")
	if err != nil {
		return err
	}
	path := PRPublicationPath(w, checkpoint.TaskID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return mdstore.WriteBytes(path, append(raw, '\n'), 0o600)
}

func validatePRPublication(checkpoint PRPublication) error {
	if checkpoint.Schema != PRPublicationSchema || checkpoint.TaskID == "" || checkpoint.Branch == "" || checkpoint.Base == "" || checkpoint.LocalOID == "" || checkpoint.UpdatedAt.IsZero() {
		return fmt.Errorf("invalid %s checkpoint", PRPublicationSchema)
	}
	switch checkpoint.Stage {
	case "observed":
		if checkpoint.RemoteOID != "" || checkpoint.PRURL != "" {
			return fmt.Errorf("invalid %s observed checkpoint", PRPublicationSchema)
		}
	case "pushed":
		if checkpoint.RemoteOID == "" || checkpoint.PRURL != "" {
			return fmt.Errorf("invalid %s pushed checkpoint", PRPublicationSchema)
		}
	case "pr-recorded":
		if checkpoint.RemoteOID == "" || checkpoint.PRURL == "" {
			return fmt.Errorf("invalid %s PR checkpoint", PRPublicationSchema)
		}
	default:
		return fmt.Errorf("invalid %s stage %q", PRPublicationSchema, checkpoint.Stage)
	}
	return nil
}
