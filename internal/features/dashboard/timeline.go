package dashboard

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mlnomadpy/dacli/internal/procmon"
	"github.com/mlnomadpy/dacli/internal/store"
	"github.com/mlnomadpy/dacli/internal/workspace"
)

const deliveryTimelineSchema = "delivery-attempt-timeline/v1"

var deliveryPhaseOrder = []string{
	"selected", "spawned", "acting", "verified", "reviewed", "pushed", "pr", "ci", "merged", "accepted",
}

// deliveryTimelineResponse is a public-safe, read-only chronology. It carries
// identities and conclusions, never transcript/prompt contents, local paths,
// review findings, or provider reasoning.
type deliveryTimelineResponse struct {
	Schema    string                `json:"schema"`
	Generated string                `json:"generated"`
	Task      deliveryTimelineTask  `json:"task"`
	Attempts  []deliveryAttemptView `json:"attempts"`
	Summary   string                `json:"summary"`
	Refusal   string                `json:"refusal,omitempty"`
}

type deliveryTimelineTask struct {
	ID         string `json:"id"`
	Sequence   int    `json:"sequence"`
	Generation int    `json:"generation"`
	Project    string `json:"project"`
	Title      string `json:"title"`
	Status     string `json:"status"`
}

type deliveryAttemptView struct {
	Attempt    int                `json:"attempt"`
	RunID      string             `json:"run_id"`
	AgentID    string             `json:"agent_id"`
	Role       string             `json:"role"`
	Runtime    string             `json:"runtime"`
	Model      string             `json:"model"`
	Generation int                `json:"generation"`
	Started    string             `json:"started"`
	Outcome    string             `json:"outcome"`
	Recovered  bool               `json:"recovered"`
	Usage      deliveryUsageView  `json:"usage"`
	Identity   deliveryIdentity   `json:"identity"`
	Spans      []deliverySpanView `json:"spans"`
}

type deliveryUsageView struct {
	Available    bool    `json:"available"`
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	Turns        int     `json:"turns"`
	CostUSD      float64 `json:"cost_usd"`
}

type deliveryIdentity struct {
	TaskID       string `json:"task_id"`
	RunID        string `json:"run_id"`
	CommitSHA    string `json:"commit_sha"`
	TreeSHA      string `json:"tree_sha"`
	PRURL        string `json:"pr_url"`
	PRGeneration int    `json:"pr_generation"`
}

type deliverySpanView struct {
	Phase      string `json:"phase"`
	Status     string `json:"status"` // complete | current | pending | skipped | refused | unknown
	Started    string `json:"started,omitempty"`
	Ended      string `json:"ended,omitempty"`
	DurationMS *int64 `json:"duration_ms"`
	Source     string `json:"source"`
	Freshness  string `json:"freshness"`
	Detail     string `json:"detail"`
	NextAction string `json:"next_action"`
	Contract   string `json:"contract,omitempty"`
	Verdict    string `json:"verdict,omitempty"`
	Correction int    `json:"correction,omitempty"`
}

type phaseJournalProjection struct {
	Schema  string `json:"schema"`
	Project string `json:"project"`
	Tasks   []struct {
		TaskID     string    `json:"task_id"`
		Sequence   int       `json:"sequence"`
		Generation int       `json:"generation"`
		RunID      string    `json:"run_id"`
		Phase      string    `json:"phase"`
		UpdatedAt  time.Time `json:"updated_at"`
	} `json:"tasks"`
}

func buildDeliveryTimeline(w *workspace.Workspace, ref string) (deliveryTimelineResponse, error) {
	resp := deliveryTimelineResponse{Schema: deliveryTimelineSchema, Generated: nowStamp(), Attempts: []deliveryAttemptView{}}
	idx, err := store.BuildTaskIndex(w)
	if err != nil {
		return resp, err
	}
	task, err := idx.Find(ref)
	if err != nil {
		return resp, taskError(ref, err)
	}
	resp.Task = deliveryTimelineTask{ID: task.ID, Sequence: task.Seq, Generation: task.Generation(), Project: task.Project, Title: task.Title, Status: string(task.Status)}

	checkpoint, journalErr := readTimelineCheckpoint(w, task)
	if journalErr != nil {
		resp.Refusal = journalErr.Error()
	}

	entries, readErr := os.ReadDir(w.RunsDir())
	if readErr != nil && !os.IsNotExist(readErr) {
		return resp, readErr
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		rec, recErr := procmon.ReadRecord(filepath.Join(w.RunDir(entry.Name()), "proc.txt"))
		if recErr != nil || rec.Task != task.ID || rec.RunID != entry.Name() {
			continue
		}
		resp.Attempts = append(resp.Attempts, buildDeliveryAttempt(w, task, rec, checkpoint, journalErr))
	}
	sort.Slice(resp.Attempts, func(i, j int) bool { return resp.Attempts[i].Started < resp.Attempts[j].Started })
	for i := range resp.Attempts {
		resp.Attempts[i].Attempt = i + 1
	}
	if len(resp.Attempts) == 0 {
		resp.Summary = "No coding-agent delivery attempt has been recorded for this task."
	} else {
		latest := resp.Attempts[len(resp.Attempts)-1]
		resp.Summary = fmt.Sprintf("Attempt %d is %s at %s; %d total attempt(s) remain independently bound to their run evidence.", latest.Attempt, latest.Outcome, currentDeliveryPhase(latest.Spans), len(resp.Attempts))
	}
	return resp, nil
}

