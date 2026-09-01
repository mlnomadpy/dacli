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
	"github.com/mlnomadpy/dacli/internal/publication"
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
	Attempt      int                `json:"attempt"`
	RunID        string             `json:"run_id"`
	AgentID      string             `json:"agent_id"`
	Role         string             `json:"role"`
	Runtime      string             `json:"runtime"`
	Model        string             `json:"model"`
	Generation   int                `json:"generation"`
	Started      string             `json:"started"`
	Outcome      string             `json:"outcome"`
	Recovered    bool               `json:"recovered"`
	Usage        deliveryUsageView  `json:"usage"`
	Identity     deliveryIdentity   `json:"identity"`
	PullRequests []deliveryPRView   `json:"pull_requests,omitempty"`
	Spans        []deliverySpanView `json:"spans"`
}

type deliveryPRView struct {
	URL        string `json:"url"`
	Generation int    `json:"generation"`
	State      string `json:"state"`
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
	var pullRequests []deliveryPRView
	if isCurrentAttempt {
		generation = task.Generation()
		pullRequests = deliveryPullRequests(task)
		if len(pullRequests) > 0 {
			identity.PRURL, identity.PRGeneration = pullRequests[len(pullRequests)-1].URL, pullRequests[len(pullRequests)-1].Generation
		}
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
	view := deliveryAttemptView{RunID: rec.RunID, AgentID: rec.Child, Role: rec.Role, Runtime: rec.Runtime, Model: model, Generation: generation, Started: rec.Started.UTC().Format(time.RFC3339Nano), Outcome: outcome, Recovered: recovered, Usage: usage, Identity: identity, PullRequests: pullRequests, Spans: []deliverySpanView{}}
	checkpointBeforeRun := isCurrentAttempt && !rec.Started.IsZero() && observedAt.Before(rec.Started)
	for _, phase := range deliveryPhaseOrder {
		status := "pending"
		if (journalErr != nil || checkpointBeforeRun) && phaseRank(phase) > phaseRank("acting") {
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
	if isCurrentAttempt {
		enrichReviewSpan(w, task, &view)
		enrichCISpan(evidence, &view)
		enrichPRSpan(&view)
		enrichHandoffSpan(w, task, rec, &view)
		enrichAcceptanceSpan(task, checkpoint.Phase, &view)
		if checkpointBeforeRun {
			refuseSpan(&view, "verified", "phase checkpoint predates this run; chronology is corrupt", "rebuild the phase journal from durable run evidence")
		}
	}
	return view
}

func deliveryPullRequests(task *store.Task) []deliveryPRView {
	sec, ok := task.Doc.Section("Log")
	if !ok {
		return nil
	}
	var urls []string
	seen := map[string]bool{}
	for _, field := range strings.Fields(sec.Content) {
		url := strings.TrimRight(field, ").,;]")
		if strings.HasPrefix(url, "https://") && strings.Contains(url, "/pull/") && !seen[url] {
			seen[url] = true
			urls = append(urls, url)
		}
	}
	out := make([]deliveryPRView, 0, len(urls))
	for i, url := range urls {
		state := "superseded"
		if i == len(urls)-1 {
			state = "current"
		}
		generation := task.Generation() - (len(urls) - 1 - i)
		if generation < 1 {
			generation = 0 // unknown; never invent a historical generation
		}
		out = append(out, deliveryPRView{URL: safeDashboardText(url), Generation: generation, State: state})
	}
	return out
}

func enrichReviewSpan(w *workspace.Workspace, task *store.Task, view *deliveryAttemptView) {
	tx, err := store.ReadReviewTransaction(w, task.Project, task.ID)
	if os.IsNotExist(err) {
		return
	}
	span := spanFor(view, "reviewed")
	if span == nil {
		return
	}
	span.Source = "independent review transaction"
	if err != nil {
		span.Status, span.Detail, span.NextAction = "refused", "Review transaction is unreadable or invalid; no verdict is trusted.", "repair or re-run the independent review"
		return
	}
	span.Correction = tx.CorrectionTurns
	span.Ended = tx.UpdatedAt.UTC().Format(time.RFC3339Nano)
	requiresExactTree := tx.State == store.ReviewApproved || tx.State == store.ReviewCorrection || tx.State == store.ReviewAwaitingRereview
	started, _ := time.Parse(time.RFC3339Nano, view.Started)
	if tx.UpdatedAt.IsZero() || !started.IsZero() && tx.UpdatedAt.Before(started) {
		span.Status, span.Verdict = "refused", "invalid-chronology"
		span.Detail = "Review observation time is missing or predates this attempt."
		span.NextAction = "re-run independent review and record a fresh exact-tree observation"
		return
	}
	if requiresExactTree && (view.Identity.CommitSHA == "" || view.Identity.TreeSHA == "" || tx.CurrentCommit == "" || tx.CurrentTree == "" || tx.CurrentCommit != view.Identity.CommitSHA || tx.CurrentTree != view.Identity.TreeSHA) {
		span.Status, span.Verdict = "refused", "stale-tree"
		span.Detail = "Review is missing exact identity or observed a different commit/tree than current verification evidence."
		span.NextAction = "re-run independent review on the exact verified tree"
		return
	}
	switch tx.State {
	case store.ReviewApproved:
		span.Status, span.Verdict, span.Detail = "complete", "approve", "Independent read-only review approved the exact current tree."
	case store.ReviewCorrection:
		span.Status, span.Verdict, span.Detail = "current", "request-changes", "Independent review requested a bounded correction."
		span.NextAction = "apply the recorded correction, then obtain an exact-tree re-review"
	case store.ReviewAwaitingRereview:
		span.Status, span.Verdict, span.Detail = "current", "awaiting-re-review", "Correction produced a new tree that still requires independent re-review."
		span.NextAction = "run the independent reviewer against the corrected tree"
	case store.ReviewHalted:
		span.Status, span.Verdict, span.Detail = "refused", "halted", "Independent review halted without an approval."
		span.NextAction = "inspect the local review transaction and resolve its typed refusal"
	default:
		span.Status, span.Verdict, span.Detail = "current", "awaiting-review", "The exact tree is awaiting independent review."
		span.NextAction = "run the configured independent reviewer"
	}
}

func enrichCISpan(evidence []store.VerificationEvidence, view *deliveryAttemptView) {
	if len(evidence) == 0 {
		return
	}
	latest := evidence[len(evidence)-1]
	if len(latest.External) == 0 {
		return
	}
	span := spanFor(view, "ci")
	if span == nil {
		return
	}
	span.Source = "typed external verification evidence"
	span.Freshness = "exact verified head"
	var green, pending []string
	for _, check := range latest.External {
		name := strings.TrimSpace(check.Name)
		if name == "" {
			name = strings.TrimSpace(check.Provider)
		}
		name = safeDashboardText(name)
		if check.HeadSHA == "" || view.Identity.CommitSHA == "" || check.HeadSHA != view.Identity.CommitSHA {
			span.Status, span.Verdict = "refused", "stale-head"
			span.Detail = "External verification is not bound to the current verified commit."
			span.NextAction = "re-observe required checks on the exact current head"
			return
		}
		switch {
		case check.State == "observed" && strings.EqualFold(check.Conclusion, "success"):
			if check.ObservedAt.IsZero() {
				span.Status, span.Verdict = "refused", "missing-observation-time"
				span.Detail = "Successful external verification is missing its observation time."
				span.NextAction = "re-observe required checks on the exact current head"
				return
			}
			green = append(green, name)
		case check.State == "pending":
			pending = append(pending, name)
		case check.State == "observed":
			span.Status, span.Verdict = "refused", "check-failed"
			span.Detail = fmt.Sprintf("Required check %s concluded %s.", name, check.Conclusion)
			span.NextAction = "inspect the check diagnosis, repair the failure, and re-observe the same head"
			return
		default:
			span.Status, span.Verdict = "refused", "check-"+check.State
			span.Detail = fmt.Sprintf("Required check %s is %s; absence is not green.", name, check.State)
			span.NextAction = "restore an observable required check on the exact head"
			return
		}
	}
	if len(pending) > 0 {
		span.Status, span.Verdict = "current", "checks-pending"
		span.Detail = "Waiting for required checks: " + strings.Join(pending, ", ") + "."
		span.NextAction = "wait for the named checks, then re-observe the exact head"
		return
	}
	span.Status, span.Verdict = "complete", "checks-green"
	span.Detail = "Required checks passed on the exact head: " + strings.Join(green, ", ") + "."
	span.NextAction = "continue to the merge gate"
}

func enrichPRSpan(view *deliveryAttemptView) {
	if len(view.PullRequests) == 0 {
		return
	}
	span := spanFor(view, "pr")
	if span == nil {
		return
	}
	superseded := len(view.PullRequests) - 1
	if superseded > 0 {
		span.Detail = fmt.Sprintf("Current PR generation %d is canonical; %d older PR generation(s) remain superseded.", view.Identity.PRGeneration, superseded)
	}
}

func enrichHandoffSpan(w *workspace.Workspace, task *store.Task, rec procmon.Record, view *deliveryAttemptView) {
	handoff, err := store.LoadRootHandoff(w, rec.RunID)
	if os.IsNotExist(err) {
		return
	}
	if err != nil || handoff.TaskID != task.ID || handoff.ChildID != rec.Child {
		refuseSpan(view, "acting", "Owner handoff evidence is unreadable or bound to another identity.", "re-capture an exact root handoff for this run")
		return
	}
	consumedPath := filepath.Join(w.RunDir(rec.RunID), store.RootHandoffConsumedFile)
	if _, err := os.Stat(consumedPath); err == nil {
		view.Recovered = true
		return
	}
	span := spanFor(view, "acting")
	if span == nil {
		return
	}
	span.Status, span.Source, span.Verdict = "refused", "root-handoff/v1", handoff.FailureClass
	span.Detail = "Worker stopped at owner handoff after " + safeDashboardText(handoff.FailedOperation) + "."
	span.NextAction = safeDashboardText(handoff.NextAction)
	span.Ended = handoff.CreatedAt.UTC().Format(time.RFC3339Nano)
}

func enrichAcceptanceSpan(task *store.Task, phase string, view *deliveryAttemptView) {
	accepted := spanFor(view, "accepted")
	merged := spanFor(view, "merged")
	if accepted == nil || merged == nil {
		return
	}
	if phaseRank(phase) >= phaseRank("merged") {
		merged.Status = "complete"
		if task.Status == "done" || normalizeDeliveryPhase(phase) == "accepted" {
			accepted.Status, accepted.Detail = "complete", "Acceptance is durably recorded for this task generation."
			return
		}
		accepted.Status = "current"
		accepted.Detail = "The PR is merged, but task acceptance has not been recorded."
		accepted.NextAction = "inspect fresh trunk, verify the exact merged head, then record acceptance"
	}
}

func spanFor(view *deliveryAttemptView, phase string) *deliverySpanView {
	for i := range view.Spans {
		if view.Spans[i].Phase == phase {
			return &view.Spans[i]
		}
	}
	return nil
}

func refuseSpan(view *deliveryAttemptView, phase, detail, next string) {
	if span := spanFor(view, phase); span != nil {
		span.Status, span.Detail, span.NextAction = "refused", detail, next
	}
}

func safeDashboardText(value string) string {
	return publication.New("", "unknown", false, false, false).Sanitize(value)
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
	for _, span := range spans {
		if span.Status == "pending" {
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
