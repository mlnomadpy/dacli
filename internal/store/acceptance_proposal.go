package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/agentid"
	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

const AcceptanceProposalSchema = "acceptance-proposal/v1"

// AcceptanceProposal is the durable, restart-safe handoff between an
// independent read-only reviewer and the task owner. Its ID binds every fact
// used by the decision; State is deliberately excluded from that digest so the
// owner can acknowledge application without changing the proposal identity.
type AcceptanceProposal struct {
	Schema          string               `json:"schema"`
	ID              string               `json:"id"`
	TaskID          string               `json:"task_id"`
	Project         string               `json:"project"`
	Proposer        string               `json:"proposer"`
	Role            string               `json:"role"`
	Grant           string               `json:"grant"`
	Runtime         string               `json:"runtime"`
	RunID           string               `json:"run_id"`
	CommitSHA       string               `json:"commit_sha"`
	TreeSHA         string               `json:"tree_sha"`
	ChecklistDigest string               `json:"checklist_digest"`
	EvidenceDigest  string               `json:"evidence_digest"`
	Evidence        VerificationEvidence `json:"evidence"`
	CreatedAt       time.Time            `json:"created_at"`
	State           string               `json:"state"`
	AppliedBy       string               `json:"applied_by,omitempty"`
	AppliedAt       *time.Time           `json:"applied_at,omitempty"`
}

type acceptanceProposalIdentity struct {
	Schema, TaskID, Project, Proposer, Role, Grant, Runtime, RunID string
	CommitSHA, TreeSHA, ChecklistDigest, EvidenceDigest            string
}

func digestJSON(v any) (string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func AcceptanceChecklistDigest(t *Task) (string, error) {
	type item struct {
		Text string `json:"text"`
		Done bool   `json:"done"`
	}
	items := make([]item, 0, len(t.Acceptance()))
	for _, criterion := range t.Acceptance() {
		items = append(items, item{Text: criterion.Text, Done: criterion.Done})
	}
	return digestJSON(items)
}

func NewAcceptanceProposal(w *workspace.Workspace, t *Task, id *agentid.Identity, evidence VerificationEvidence, now time.Time) (AcceptanceProposal, error) {
	if id.Grant != "ro" {
		return AcceptanceProposal{}, fmt.Errorf("independent acceptance proposals require a read-only reviewer grant")
	}
	if ClaimedBy(t) == id.ID || t.Owner() == id.ID {
		return AcceptanceProposal{}, fmt.Errorf("reviewer %s is not independent of task %03d", id.ID, t.Seq)
	}
	if err := ValidateFinalTreeVerification(evidence, evidence.CommitSHA, evidence.TreeSHA); err != nil {
		return AcceptanceProposal{}, err
	}
	if evidence.ExitCode != 0 || evidence.Verifier != id.ID {
		return AcceptanceProposal{}, fmt.Errorf("verification evidence must pass and belong to proposer %s", id.ID)
	}
	runID, runtime, err := acceptanceProposerRuntime(w, t.ID, id.ID)
	if err != nil {
		return AcceptanceProposal{}, err
	}
	checklist, err := AcceptanceChecklistDigest(t)
	if err != nil {
		return AcceptanceProposal{}, err
	}
	evidenceDigest, err := digestJSON(evidence)
	if err != nil {
		return AcceptanceProposal{}, err
	}
	p := AcceptanceProposal{Schema: AcceptanceProposalSchema, TaskID: t.ID, Project: t.Project, Proposer: id.ID, Role: id.Role,
		Grant: string(id.Grant), Runtime: runtime, RunID: runID, CommitSHA: evidence.CommitSHA, TreeSHA: evidence.TreeSHA,
		ChecklistDigest: checklist, EvidenceDigest: evidenceDigest, Evidence: evidence, CreatedAt: now.UTC(), State: "pending"}
	p.ID, err = p.identityDigest()
	if err != nil {
		return AcceptanceProposal{}, err
	}
	p.ID = "ap-" + strings.TrimPrefix(p.ID, "sha256:")[:24]
	return p, nil
}

func (p AcceptanceProposal) identityDigest() (string, error) {
	return digestJSON(acceptanceProposalIdentity{p.Schema, p.TaskID, p.Project, p.Proposer, p.Role, p.Grant, p.Runtime, p.RunID,
		p.CommitSHA, p.TreeSHA, p.ChecklistDigest, p.EvidenceDigest})
}

func acceptanceProposerRuntime(w *workspace.Workspace, taskID, proposer string) (string, string, error) {
	entries, err := os.ReadDir(w.RunsDir())
	if err != nil {
		return "", "", fmt.Errorf("read reviewer runs: %w", err)
	}
	type candidate struct {
		run, runtime string
		started      time.Time
	}
	var candidates []candidate
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		rec, readErr := procmon.ReadRecord(filepath.Join(w.RunDir(entry.Name()), "proc.txt"))
		if readErr == nil && rec.Child == proposer && rec.Task == taskID && rec.Runtime != "" {
			candidates = append(candidates, candidate{rec.RunID, rec.Runtime, rec.Started})
		}
	}
	if len(candidates) == 0 {
		return "", "", fmt.Errorf("no dacli-spawned runtime record binds reviewer %s to task %s; spawn the reviewer for this task and retry", proposer, taskID)
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].started.After(candidates[j].started) })
	return candidates[0].run, candidates[0].runtime, nil
}