func readTimelineCheckpoint(w *workspace.Workspace, task *store.Task) (*struct {
	RunID, Phase string
	UpdatedAt    time.Time
}, error) {
	raw, err := os.ReadFile(filepath.Join(w.Root, workspace.Dir, "loop", task.Project+"-phases.json"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("phase journal is unreadable; timeline refuses to infer progress")
	}
	var journal phaseJournalProjection
	if json.Unmarshal(raw, &journal) != nil || journal.Schema != "loop-phase-journal/v1" || journal.Project != task.Project {
		return nil, fmt.Errorf("phase journal is malformed or belongs to another project; timeline refuses to infer progress")
	}
	for _, candidate := range journal.Tasks {
		if candidate.TaskID == task.ID && candidate.Sequence == task.Seq && candidate.Generation == task.Generation() {
			if phaseRank(candidate.Phase) < 0 || candidate.UpdatedAt.IsZero() {
				return nil, fmt.Errorf("current phase checkpoint is corrupt; timeline refuses to render it as success")
			}
			return &struct {
				RunID, Phase string
				UpdatedAt    time.Time
			}{candidate.RunID, candidate.Phase, candidate.UpdatedAt.UTC()}, nil
		}
	}
	return nil, nil
}

func buildDeliveryAttempt(w *workspace.Workspace, task *store.Task, rec procmon.Record, checkpoint *struct {
	RunID, Phase string
	UpdatedAt    time.Time
}, journalErr error) deliveryAttemptView {
	model := ""
	if role, ok := store.LoadRole(w, rec.Role); ok {
		model = role.Model
	}
	isCurrentAttempt := checkpoint != nil && checkpoint.RunID == rec.RunID
	u, ok := readTimelineUsage(w.RunDir(rec.RunID))
	usage := u
	usage.Available = ok
	generation := 0
	identity := deliveryIdentity{TaskID: task.ID, RunID: rec.RunID}
	if isCurrentAttempt {
		generation = task.Generation()
		identity.PRURL, identity.PRGeneration = store.RecordedPRURL(task), task.Generation()
	}
	evidence := store.VerificationEvidenceRecords(task)
	if isCurrentAttempt && len(evidence) > 0 {
		latest := evidence[len(evidence)-1]
		identity.CommitSHA, identity.TreeSHA = latest.CommitSHA, latest.TreeSHA
	}
	current := "acting"
	observedAt := time.Time{}
	if isCurrentAttempt {
		current, observedAt = normalizeDeliveryPhase(checkpoint.Phase), checkpoint.UpdatedAt
	}
	recovered := isCurrentAttempt && rec.Outcome != "" && phaseRank(checkpoint.Phase) >= phaseRank("pr-created")
	outcome := rec.Outcome
	if outcome == "" {
		outcome = "running"
	}
	view := deliveryAttemptView{RunID: rec.RunID, AgentID: rec.Child, Role: rec.Role, Runtime: rec.Runtime, Model: model, Generation: generation, Started: rec.Started.UTC().Format(time.RFC3339Nano), Outcome: outcome, Recovered: recovered, Usage: usage, Identity: identity, Spans: []deliverySpanView{}}
	for _, phase := range deliveryPhaseOrder {
		status := "pending"
		if journalErr != nil && phaseRank(phase) > phaseRank("acting") {
			status = "refused"
		} else if !isCurrentAttempt && rec.Outcome != "" {
			switch {
			case phaseRank(phase) <= phaseRank("acting"):
				status = "complete"
			case phase == "verified":
				status = "unknown"
			}
		} else if phaseRank(phase) < phaseRank(current) {
			status = "complete"
		} else if phase == current {
			status = "current"
		}
		if rec.Outcome != "" && status == "current" && phase == "verified" && !successfulRunOutcome(rec.Outcome) {
			status = "refused"
		}
		span := deliverySpanView{Phase: phase, Status: status, Source: phaseSource(phase), Freshness: "current task generation", Detail: phaseDetail(phase, status), NextAction: phaseNextAction(phase, status)}
		if phase == "spawned" || phase == "acting" {
			span.Started = rec.Started.UTC().Format(time.RFC3339Nano)
		}
		if !observedAt.IsZero() && phaseRank(phase) <= phaseRank(current) {
			span.Ended = observedAt.Format(time.RFC3339Nano)
			if !rec.Started.IsZero() && observedAt.After(rec.Started) && (phase == "acting" || phase == current) {
				ms := observedAt.Sub(rec.Started).Milliseconds()
				span.DurationMS = &ms
			}
		}
		if phase == "verified" && isCurrentAttempt && len(evidence) > 0 {
			latest := evidence[len(evidence)-1]
			span.Contract = strings.Join(latest.Argv, " ")
		}
		view.Spans = append(view.Spans, span)
	}
	return view
}

func successfulRunOutcome(outcome string) bool {
	switch strings.ToLower(strings.TrimSpace(outcome)) {
	case "ok", "success", "succeeded", "complete", "completed", "passed", "exit 0":
		return true
	default:
		return false
	}
}

func readTimelineUsage(runDir string) (deliveryUsageView, bool) {
	f, err := os.Open(filepath.Join(runDir, "usage.txt"))
	if err != nil {
		return deliveryUsageView{}, false
	}
	defer func() { _ = f.Close() }()
	var u deliveryUsageView
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		key, raw, found := strings.Cut(scanner.Text(), ":")
		if !found {
			continue
		}
		value := strings.TrimSpace(raw)
		switch strings.TrimSpace(key) {
		case "input_tokens":
			u.InputTokens, _ = strconv.Atoi(value)
		case "output_tokens":
			u.OutputTokens, _ = strconv.Atoi(value)
		case "num_turns":
			u.Turns, _ = strconv.Atoi(value)
		case "cost_usd":
			u.CostUSD, _ = strconv.ParseFloat(value, 64)
		}
	}
	u.Available = u.InputTokens > 0 || u.OutputTokens > 0 || u.Turns > 0 || u.CostUSD > 0
	return u, u.Available
}