func AcceptanceProposalPath(w *workspace.Workspace, p AcceptanceProposal) string {
	return filepath.Join(w.Root, workspace.Dir, "acceptance-proposals", p.Project, p.TaskID, p.ID+".json")
}

func WriteAcceptanceProposal(w *workspace.Workspace, p AcceptanceProposal) error {
	if !workspace.SafeSegment(p.Project) || !workspace.SafeSegment(p.TaskID) || !workspace.SafeSegment(p.ID) {
		return fmt.Errorf("unsafe acceptance proposal identity")
	}
	raw, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	path := AcceptanceProposalPath(w, p)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return atomicWrite(path, append(raw, '\n'))
}

func ReadAcceptanceProposal(w *workspace.Workspace, project, taskID, proposalID string) (AcceptanceProposal, error) {
	p := AcceptanceProposal{Project: project, TaskID: taskID, ID: proposalID}
	raw, err := os.ReadFile(AcceptanceProposalPath(w, p))
	if err != nil {
		return p, err
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return p, err
	}
	if err := ValidateAcceptanceProposal(p); err != nil {
		return p, err
	}
	if p.Project != project || p.TaskID != taskID || p.ID != proposalID {
		return p, fmt.Errorf("acceptance proposal path identity does not match its content")
	}
	return p, nil
}

func ValidateAcceptanceProposal(p AcceptanceProposal) error {
	if p.Schema != AcceptanceProposalSchema || p.ID == "" || p.TaskID == "" || p.Project == "" || p.Proposer == "" || p.Role == "" || p.Grant != "ro" || p.Runtime == "" || p.RunID == "" {
		return fmt.Errorf("invalid %s identity", AcceptanceProposalSchema)
	}
	if p.State != "pending" && p.State != "applied" {
		return fmt.Errorf("invalid acceptance proposal state %q", p.State)
	}
	evidenceDigest, err := digestJSON(p.Evidence)
	if err != nil || evidenceDigest != p.EvidenceDigest {
		return fmt.Errorf("acceptance proposal evidence digest mismatch")
	}
	if p.Evidence.ExitCode != 0 || p.Evidence.Verifier != p.Proposer || p.Evidence.CommitSHA != p.CommitSHA || p.Evidence.TreeSHA != p.TreeSHA || !p.Evidence.Clean {
		return fmt.Errorf("acceptance proposal carries invalid verification evidence")
	}
	digest, err := p.identityDigest()
	if err != nil {
		return err
	}
	want := "ap-" + strings.TrimPrefix(digest, "sha256:")[:24]
	if p.ID != want {
		return fmt.Errorf("acceptance proposal identity digest mismatch")
	}
	return nil
}

func PendingAcceptanceProposals(w *workspace.Workspace, t *Task) ([]AcceptanceProposal, error) {
	dir := filepath.Dir(AcceptanceProposalPath(w, AcceptanceProposal{Project: t.Project, TaskID: t.ID, ID: "placeholder"}))
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return []AcceptanceProposal{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := make([]AcceptanceProposal, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		p, readErr := ReadAcceptanceProposal(w, t.Project, t.ID, strings.TrimSuffix(entry.Name(), ".json"))
		if readErr != nil {
			return nil, readErr
		}
		if p.State == "pending" {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func MarkAcceptanceProposalApplied(w *workspace.Workspace, p AcceptanceProposal, actor string, now time.Time) error {
	p.State, p.AppliedBy = "applied", actor
	at := now.UTC()
	p.AppliedAt = &at
	return WriteAcceptanceProposal(w, p)
}