func normalizeDeliveryPhase(phase string) string {
	switch phase {
	case "spawned", "waited":
		return "acting"
	case "verified":
		return "verified"
	case "review-pending", "correction-pending", "re-review-pending", "reviewed":
		return "reviewed"
	case "pushed":
		return "pushed"
	case "pr-created":
		return "pr"
	case "ci-pending":
		return "ci"
	case "merged":
		return "merged"
	case "record-accepted":
		return "accepted"
	default:
		return "unknown"
	}
}

func phaseRank(phase string) int {
	normalized := normalizeDeliveryPhase(phase)
	if normalized == "unknown" {
		normalized = phase
	}
	for i, candidate := range deliveryPhaseOrder {
		if candidate == normalized {
			return i
		}
	}
	return -1
}

func currentDeliveryPhase(spans []deliverySpanView) string {
	for _, span := range spans {
		if span.Status == "current" || span.Status == "refused" {
			return span.Phase
		}
	}
	return "unknown"
}

func phaseSource(phase string) string {
	if phase == "selected" {
		return "task record"
	}
	if phase == "spawned" || phase == "acting" {
		return "run/proc.txt"
	}
	if phase == "verified" {
		return "task verification evidence + loop phase journal"
	}
	return "loop phase journal"
}

func phaseDetail(phase, status string) string {
	if status == "refused" {
		return "Evidence is missing, stale, failed, or corrupt; this phase is not green."
	}
	if status == "pending" {
		return "No current-generation completion evidence has been observed."
	}
	return phase + " evidence is bound to this attempt."
}

func phaseNextAction(phase, status string) string {
	if status == "complete" {
		return "inspect the next phase"
	}
	if status == "refused" {
		return "repair or re-observe the named evidence before continuing"
	}
	if status == "pending" {
		return "wait for or run this governed phase"
	}
	return "observe this phase until it reaches a durable boundary"
}
